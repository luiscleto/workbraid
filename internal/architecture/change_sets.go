package architecture

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"go.yaml.in/yaml/v4"
)

const (
	activeChangeSetPrefix  = "refs/workbraid/change-sets/active/"
	appliedChangeSetPrefix = "refs/workbraid/change-sets/applied/"
)

// ChangeSet is one validated durable proposal record. Its concrete changes
// are the only authored Architecture state; Candidate is always rebuilt by
// Manager.ConstructCandidate.
type ChangeSet struct {
	ID              string
	Name            string
	Lifecycle       string
	BaseRevision    string
	Generation      uint64
	Proposal        string
	AppliedRevision string
	RefObject       string
	BaseSnapshot    Snapshot
	Changes         []ComponentChange
	Composition     CandidateComposition
	Candidate       *Candidate
	Review          *ChangeSetReview
	ValidationError error
}

type ChangeSetReview struct {
	BaseRevision  string
	CandidateTree string
	Generation    uint64
}

type UnavailableChangeSet struct {
	ID        string
	Name      string
	Lifecycle string
	Ref       string
	Reason    string
}

type changeSetMetadata struct {
	Format          string           `yaml:"format"`
	Version         int              `yaml:"version"`
	ID              string           `yaml:"id"`
	Name            string           `yaml:"name"`
	BaseRevision    string           `yaml:"base_revision"`
	Generation      uint64           `yaml:"generation"`
	Review          *changeSetReview `yaml:"review,omitempty"`
	AppliedRevision string           `yaml:"applied_revision,omitempty"`
}

type changeSetReview struct {
	BaseRevision  string `yaml:"base_revision"`
	CandidateTree string `yaml:"candidate_tree"`
	Generation    uint64 `yaml:"generation"`
}

type changeState struct {
	DetailReassignments []DetailReassignment        `yaml:"detail_reassignments"`
	Format              string                      `yaml:"format"`
	Version             int                         `yaml:"version"`
	Components          []changeStateComponent      `yaml:"components"`
	NewComponentHomes   []NewComponentHome          `yaml:"new_component_homes"`
	DetailDiagrams      []DetailDiagramChange       `yaml:"detail_diagrams"`
	DiagramTitles       []DiagramTitleChange        `yaml:"diagram_titles"`
	HomeMoves           []ComponentHomeMove         `yaml:"home_moves"`
	References          []ReferenceAppearanceChange `yaml:"references"`
}

type changeStateComponent struct {
	ID                   string                    `yaml:"id"`
	Path                 string                    `yaml:"path"`
	New                  bool                      `yaml:"new"`
	Title                string                    `yaml:"title"`
	Description          string                    `yaml:"description"`
	TitleChanged         bool                      `yaml:"title_changed"`
	DescriptionChanged   bool                      `yaml:"description_changed"`
	RelationshipsChanged bool                      `yaml:"relationships_changed"`
	Relationships        []changeStateRelationship `yaml:"relationships"`
}

type changeStateRelationship struct {
	Target string `yaml:"target"`
	Label  string `yaml:"label"`
}

func (manager *Manager) NewChangeSet(existing []ChangeSet, requestedName string) (string, string, error) {
	id := uuid.NewString()
	name := strings.TrimSpace(requestedName)
	if requestedName != "" && name == "" {
		return "", "", errors.New("change-set name is required")
	}
	if strings.ContainsAny(name, "\r\n") {
		return "", "", errors.New("change-set name must fit on one line")
	}
	used := make([]string, 0, len(existing))
	for _, record := range existing {
		if record.Lifecycle == "active" {
			used = append(used, record.Name)
		}
	}
	if name != "" {
		for _, current := range used {
			if strings.EqualFold(current, name) {
				return "", "", errors.New("change-set name is already active")
			}
		}
		return id, name, nil
	}
	adjectives := [...]string{"calm", "clear", "gentle", "quiet", "steady", "bright", "open", "swift"}
	nouns := [...]string{"harbor", "bridge", "meadow", "signal", "river", "lantern", "grove", "compass"}
	parsed := uuid.MustParse(id)
	base := adjectives[int(parsed[0])%len(adjectives)] + "-" + nouns[int(parsed[1])%len(nouns)]
	name = base
	for suffix := 2; ; suffix++ {
		collision := false
		for _, current := range used {
			if strings.EqualFold(current, name) {
				collision = true
				break
			}
		}
		if !collision {
			return id, name, nil
		}
		name = fmt.Sprintf("%s-%d", base, suffix)
	}
}

func ValidateChangeSetName(name string) error {
	if name == "" || name != strings.TrimSpace(name) {
		return errors.New("change-set name must be trimmed and non-empty")
	}
	if strings.ContainsAny(name, "\r\n") {
		return errors.New("change-set name must fit on one line")
	}
	return nil
}

// LoadChangeSets enumerates only the two Change Sets-owned ref namespaces.
// Each bad record remains visible without preventing other records or Accepted
// Architecture from loading.
func (manager *Manager) LoadChangeSets(ctx context.Context, storeID string) ([]ChangeSet, []UnavailableChangeSet, error) {
	storePath, err := manager.StorePath(storeID)
	if err != nil {
		return nil, nil, err
	}
	active, err := manager.git.refsUnder(ctx, storePath, activeChangeSetPrefix)
	if err != nil {
		return nil, nil, fmt.Errorf("enumerate active change sets: %w", err)
	}
	applied, err := manager.git.refsUnder(ctx, storePath, appliedChangeSetPrefix)
	if err != nil {
		return nil, nil, fmt.Errorf("enumerate applied change sets: %w", err)
	}
	type ownedRef struct {
		refEntry
		lifecycle string
		id        string
	}
	refs := make([]ownedRef, 0, len(active)+len(applied))
	for _, entry := range active {
		refs = append(refs, ownedRef{refEntry: entry, lifecycle: "active", id: strings.TrimPrefix(entry.Name, activeChangeSetPrefix)})
	}
	for _, entry := range applied {
		refs = append(refs, ownedRef{refEntry: entry, lifecycle: "applied", id: strings.TrimPrefix(entry.Name, appliedChangeSetPrefix)})
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].Name < refs[j].Name })
	counts := make(map[string]int)
	for _, entry := range refs {
		counts[entry.id]++
	}
	valid := make([]ChangeSet, 0, len(refs))
	unavailable := make([]UnavailableChangeSet, 0)
	for _, entry := range refs {
		parsed, parseErr := uuid.Parse(entry.id)
		unavailableReason := ""
		if parseErr != nil || parsed.String() != entry.id || strings.Contains(entry.id, "/") {
			unavailableReason = "invalid change-set ref identity"
		} else if counts[entry.id] != 1 {
			unavailableReason = "change-set identity occurs in more than one lifecycle"
		}
		if unavailableReason != "" {
			name := ""
			if entry.lifecycle == "active" {
				name = manager.readableChangeSetNameAtObject(ctx, storePath, entry.Object)
			}
			unavailable = append(unavailable, UnavailableChangeSet{ID: entry.id, Name: name, Lifecycle: entry.lifecycle, Ref: entry.Name, Reason: unavailableReason})
			continue
		}
		record, loadErr := manager.loadChangeSet(ctx, storePath, storeID, entry.lifecycle, entry.id, entry.Name, entry.Object)
		if loadErr != nil {
			unavailable = append(unavailable, UnavailableChangeSet{ID: entry.id, Name: record.Name, Lifecycle: entry.lifecycle, Ref: entry.Name, Reason: loadErr.Error()})
			continue
		}
		valid = append(valid, record)
	}

	// A readable active name on any record participates in conflict detection,
	// including a record unavailable for a different reason. No enumeration
	// order is allowed to choose a winner.
	conflicting := make(map[int]bool)
	for left, record := range valid {
		if record.Lifecycle != "active" {
			continue
		}
		for right := left + 1; right < len(valid); right++ {
			if valid[right].Lifecycle == "active" && strings.EqualFold(record.Name, valid[right].Name) {
				conflicting[left], conflicting[right] = true, true
			}
		}
		for _, unavailableRecord := range unavailable {
			if unavailableRecord.Lifecycle == "active" && unavailableRecord.Name != "" && strings.EqualFold(record.Name, unavailableRecord.Name) {
				conflicting[left] = true
			}
		}
	}
	if len(conflicting) > 0 {
		kept := make([]ChangeSet, 0, len(valid)-len(conflicting))
		for index, record := range valid {
			if conflicting[index] {
				unavailable = append(unavailable, UnavailableChangeSet{ID: record.ID, Name: record.Name, Lifecycle: record.Lifecycle, Ref: changeSetRef(record.Lifecycle, record.ID), Reason: "active change-set name conflicts with another record"})
				continue
			}
			kept = append(kept, record)
		}
		valid = kept
	}
	sort.Slice(valid, func(i, j int) bool {
		if valid[i].Lifecycle != valid[j].Lifecycle {
			return valid[i].Lifecycle < valid[j].Lifecycle
		}
		if valid[i].Name != valid[j].Name {
			return valid[i].Name < valid[j].Name
		}
		return valid[i].ID < valid[j].ID
	})
	sort.Slice(unavailable, func(i, j int) bool { return unavailable[i].Ref < unavailable[j].Ref })
	return valid, unavailable, nil
}

func (manager *Manager) loadChangeSet(ctx context.Context, storePath, storeID, lifecycle, id, ref, object string) (ChangeSet, error) {
	record := ChangeSet{ID: id, Lifecycle: lifecycle, RefObject: object}
	objectType, err := manager.git.objectType(ctx, storePath, object)
	if err != nil || objectType != "commit" {
		if lifecycle == "active" {
			record.Name = manager.readableChangeSetNameAtObject(ctx, storePath, object)
		}
		return record, errors.New("change-set ref does not name a commit")
	}
	parent, err := manager.git.commitParent(ctx, storePath, object)
	if err != nil {
		if lifecycle == "active" {
			record.Name = manager.readableChangeSetNameAtObject(ctx, storePath, object)
		}
		return record, err
	}
	entries, err := manager.git.directTreeEntries(ctx, storePath, object)
	if err != nil {
		return record, fmt.Errorf("read change-set envelope: %w", err)
	}
	byPath := make(map[string]treeEntry, len(entries))
	var envelopeErr error
	for _, entry := range entries {
		if _, duplicate := byPath[entry.Path]; duplicate {
			if envelopeErr == nil {
				envelopeErr = errors.New("change-set envelope contains a duplicate path")
			}
		}
		byPath[entry.Path] = entry
	}
	for path, entry := range byPath {
		switch path {
		case "change-set.yaml", "proposal.md", "changes.yaml":
			if (entry.Type != "blob" || entry.Mode != "100644") && envelopeErr == nil {
				envelopeErr = fmt.Errorf("change-set envelope path %q is not a 100644 blob", path)
			}
		case "architecture":
			if (entry.Type != "tree" || entry.Mode != "040000") && envelopeErr == nil {
				envelopeErr = errors.New("change-set architecture entry is not a tree")
			}
		default:
			if envelopeErr == nil {
				envelopeErr = fmt.Errorf("change-set envelope contains unknown path %q", path)
			}
		}
	}
	for _, required := range []string{"change-set.yaml", "proposal.md", "changes.yaml"} {
		if _, present := byPath[required]; !present {
			if envelopeErr == nil {
				envelopeErr = fmt.Errorf("change-set envelope is missing %q", required)
			}
		}
	}
	metadataEntry, metadataPresent := byPath["change-set.yaml"]
	if !metadataPresent || metadataEntry.Type != "blob" {
		if envelopeErr == nil {
			envelopeErr = errors.New("change-set envelope has no readable metadata")
		}
		return record, envelopeErr
	}
	metadataBytes, err := manager.git.readBlob(ctx, storePath, metadataEntry.Object)
	if err != nil {
		return record, errors.New("read change-set metadata")
	}
	record.Name = readableChangeSetName(metadataBytes)
	metadata, err := parseChangeSetMetadata(metadataBytes)
	if err != nil {
		return record, err
	}
	record.Name = metadata.Name
	if envelopeErr != nil {
		return record, envelopeErr
	}
	if metadata.ID != id {
		return record, errors.New("change-set metadata identity does not match its ref")
	}
	if lifecycle == "active" && metadata.AppliedRevision != "" {
		return record, errors.New("active change set contains applied_revision")
	}
	if lifecycle == "applied" && metadata.AppliedRevision == "" {
		return record, errors.New("applied change set is missing applied_revision")
	}
	if lifecycle == "active" && parent != metadata.BaseRevision {
		return record, errors.New("active state commit parent does not match base_revision")
	}
	if lifecycle == "applied" && parent != metadata.AppliedRevision {
		return record, errors.New("applied state commit parent does not match applied_revision")
	}
	proposal, err := manager.git.readBlob(ctx, storePath, byPath["proposal.md"].Object)
	if err != nil || !utf8.Valid(proposal) {
		return record, errors.New("proposal.md is not valid UTF-8")
	}
	changesBytes, err := manager.git.readBlob(ctx, storePath, byPath["changes.yaml"].Object)
	if err != nil {
		return record, errors.New("read changes.yaml")
	}
	changes, composition, err := parseChangeState(changesBytes)
	if err != nil {
		return record, err
	}
	base, err := manager.load(ctx, storePath, uuid.MustParse(storeID), metadata.BaseRevision)
	if err != nil {
		return record, fmt.Errorf("load change-set base: %w", err)
	}
	record.BaseRevision = metadata.BaseRevision
	record.Generation = metadata.Generation
	record.Proposal = string(proposal)
	record.AppliedRevision = metadata.AppliedRevision
	record.BaseSnapshot = base
	record.Changes = changes
	record.Composition = composition
	if metadata.Review != nil {
		record.Review = &ChangeSetReview{BaseRevision: metadata.Review.BaseRevision, CandidateTree: metadata.Review.CandidateTree, Generation: metadata.Review.Generation}
	}

	candidate, candidateErr := manager.ConstructCandidate(ctx, base, changes, composition)
	architectureEntry, hasArchitecture := byPath["architecture"]
	if !hasArchitecture {
		if candidateErr == nil {
			return record, errors.New("valid change set is missing its architecture tree")
		}
		if !isCandidateValidationError(candidateErr) {
			return record, fmt.Errorf("reconstruct invalid change set: %w", candidateErr)
		}
		if record.Review != nil {
			return record, errors.New("invalid change set contains a review binding")
		}
		if lifecycle == "applied" {
			return record, errors.New("applied change set has no valid architecture tree")
		}
		record.ValidationError = candidateErr
		return record, nil
	}
	if candidateErr != nil {
		return record, fmt.Errorf("reconstruct change-set candidate: %w", candidateErr)
	}
	if candidate.Tree() != architectureEntry.Object {
		return record, errors.New("change-set candidate reconstruction does not match architecture tree")
	}
	if _, err := manager.load(ctx, storePath, uuid.MustParse(storeID), architectureEntry.Object); err != nil {
		return record, fmt.Errorf("load nested change-set Architecture: %w", err)
	}
	record.Candidate = &candidate
	if record.Review != nil && (record.Review.BaseRevision != record.BaseRevision || record.Review.CandidateTree != candidate.Tree() || record.Review.Generation != record.Generation) {
		return record, errors.New("change-set review binding does not match current record")
	}
	if lifecycle == "applied" {
		if record.Review == nil {
			return record, errors.New("applied change set is missing its review binding")
		}
		appliedParent, err := manager.git.commitParent(ctx, storePath, metadata.AppliedRevision)
		if err != nil || appliedParent != metadata.BaseRevision {
			return record, errors.New("applied revision is not the direct successor of base_revision")
		}
		appliedTree, err := manager.git.commitTree(ctx, storePath, metadata.AppliedRevision)
		if err != nil || appliedTree != candidate.Tree() {
			return record, errors.New("applied revision tree does not match the change-set candidate")
		}
	}
	_ = ref
	return record, nil
}

func isCandidateValidationError(err error) bool {
	var componentError *ComponentValidationError
	var diagramError *DiagramValidationError
	return errors.As(err, &componentError) || errors.As(err, &diagramError)
}

func (manager *Manager) WriteActiveChangeSet(ctx context.Context, storeID string, record ChangeSet, expectedObject string) (string, error) {
	if record.Lifecycle != "active" || record.AppliedRevision != "" {
		return "", errors.New("only an active change set can be written")
	}
	return manager.writeChangeSet(ctx, storeID, record, expectedObject)
}

func (manager *Manager) writeChangeSet(ctx context.Context, storeID string, record ChangeSet, expectedObject string) (string, error) {
	commit, err := manager.prepareChangeSet(ctx, storeID, record)
	if err != nil {
		return "", err
	}
	storePath, err := manager.StorePath(storeID)
	if err != nil {
		return "", err
	}
	ref := changeSetRef(record.Lifecycle, record.ID)
	if expectedObject == "" {
		err = manager.git.createRef(ctx, storePath, ref, commit)
	} else {
		err = manager.git.updateRef(ctx, storePath, ref, commit, expectedObject)
	}
	if err != nil {
		return "", err
	}
	return commit, nil
}

func (manager *Manager) PrepareActiveChangeSet(ctx context.Context, storeID string, record ChangeSet) (string, error) {
	if record.Lifecycle != "active" || record.AppliedRevision != "" {
		return "", errors.New("only an active change set can be prepared")
	}
	return manager.prepareChangeSet(ctx, storeID, record)
}

func (manager *Manager) prepareChangeSet(ctx context.Context, storeID string, record ChangeSet) (string, error) {
	storePath, err := manager.StorePath(storeID)
	if err != nil {
		return "", err
	}
	if err := manager.validateChangeSetForWrite(ctx, storeID, record); err != nil {
		return "", err
	}
	metadata := changeSetMetadata{Format: "workbraid-change-set", Version: 1, ID: record.ID, Name: record.Name, BaseRevision: record.BaseRevision, Generation: record.Generation, AppliedRevision: record.AppliedRevision}
	if record.Review != nil {
		metadata.Review = &changeSetReview{BaseRevision: record.Review.BaseRevision, CandidateTree: record.Review.CandidateTree, Generation: record.Review.Generation}
	}
	metadataBytes, err := yaml.Marshal(metadata)
	if err != nil {
		return "", err
	}
	changesBytes, err := marshalChangeState(record.Changes, record.Composition)
	if err != nil {
		return "", err
	}
	blobs := make(map[string]string, 3)
	for path, contents := range map[string][]byte{"change-set.yaml": metadataBytes, "proposal.md": []byte(record.Proposal), "changes.yaml": changesBytes} {
		blob, err := manager.git.writeBlob(ctx, storePath, contents)
		if err != nil {
			return "", fmt.Errorf("write %s: %w", path, err)
		}
		blobs[path] = blob
	}
	var treeSource strings.Builder
	if record.Candidate != nil {
		fmt.Fprintf(&treeSource, "040000 tree %s\tarchitecture\n", record.Candidate.Tree())
	}
	for _, path := range []string{"change-set.yaml", "changes.yaml", "proposal.md"} {
		fmt.Fprintf(&treeSource, "100644 blob %s\t%s\n", blobs[path], path)
	}
	tree, err := manager.git.makeTree(ctx, storePath, []byte(treeSource.String()))
	if err != nil {
		return "", fmt.Errorf("create change-set envelope: %w", err)
	}
	parent := record.BaseRevision
	if record.Lifecycle == "applied" {
		parent = record.AppliedRevision
	}
	commit, err := manager.git.makeStateCommit(ctx, storePath, tree, parent)
	if err != nil {
		return "", fmt.Errorf("create change-set state commit: %w", err)
	}
	return commit, nil
}

func (manager *Manager) PrepareAppliedChangeSet(ctx context.Context, storeID string, record ChangeSet) (string, error) {
	if record.Lifecycle != "applied" || record.AppliedRevision == "" {
		return "", errors.New("applied record is incomplete")
	}
	// Write the immutable envelope and commit without publishing a ref. The
	// later three-ref transaction is the sole lifecycle boundary.
	return manager.prepareChangeSet(ctx, storeID, record)
}

func (manager *Manager) PublishReconciliation(ctx context.Context, storeID, id, accepted, oldState, newState string) error {
	if !canonicalUUID(id) || !validObjectID(accepted) || !validObjectID(oldState) || !validObjectID(newState) {
		return errors.New("invalid reconciliation transaction inputs")
	}
	storePath, err := manager.StorePath(storeID)
	if err != nil {
		return err
	}
	return manager.git.reconcileChangeSet(ctx, storePath, accepted, changeSetRef("active", id), oldState, newState)
}

func (manager *Manager) DeleteActiveChangeSet(ctx context.Context, storeID, id, expectedObject string) error {
	storePath, err := manager.StorePath(storeID)
	if err != nil {
		return err
	}
	return manager.git.deleteRef(ctx, storePath, changeSetRef("active", id), expectedObject)
}

// ObserveActiveChangeSet reads the real active ref without interpreting a
// cached generation as authority. Callers retain the application mutex.
func (manager *Manager) ObserveActiveChangeSet(ctx context.Context, storeID, id string) (string, bool, error) {
	if !canonicalUUID(id) {
		return "", false, errors.New("invalid change-set identity")
	}
	storePath, err := manager.StorePath(storeID)
	if err != nil {
		return "", false, err
	}
	return manager.git.resolveRef(ctx, storePath, changeSetRef("active", id))
}

func (manager *Manager) AcceptChangeSet(ctx context.Context, storeID, base, successor, id, activeObject, appliedObject string) error {
	storePath, err := manager.StorePath(storeID)
	if err != nil {
		return err
	}
	return manager.git.acceptChangeSet(ctx, storePath, base, successor, changeSetRef("active", id), activeObject, changeSetRef("applied", id), appliedObject)
}

func changeSetRef(lifecycle, id string) string {
	if lifecycle == "applied" {
		return appliedChangeSetPrefix + id
	}
	return activeChangeSetPrefix + id
}

func (manager *Manager) validateChangeSetForWrite(ctx context.Context, storeID string, record ChangeSet) error {
	parsed, err := uuid.Parse(record.ID)
	if err != nil || parsed.String() != record.ID {
		return errors.New("change-set identity is invalid")
	}
	if err := ValidateChangeSetName(record.Name); err != nil {
		return err
	}
	if !validObjectID(record.BaseRevision) || !utf8.ValidString(record.Proposal) {
		return errors.New("change-set base or proposal is invalid")
	}
	if record.BaseSnapshot.StoreID() != storeID || record.BaseSnapshot.Revision() != record.BaseRevision {
		return errors.New("change-set base snapshot does not match its store and revision")
	}
	if record.Candidate == nil && record.Review != nil {
		return errors.New("invalid change set cannot contain review")
	}
	if record.Review != nil && (record.Review.BaseRevision != record.BaseRevision || record.Review.Generation != record.Generation || record.Candidate == nil || record.Review.CandidateTree != record.Candidate.Tree()) {
		return errors.New("change-set review does not match current state")
	}
	if record.Lifecycle == "applied" && (!validObjectID(record.AppliedRevision) || record.Review == nil || record.Candidate == nil) {
		return errors.New("applied change set is incomplete")
	}
	if err := validateChangeState(record.Changes, record.Composition); err != nil {
		return err
	}
	reconstructed, reconstructionErr := manager.ConstructCandidate(ctx, record.BaseSnapshot, record.Changes, record.Composition)
	switch {
	case reconstructionErr == nil && record.Candidate == nil:
		return errors.New("valid change set is missing its candidate")
	case reconstructionErr == nil && record.Candidate.Tree() != reconstructed.Tree():
		return errors.New("change-set candidate does not match its concrete facts")
	case reconstructionErr != nil && record.Candidate != nil:
		return errors.New("invalid change set contains a candidate")
	case reconstructionErr != nil && !isCandidateValidationError(reconstructionErr):
		return fmt.Errorf("change-set concrete facts cannot be reconstructed: %w", reconstructionErr)
	}
	return nil
}

func validObjectID(value string) bool {
	if len(value) != 40 {
		return false
	}
	for _, character := range value {
		if !(character >= '0' && character <= '9') && !(character >= 'a' && character <= 'f') {
			return false
		}
	}
	return true
}

func marshalChangeState(changes []ComponentChange, composition CandidateComposition) ([]byte, error) {
	state := changeState{Format: "workbraid-change-state", Version: 2,
		DetailReassignments: append([]DetailReassignment{}, composition.DetailReassignments...),
		Components:          make([]changeStateComponent, len(changes)),
		NewComponentHomes:   nonNilHomes(composition.NewComponentHomes), DetailDiagrams: nonNilDetails(composition.DetailDiagrams),
		DiagramTitles: nonNilTitles(composition.DiagramTitles), HomeMoves: nonNilMoves(composition.HomeMoves), References: nonNilReferences(composition.References),
	}
	for index, change := range changes {
		item := changeStateComponent{ID: change.ID, Path: change.Path, New: change.New, Title: change.Title, Description: change.Description,
			TitleChanged: change.TitleChanged, DescriptionChanged: change.DescriptionChanged, RelationshipsChanged: change.RelationshipsChanged,
			Relationships: make([]changeStateRelationship, len(change.Relationships)),
		}
		for relationshipIndex, relationship := range change.Relationships {
			item.Relationships[relationshipIndex] = changeStateRelationship{Target: relationship.TargetID, Label: relationship.Label}
		}
		state.Components[index] = item
	}
	return yaml.Marshal(state)
}

func parseChangeState(contents []byte) ([]ComponentChange, CandidateComposition, error) {
	root, err := oneYAMLMapping(contents, "changes.yaml")
	if err != nil {
		return nil, CandidateComposition{}, err
	}
	if err := validateChangeStateYAML(root); err != nil {
		return nil, CandidateComposition{}, err
	}
	var state changeState
	if err := root.Decode(&state); err != nil {
		return nil, CandidateComposition{}, fmt.Errorf("parse changes.yaml: %w", err)
	}
	changes := make([]ComponentChange, len(state.Components))
	for index, item := range state.Components {
		relationships := make([]AuthoringRelationship, len(item.Relationships))
		for relationshipIndex, relationship := range item.Relationships {
			relationships[relationshipIndex] = AuthoringRelationship{TargetID: relationship.Target, Label: relationship.Label}
		}
		changes[index] = ComponentChange{ID: item.ID, Path: item.Path, New: item.New, Title: item.Title, Description: item.Description,
			TitleChanged: item.TitleChanged, DescriptionChanged: item.DescriptionChanged, RelationshipsChanged: item.RelationshipsChanged, Relationships: relationships,
		}
	}
	composition := CandidateComposition{DetailReassignments: state.DetailReassignments, NewComponentHomes: state.NewComponentHomes, DetailDiagrams: state.DetailDiagrams, DiagramTitles: state.DiagramTitles, HomeMoves: state.HomeMoves, References: state.References}
	if err := validateChangeState(changes, composition); err != nil {
		return nil, CandidateComposition{}, err
	}
	return changes, composition, nil
}

func parseChangeSetMetadata(contents []byte) (changeSetMetadata, error) {
	root, err := oneYAMLMapping(contents, "change-set.yaml")
	if err != nil {
		return changeSetMetadata{}, err
	}
	if err := validateChangeSetMetadataYAML(root); err != nil {
		return changeSetMetadata{}, err
	}
	var metadata changeSetMetadata
	if err := root.Decode(&metadata); err != nil {
		return changeSetMetadata{}, fmt.Errorf("parse change-set.yaml: %w", err)
	}
	if metadata.Format != "workbraid-change-set" || metadata.Version != 1 || metadata.ID == "" || !validObjectID(metadata.BaseRevision) || ValidateChangeSetName(metadata.Name) != nil {
		return metadata, errors.New("change-set metadata values are invalid")
	}
	parsed, err := uuid.Parse(metadata.ID)
	if err != nil || parsed.String() != metadata.ID {
		return metadata, errors.New("change-set metadata identity is invalid")
	}
	if metadata.Review != nil && (!validObjectID(metadata.Review.BaseRevision) || !validObjectID(metadata.Review.CandidateTree)) {
		return metadata, errors.New("change-set review object identity is invalid")
	}
	if metadata.AppliedRevision != "" && !validObjectID(metadata.AppliedRevision) {
		return metadata, errors.New("applied revision is invalid")
	}
	return metadata, nil
}

// readableChangeSetName extracts only an unambiguous valid display name from
// an otherwise malformed metadata mapping. It reserves no meaning from the
// malformed record beyond the approved active-name collision rule.
func readableChangeSetName(contents []byte) string {
	root, err := oneYAMLMapping(contents, "change-set.yaml")
	if err != nil {
		return ""
	}
	name := ""
	for index := 0; index < len(root.Content); index += 2 {
		key, value := root.Content[index], root.Content[index+1]
		if key.Kind != yaml.ScalarNode || key.ShortTag() != "!!str" || key.Value != "name" {
			continue
		}
		if name != "" || value.Kind != yaml.ScalarNode || value.ShortTag() != "!!str" || ValidateChangeSetName(value.Value) != nil {
			return ""
		}
		name = value.Value
	}
	return name
}

func (manager *Manager) readableChangeSetNameAtObject(ctx context.Context, storePath, object string) string {
	entries, err := manager.git.directTreeEntries(ctx, storePath, object)
	if err != nil {
		return ""
	}
	var metadata treeEntry
	found := false
	for _, entry := range entries {
		if entry.Path != "change-set.yaml" {
			continue
		}
		if found || entry.Type != "blob" {
			return ""
		}
		metadata, found = entry, true
	}
	if !found {
		return ""
	}
	contents, err := manager.git.readBlob(ctx, storePath, metadata.Object)
	if err != nil {
		return ""
	}
	return readableChangeSetName(contents)
}

func oneYAMLMapping(contents []byte, name string) (*yaml.Node, error) {
	if !utf8.Valid(contents) {
		return nil, fmt.Errorf("%s is not valid UTF-8", name)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(contents))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("parse %s: %w", name, err)
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%s must contain one YAML document", name)
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode || document.Content[0].ShortTag() != "!!map" {
		return nil, fmt.Errorf("%s must contain one mapping", name)
	}
	return document.Content[0], nil
}

func validateChangeSetMetadataYAML(root *yaml.Node) error {
	required := map[string]string{"format": "!!str", "version": "!!int", "id": "!!str", "name": "!!str", "base_revision": "!!str", "generation": "!!int"}
	optional := map[string]string{"review": "!!map", "applied_revision": "!!str"}
	seen, err := validateClosedMapping(root, "change-set.yaml", required, optional)
	if err != nil {
		return err
	}
	if review := seen["review"]; review != nil {
		_, err = validateClosedMapping(review, "change-set.yaml review", map[string]string{"base_revision": "!!str", "candidate_tree": "!!str", "generation": "!!int"}, nil)
	}
	return err
}

func validateChangeStateYAML(root *yaml.Node) error {
	required := map[string]string{"format": "!!str", "version": "!!int", "components": "!!seq", "new_component_homes": "!!seq", "detail_diagrams": "!!seq", "diagram_titles": "!!seq", "home_moves": "!!seq", "references": "!!seq"}
	seen, err := validateClosedMapping(root, "changes.yaml", required, map[string]string{"detail_reassignments": "!!seq"})
	if err != nil {
		return err
	}
	if scalarValue(seen["format"]) != "workbraid-change-state" || (scalarValue(seen["version"]) != "1" && scalarValue(seen["version"]) != "2") {
		return errors.New("changes.yaml format or version is unsupported")
	}
	if (scalarValue(seen["version"]) == "2") != (seen["detail_reassignments"] != nil) {
		return errors.New("changes.yaml detail_reassignments is required only in version 2")
	}
	if sequence := seen["detail_reassignments"]; sequence != nil {
		for _, item := range sequence.Content {
			if _, err := validateClosedMapping(item, "detail reassignment", map[string]string{"diagram_id": "!!str", "anchor_component_id": "!!str"}, nil); err != nil {
				return err
			}
		}
	}
	for index, item := range seen["components"].Content {
		fields, err := validateClosedMapping(item, fmt.Sprintf("changes.yaml component %d", index+1), map[string]string{
			"id": "!!str", "path": "!!str", "new": "!!bool", "title": "!!str", "description": "!!str", "title_changed": "!!bool", "description_changed": "!!bool", "relationships_changed": "!!bool", "relationships": "!!seq",
		}, nil)
		if err != nil {
			return err
		}
		for relationshipIndex, relationship := range fields["relationships"].Content {
			if _, err := validateClosedMapping(relationship, fmt.Sprintf("changes.yaml relationship %d.%d", index+1, relationshipIndex+1), map[string]string{"target": "!!str", "label": "!!str"}, nil); err != nil {
				return err
			}
		}
	}
	for _, spec := range []struct {
		field string
		keys  map[string]string
	}{
		{"new_component_homes", map[string]string{"component_id": "!!str", "diagram_id": "!!str"}},
		{"detail_diagrams", map[string]string{"id": "!!str", "path": "!!str", "title": "!!str", "anchor_component_id": "!!str"}},
		{"diagram_titles", map[string]string{"diagram_id": "!!str", "title": "!!str"}},
		{"home_moves", map[string]string{"component_id": "!!str", "diagram_id": "!!str"}},
		{"references", map[string]string{"diagram_id": "!!str", "component_id": "!!str", "present": "!!bool"}},
	} {
		for index, item := range seen[spec.field].Content {
			if _, err := validateClosedMapping(item, fmt.Sprintf("changes.yaml %s item %d", spec.field, index+1), spec.keys, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateClosedMapping(root *yaml.Node, name string, required, optional map[string]string) (map[string]*yaml.Node, error) {
	if root == nil || root.Kind != yaml.MappingNode || root.ShortTag() != "!!map" {
		return nil, fmt.Errorf("%s must be a mapping", name)
	}
	seen := make(map[string]*yaml.Node, len(required)+len(optional))
	for index := 0; index < len(root.Content); index += 2 {
		key, value := root.Content[index], root.Content[index+1]
		if key.Kind != yaml.ScalarNode || key.ShortTag() != "!!str" {
			return nil, fmt.Errorf("%s field names must be strings", name)
		}
		expected, known := required[key.Value]
		if !known {
			expected, known = optional[key.Value]
		}
		if !known {
			return nil, fmt.Errorf("%s contains unknown field %q", name, key.Value)
		}
		if _, duplicate := seen[key.Value]; duplicate {
			return nil, fmt.Errorf("%s contains duplicate field %q", name, key.Value)
		}
		if value.ShortTag() != expected {
			return nil, fmt.Errorf("%s field %s has the wrong type", name, key.Value)
		}
		seen[key.Value] = value
	}
	for key := range required {
		if seen[key] == nil {
			return nil, fmt.Errorf("%s is missing field %q", name, key)
		}
	}
	return seen, nil
}

func scalarValue(node *yaml.Node) string {
	if node == nil {
		return ""
	}
	return node.Value
}

func validateChangeState(changes []ComponentChange, composition CandidateComposition) error {
	seenComponents := make(map[string]struct{}, len(changes))
	for _, change := range changes {
		if !canonicalUUID(change.ID) || !validNewComponentPath(change.Path) {
			return errors.New("changes.yaml Component identity or path is invalid")
		}
		if _, duplicate := seenComponents[change.ID]; duplicate {
			return errors.New("changes.yaml changes a Component more than once")
		}
		seenComponents[change.ID] = struct{}{}
	}
	seenHomes := make(map[string]struct{}, len(composition.NewComponentHomes))
	for _, value := range composition.NewComponentHomes {
		if !canonicalUUID(value.ComponentID) || !canonicalUUID(value.DiagramID) {
			return errors.New("changes.yaml new Component home identity is invalid")
		}
		if _, duplicate := seenHomes[value.ComponentID]; duplicate {
			return errors.New("changes.yaml contains duplicate new Component home")
		}
		seenHomes[value.ComponentID] = struct{}{}
	}
	seenDetails := make(map[string]struct{}, len(composition.DetailDiagrams))
	for _, value := range composition.DetailDiagrams {
		if !canonicalUUID(value.ID) || !canonicalUUID(value.AnchorComponentID) || !validNewDiagramPath(value.Path) {
			return errors.New("changes.yaml detail Diagram identity or path is invalid")
		}
		if _, duplicate := seenDetails[value.ID]; duplicate {
			return errors.New("changes.yaml contains duplicate detail Diagram")
		}
		seenDetails[value.ID] = struct{}{}
	}
	seenReassignments := make(map[string]bool)
	for _, value := range composition.DetailReassignments {
		_, isNew := seenDetails[value.DiagramID]
		if !canonicalUUID(value.DiagramID) || !canonicalUUID(value.AnchorComponentID) || isNew || seenReassignments[value.DiagramID] {
			return errors.New("changes.yaml detail reassignment must uniquely name a base Diagram and anchor")
		}
		seenReassignments[value.DiagramID] = true
	}
	if duplicateFieldID(composition.DiagramTitles, func(value DiagramTitleChange) string { return value.DiagramID }) || duplicateFieldID(composition.HomeMoves, func(value ComponentHomeMove) string { return value.ComponentID }) {
		return errors.New("changes.yaml contains duplicate title or home changes")
	}
	for _, value := range composition.DiagramTitles {
		if !canonicalUUID(value.DiagramID) {
			return errors.New("changes.yaml Diagram title identity is invalid")
		}
	}
	for _, value := range composition.HomeMoves {
		if !canonicalUUID(value.ComponentID) || !canonicalUUID(value.DiagramID) {
			return errors.New("changes.yaml home move identity is invalid")
		}
	}
	seenReferences := make(map[string]struct{}, len(composition.References))
	for _, value := range composition.References {
		if !canonicalUUID(value.ComponentID) || !canonicalUUID(value.DiagramID) {
			return errors.New("changes.yaml reference identity is invalid")
		}
		key := value.DiagramID + "\x00" + value.ComponentID
		if _, duplicate := seenReferences[key]; duplicate {
			return errors.New("changes.yaml contains duplicate reference state")
		}
		seenReferences[key] = struct{}{}
	}
	return nil
}

func duplicateFieldID[T any](values []T, identity func(T) string) bool {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		id := identity(value)
		if _, duplicate := seen[id]; duplicate {
			return true
		}
		seen[id] = struct{}{}
	}
	return false
}

func canonicalUUID(value string) bool {
	parsed, err := uuid.Parse(value)
	return err == nil && parsed.String() == value
}

func nonNilHomes(values []NewComponentHome) []NewComponentHome {
	return append([]NewComponentHome{}, values...)
}
func nonNilDetails(values []DetailDiagramChange) []DetailDiagramChange {
	return append([]DetailDiagramChange{}, values...)
}
func nonNilTitles(values []DiagramTitleChange) []DiagramTitleChange {
	return append([]DiagramTitleChange{}, values...)
}
func nonNilMoves(values []ComponentHomeMove) []ComponentHomeMove {
	return append([]ComponentHomeMove{}, values...)
}
func nonNilReferences(values []ReferenceAppearanceChange) []ReferenceAppearanceChange {
	return append([]ReferenceAppearanceChange{}, values...)
}
