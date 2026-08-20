package architecture

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	"go.yaml.in/yaml/v4"
)

const (
	acceptedRef = "refs/heads/accepted"
	zeroObject  = "0000000000000000000000000000000000000000"
)

var (
	ErrIncomplete  = errors.New("Architecture setup is incomplete")
	ErrUnavailable = errors.New("Architecture is unavailable")
	ErrInvalid     = errors.New("Architecture store is invalid")
	ErrUnsupported = errors.New("Architecture is unsupported")
)

// Snapshot is immutable accepted Architecture state pinned to one Git commit.
type Snapshot struct {
	storeID       uuid.UUID
	revision      string
	formatVersion int
	components    []component
	rootDiagram   uuid.UUID
	diagrams      []diagram
}

type diagram struct {
	id          uuid.UUID
	path        string
	title       string
	appearances []diagramAppearance
}

type diagramAppearance struct {
	component     uuid.UUID
	role          string
	detailDiagram uuid.UUID
	hasDetailLink bool
}

type component struct {
	id            uuid.UUID
	path          string
	title         string
	body          []byte
	relationships []componentRelationship
	mode          string
	source        []byte
	markdownStart int
	headingStart  int
	headingEnd    int
	headingStyle  headingStyle
}

type headingStyle uint8

const (
	headingATX headingStyle = iota
	headingSetext
)

type componentRelationship struct {
	target uuid.UUID
	label  string
}

func (snapshot Snapshot) Revision() string    { return snapshot.revision }
func (snapshot Snapshot) ComponentCount() int { return len(snapshot.components) }
func (snapshot Snapshot) StoreID() string     { return snapshot.storeID.String() }
func (snapshot Snapshot) FormatVersion() int  { return snapshot.formatVersion }
func (snapshot Snapshot) ComponentTitles() []string {
	titles := make([]string, len(snapshot.components))
	for index := range snapshot.components {
		titles[index] = snapshot.components[index].title
	}
	return titles
}

// DiagramProjection is the immutable browser-facing projection of one
// accepted v2 Diagram. Hierarchy, appearances, navigation, ordinary edges,
// and boundary references are resolved by the accepted loader rather than by
// the browser.
type DiagramProjection struct {
	ID                      string
	Title                   string
	Depth                   int
	ParentDiagramID         string
	ParentAnchorComponentID string
	Breadcrumbs             []DiagramBreadcrumb
	Appearances             []DiagramAppearance
	Boundaries              []DiagramBoundary
	Relationships           []DiagramRelationship
}

type DiagramBreadcrumb struct {
	ID                     string
	Title                  string
	FocusAnchorComponentID string
}

type DiagramAppearance struct {
	ComponentID        string
	Role               string
	DetailDiagramID    string
	DetailDiagramTitle string
}

type DiagramBoundary struct {
	Key              string
	ComponentID      string
	Title            string
	Context          string
	HomeDiagramID    string
	HomeDiagramTitle string
}

type DiagramRelationship struct {
	Key               string
	SourceNodeKey     string
	TargetNodeKey     string
	SourceComponentID string
	TargetComponentID string
	Label             string
}

func (snapshot Snapshot) RootDiagramID() string {
	if snapshot.formatVersion != 2 {
		return ""
	}
	return snapshot.rootDiagram.String()
}

func (snapshot Snapshot) DiagramProjections() []DiagramProjection {
	if snapshot.formatVersion != 2 {
		return nil
	}
	componentsByID := make(map[uuid.UUID]component, len(snapshot.components))
	componentTitleCounts := make(map[string]int, len(snapshot.components))
	for _, component := range snapshot.components {
		componentsByID[component.id] = component
		componentTitleCounts[component.title]++
	}
	diagramsByID := make(map[uuid.UUID]diagram, len(snapshot.diagrams))
	homeByComponent := make(map[uuid.UUID]uuid.UUID, len(snapshot.components))
	parentByDiagram := make(map[uuid.UUID]diagramParent, len(snapshot.diagrams))
	for _, current := range snapshot.diagrams {
		diagramsByID[current.id] = current
		for _, appearance := range current.appearances {
			if appearance.role == "home" {
				homeByComponent[appearance.component] = current.id
			}
			if appearance.hasDetailLink {
				parentByDiagram[appearance.detailDiagram] = diagramParent{diagram: current.id, anchor: appearance.component}
			}
		}
	}

	ordered := make([]diagram, 0, len(snapshot.diagrams))
	var appendDiagram func(uuid.UUID)
	appendDiagram = func(id uuid.UUID) {
		current := diagramsByID[id]
		ordered = append(ordered, current)
		for _, appearance := range current.appearances {
			if appearance.hasDetailLink {
				appendDiagram(appearance.detailDiagram)
			}
		}
	}
	appendDiagram(snapshot.rootDiagram)

	projections := make([]DiagramProjection, 0, len(ordered))
	for _, current := range ordered {
		projection := DiagramProjection{ID: current.id.String(), Title: current.title}
		if parent, exists := parentByDiagram[current.id]; exists {
			projection.ParentDiagramID = parent.diagram.String()
			projection.ParentAnchorComponentID = parent.anchor.String()
		}
		projection.Breadcrumbs = diagramBreadcrumbs(current.id, snapshot.rootDiagram, diagramsByID, parentByDiagram)
		projection.Depth = len(projection.Breadcrumbs) - 1

		present := make(map[uuid.UUID]string, len(current.appearances))
		for _, appearance := range current.appearances {
			value := DiagramAppearance{ComponentID: appearance.component.String(), Role: appearance.role}
			present[appearance.component] = appearance.component.String()
			if appearance.hasDetailLink {
				value.DetailDiagramID = appearance.detailDiagram.String()
				value.DetailDiagramTitle = diagramsByID[appearance.detailDiagram].title
			}
			projection.Appearances = append(projection.Appearances, value)
		}

		boundaries := make(map[uuid.UUID]string)
		for _, source := range snapshot.components {
			for relationshipIndex, relationship := range source.relationships {
				sourceKey, sourcePresent := present[source.id]
				targetKey, targetPresent := present[relationship.target]
				if sourcePresent == targetPresent {
					if !sourcePresent {
						continue
					}
				} else if !sourcePresent {
					sourceKey = ensureDiagramBoundary(&projection, boundaries, source.id, componentsByID, componentTitleCounts, homeByComponent, diagramsByID)
				} else {
					targetKey = ensureDiagramBoundary(&projection, boundaries, relationship.target, componentsByID, componentTitleCounts, homeByComponent, diagramsByID)
				}
				projection.Relationships = append(projection.Relationships, DiagramRelationship{
					Key:           fmt.Sprintf("diagram:%s:%s:%d", current.id, source.id, relationshipIndex),
					SourceNodeKey: sourceKey, TargetNodeKey: targetKey,
					SourceComponentID: source.id.String(), TargetComponentID: relationship.target.String(), Label: relationship.label,
				})
			}
		}
		projections = append(projections, projection)
	}
	return projections
}

type diagramParent struct {
	diagram uuid.UUID
	anchor  uuid.UUID
}

func diagramBreadcrumbs(selected, root uuid.UUID, diagrams map[uuid.UUID]diagram, parents map[uuid.UUID]diagramParent) []DiagramBreadcrumb {
	path := []uuid.UUID{selected}
	for current := selected; current != root; {
		parent, exists := parents[current]
		if !exists {
			break
		}
		current = parent.diagram
		path = append(path, current)
	}
	for left, right := 0, len(path)-1; left < right; left, right = left+1, right-1 {
		path[left], path[right] = path[right], path[left]
	}
	result := make([]DiagramBreadcrumb, len(path))
	for index, id := range path {
		result[index] = DiagramBreadcrumb{ID: id.String(), Title: diagrams[id].title}
		if index+1 < len(path) {
			result[index].FocusAnchorComponentID = parents[path[index+1]].anchor.String()
		}
	}
	return result
}

func ensureDiagramBoundary(projection *DiagramProjection, existing map[uuid.UUID]string, componentID uuid.UUID, components map[uuid.UUID]component, titleCounts map[string]int, homes map[uuid.UUID]uuid.UUID, diagrams map[uuid.UUID]diagram) string {
	if key, found := existing[componentID]; found {
		return key
	}
	key := "boundary:" + componentID.String()
	homeID := homes[componentID]
	boundary := DiagramBoundary{
		Key: key, ComponentID: componentID.String(), Title: components[componentID].title,
		HomeDiagramID: homeID.String(), HomeDiagramTitle: diagrams[homeID].title,
	}
	if titleCounts[boundary.Title] > 1 {
		boundary.Context = filepath.Base(components[componentID].path)
	}
	projection.Boundaries = append(projection.Boundaries, boundary)
	existing[componentID] = key
	return key
}

// AuthoringComponent is the structured projection used by the local browser.
// Canonical Markdown interpretation remains owned by the accepted loader.
type AuthoringComponent struct {
	ID            string
	Title         string
	Description   string
	Filename      string
	Relationships []AuthoringRelationship
}

// AuthoringRelationship is the accepted browser projection of an authored
// outgoing relationship. Its slice order is retained for faithful
// representation only and has no domain meaning.
type AuthoringRelationship struct {
	TargetID string
	Label    string
}

func (snapshot Snapshot) AuthoringComponents() []AuthoringComponent {
	components := make([]AuthoringComponent, len(snapshot.components))
	for index := range snapshot.components {
		relationships := make([]AuthoringRelationship, len(snapshot.components[index].relationships))
		for relationshipIndex, relationship := range snapshot.components[index].relationships {
			relationships[relationshipIndex] = AuthoringRelationship{
				TargetID: relationship.target.String(),
				Label:    relationship.label,
			}
		}
		components[index] = AuthoringComponent{
			ID:            snapshot.components[index].id.String(),
			Title:         snapshot.components[index].title,
			Description:   string(snapshot.components[index].body),
			Filename:      filepath.Base(snapshot.components[index].path),
			Relationships: relationships,
		}
	}
	return components
}

func (snapshot Snapshot) ChangeForAcceptedComponent(id string) (ComponentChange, bool) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return ComponentChange{}, false
	}
	for _, component := range snapshot.components {
		if component.id == parsed {
			relationships := make([]AuthoringRelationship, len(component.relationships))
			for index, relationship := range component.relationships {
				relationships[index] = AuthoringRelationship{TargetID: relationship.target.String(), Label: relationship.label}
			}
			return ComponentChange{
				ID:            component.id.String(),
				Title:         component.title,
				Description:   string(component.body),
				Path:          component.path,
				Relationships: relationships,
			}, true
		}
	}
	return ComponentChange{}, false
}

// ComponentChange is one addition or replacement in a multi-file pending
// Architecture change set. Its path and identity are assigned by the backend,
// never by browser input.
type ComponentChange struct {
	ID                   string
	Title                string
	Description          string
	Path                 string
	New                  bool
	TitleChanged         bool
	DescriptionChanged   bool
	Relationships        []AuthoringRelationship
	RelationshipsChanged bool
}

// Candidate is a completely constructed and validated non-canonical tree.
type Candidate struct {
	tree     string
	snapshot Snapshot
}

func (candidate Candidate) Tree() string       { return candidate.tree }
func (candidate Candidate) Snapshot() Snapshot { return candidate.snapshot }
func (candidate Candidate) SnapshotAt(revision string) Snapshot {
	snapshot := candidate.snapshot
	snapshot.revision = revision
	return snapshot
}

var (
	ErrTitleRequired              = errors.New("component title is required")
	ErrTitleOneLine               = errors.New("component title must fit on one line")
	ErrRelationshipLabelRequired  = errors.New("relationship label is required")
	ErrRelationshipTargetRequired = errors.New("relationship target is required")
)

type ComponentValidationError struct {
	ComponentID          string
	RelationshipPosition int
	RelationshipField    string
	Err                  error
}

func (err *ComponentValidationError) Error() string { return err.Err.Error() }
func (err *ComponentValidationError) Unwrap() error { return err.Err }

type Manager struct {
	storeRoot string
	git       gitRunner
}

func NewManager(dataDirectory string) *Manager {
	return &Manager{
		storeRoot: filepath.Join(dataDirectory, "architecture"),
		git:       gitRunner{},
	}
}

func (manager *Manager) ValidateSourceIsolation(sourceRoot string) error {
	relative, err := filepath.Rel(sourceRoot, manager.storeRoot)
	if err != nil {
		return fmt.Errorf("compare project and private Architecture paths: %w", err)
	}
	if relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))) {
		return errors.New("private Architecture directory must be outside the project folder")
	}
	return nil
}

func (manager *Manager) StorePath(storeID string) (string, error) {
	id, err := uuid.Parse(storeID)
	if err != nil {
		return "", fmt.Errorf("%w: associated store ID is not a UUID", ErrInvalid)
	}
	return filepath.Join(manager.storeRoot, id.String()+".git"), nil
}

// LoadAccepted reads the exact Architecture revision named by accepted. It is
// deliberately read-only: missing repositories or refs remain missing, and no
// other ref or object is considered as a fallback authority.
func (manager *Manager) LoadAccepted(ctx context.Context, storeID string) (Snapshot, error) {
	parsedStoreID, err := uuid.Parse(storeID)
	if err != nil {
		return Snapshot{}, fmt.Errorf("%w: associated store ID is not a UUID", ErrInvalid)
	}
	storePath, _ := manager.StorePath(parsedStoreID.String())

	info, err := os.Stat(storePath)
	if errors.Is(err, os.ErrNotExist) {
		return Snapshot{}, fmt.Errorf("%w: private Architecture location is missing", ErrUnavailable)
	}
	if err != nil {
		return Snapshot{}, fmt.Errorf("%w: inspect private Architecture location: %v", ErrUnavailable, err)
	}
	if !info.IsDir() {
		return Snapshot{}, fmt.Errorf("%w: private Architecture location is not a directory", ErrInvalid)
	}

	bare, err := manager.git.isBare(ctx, storePath)
	if err != nil || !bare {
		return Snapshot{}, fmt.Errorf("%w: private Architecture location is not a compatible bare Git repository", ErrInvalid)
	}
	revision, present, err := manager.git.resolveRef(ctx, storePath, acceptedRef)
	if err != nil {
		return Snapshot{}, fmt.Errorf("%w: observe accepted Architecture: %v", ErrInvalid, err)
	}
	if !present {
		return Snapshot{}, fmt.Errorf("%w: accepted Architecture is missing", ErrUnavailable)
	}

	return manager.load(ctx, storePath, parsedStoreID, revision)
}

// LoadRevision reads and validates one exact committed Architecture revision.
// It shares the accepted loader's parser and validation path, but does not
// observe or assign authority to any ref. Callers must separately establish
// that accepted still names the revision before publishing the snapshot.
func (manager *Manager) LoadRevision(ctx context.Context, base Snapshot, revision string) (Snapshot, error) {
	storePath, err := manager.StorePath(base.storeID.String())
	if err != nil {
		return Snapshot{}, err
	}
	return manager.load(ctx, storePath, base.storeID, revision)
}

// InitializeOrLoad completes a compatible manifest-only bootstrap or loads the
// exact valid revision already named by accepted. It never changes an existing
// accepted ref.
func (manager *Manager) InitializeOrLoad(ctx context.Context, storeID, projectName, sourceHint string) (Snapshot, error) {
	parsedStoreID, err := uuid.Parse(storeID)
	if err != nil {
		return Snapshot{}, fmt.Errorf("%w: associated store ID is not a UUID", ErrInvalid)
	}
	storePath, _ := manager.StorePath(parsedStoreID.String())

	initialized, err := manager.ensureBareRepository(ctx, storePath)
	if err != nil {
		return Snapshot{}, err
	}

	revision, present, err := manager.git.resolveRef(ctx, storePath, acceptedRef)
	if err != nil {
		return Snapshot{}, fmt.Errorf("%w: observe accepted Architecture: %v", ErrInvalid, err)
	}
	if present {
		return manager.load(ctx, storePath, parsedStoreID, revision)
	}
	if !initialized {
		refs, err := manager.git.refs(ctx, storePath)
		if err != nil {
			return Snapshot{}, fmt.Errorf("%w: inspect private repository refs: %v", ErrInvalid, err)
		}
		if len(refs) != 0 {
			return Snapshot{}, fmt.Errorf("%w: accepted Architecture is missing while other refs exist", ErrInvalid)
		}
	}

	manifestBytes, err := marshalManifest(manifest{
		Format:  "workbraid-architecture",
		Version: 1,
		StoreID: parsedStoreID.String(),
		Project: manifestProject{Name: projectName, SourceHint: sourceHint},
	})
	if err != nil {
		return Snapshot{}, fmt.Errorf("%w: prepare Architecture identity: %v", ErrIncomplete, err)
	}
	blob, err := manager.git.writeBlob(ctx, storePath, manifestBytes)
	if err != nil {
		return Snapshot{}, fmt.Errorf("%w: write Architecture identity: %v", ErrIncomplete, err)
	}
	tree, err := manager.git.makeBootstrapTree(ctx, storePath, blob)
	if err != nil {
		return Snapshot{}, fmt.Errorf("%w: create bootstrap tree: %v", ErrIncomplete, err)
	}
	commit, err := manager.git.makeBootstrapCommit(ctx, storePath, tree)
	if err != nil {
		return Snapshot{}, fmt.Errorf("%w: create bootstrap commit: %v", ErrIncomplete, err)
	}
	if err := manager.git.createRef(ctx, storePath, acceptedRef, commit); err != nil {
		observed, nowPresent, observeErr := manager.git.resolveRef(ctx, storePath, acceptedRef)
		if observeErr != nil {
			return Snapshot{}, fmt.Errorf("%w: verify accepted Architecture after update failure: %v", ErrIncomplete, observeErr)
		}
		if !nowPresent {
			return Snapshot{}, fmt.Errorf("%w: accepted Architecture was not created", ErrIncomplete)
		}
		return manager.load(ctx, storePath, parsedStoreID, observed)
	}
	return manager.load(ctx, storePath, parsedStoreID, commit)
}

// ensureBareRepository returns true only when it created the repository.
func (manager *Manager) ensureBareRepository(ctx context.Context, storePath string) (bool, error) {
	info, err := os.Stat(storePath)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(manager.storeRoot, 0o700); err != nil {
			return false, fmt.Errorf("%w: create private Architecture directory: %v", ErrIncomplete, err)
		}
		if err := manager.git.initBare(ctx, storePath); err != nil {
			return false, fmt.Errorf("%w: create private Architecture repository: %v", ErrIncomplete, err)
		}
		if err := os.Chmod(storePath, 0o700); err != nil {
			return false, fmt.Errorf("%w: protect private Architecture repository: %v", ErrIncomplete, err)
		}
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("%w: inspect private Architecture location: %v", ErrIncomplete, err)
	}
	if !info.IsDir() {
		return false, fmt.Errorf("%w: private Architecture location is not a directory", ErrInvalid)
	}
	bare, err := manager.git.isBare(ctx, storePath)
	if err != nil || !bare {
		empty, readErr := directoryEmpty(storePath)
		if readErr != nil {
			return false, fmt.Errorf("%w: inspect incomplete private Architecture repository: %v", ErrIncomplete, readErr)
		}
		if empty {
			if err := manager.git.initBare(ctx, storePath); err != nil {
				return false, fmt.Errorf("%w: complete private Architecture repository: %v", ErrIncomplete, err)
			}
			if err := os.Chmod(storePath, 0o700); err != nil {
				return false, fmt.Errorf("%w: protect private Architecture repository: %v", ErrIncomplete, err)
			}
			return true, nil
		}
		return false, fmt.Errorf("%w: private Architecture repository is not a compatible bare Git repository", ErrInvalid)
	}
	return false, nil
}

func directoryEmpty(path string) (bool, error) {
	directory, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer directory.Close()
	_, err = directory.Readdirnames(1)
	if errors.Is(err, io.EOF) {
		return true, nil
	}
	return false, err
}

func (manager *Manager) load(ctx context.Context, storePath string, expectedStoreID uuid.UUID, revision string) (Snapshot, error) {
	entries, err := manager.git.treeEntries(ctx, storePath, revision)
	if err != nil {
		return Snapshot{}, fmt.Errorf("%w: read accepted Architecture tree: %v", ErrInvalid, err)
	}

	var manifestEntry *treeEntry
	for index := range entries {
		entry := entries[index]
		if entry.Path == "architecture.yaml" {
			if entry.Type != "blob" || (entry.Mode != "100644" && entry.Mode != "100755") {
				return Snapshot{}, fmt.Errorf("%w: architecture.yaml is not an ordinary file", ErrInvalid)
			}
			manifestEntry = &entry
		}
	}
	if manifestEntry == nil {
		return Snapshot{}, fmt.Errorf("%w: architecture.yaml is missing", ErrInvalid)
	}
	contents, err := manager.git.readBlob(ctx, storePath, manifestEntry.Object)
	if err != nil {
		return Snapshot{}, fmt.Errorf("%w: read architecture.yaml: %v", ErrInvalid, err)
	}
	parsed, err := parseManifest(contents)
	if err != nil {
		if errors.Is(err, ErrUnsupported) {
			return Snapshot{}, fmt.Errorf("%w: %v", ErrUnsupported, err)
		}
		return Snapshot{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	manifestStoreID, err := uuid.Parse(parsed.StoreID)
	if err != nil || manifestStoreID != expectedStoreID {
		return Snapshot{}, fmt.Errorf("%w: Architecture identity does not match this project", ErrInvalid)
	}
	componentEntries, diagramEntries, err := acceptedTreeEntries(entries, parsed.Version)
	if err != nil {
		return Snapshot{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	components := make([]component, 0, len(componentEntries))
	componentIDs := make(map[uuid.UUID]struct{}, len(componentEntries))
	for _, entry := range componentEntries {
		contents, err := manager.git.readBlob(ctx, storePath, entry.Object)
		if err != nil {
			return Snapshot{}, fmt.Errorf("%w: read component %q: %v", ErrInvalid, entry.Path, err)
		}
		component, err := parseComponent(entry.Path, contents)
		if err != nil {
			return Snapshot{}, fmt.Errorf("%w: component %q: %v", ErrInvalid, entry.Path, err)
		}
		if _, duplicate := componentIDs[component.id]; duplicate {
			return Snapshot{}, fmt.Errorf("%w: duplicate component ID %s", ErrInvalid, component.id)
		}
		component.mode = entry.Mode
		component.source = append([]byte(nil), contents...)
		componentIDs[component.id] = struct{}{}
		components = append(components, component)
	}
	for _, component := range components {
		for _, relationship := range component.relationships {
			if _, exists := componentIDs[relationship.target]; !exists {
				return Snapshot{}, fmt.Errorf("%w: component %q has a relationship to an unknown component", ErrInvalid, component.path)
			}
		}
	}
	snapshot := Snapshot{storeID: expectedStoreID, revision: revision, formatVersion: parsed.Version, components: components}
	if parsed.Version == 2 {
		diagrams, root, err := manager.loadDiagrams(ctx, storePath, diagramEntries, parsed.RootDiagram, components)
		if err != nil {
			return Snapshot{}, fmt.Errorf("%w: %v", ErrInvalid, err)
		}
		snapshot.rootDiagram = root
		snapshot.diagrams = diagrams
	}
	return snapshot, nil
}

func acceptedTreeEntries(entries []treeEntry, version int) ([]treeEntry, []treeEntry, error) {
	componentsTreePresent := false
	diagramsTreePresent := false
	var componentEntries []treeEntry
	var diagramEntries []treeEntry
	for _, entry := range entries {
		switch {
		case entry.Path == "architecture.yaml":
			// Already checked by the caller.
		case entry.Path == "components" && entry.Type == "tree":
			componentsTreePresent = true
		case strings.HasPrefix(entry.Path, "components/"):
			relative := strings.TrimPrefix(entry.Path, "components/")
			if strings.Contains(relative, "/") || !strings.HasSuffix(relative, ".md") || entry.Type != "blob" || !ordinaryFileMode(entry.Mode) {
				return nil, nil, errors.New("accepted tree contains an invalid component path")
			}
			componentEntries = append(componentEntries, entry)
		case version == 2 && entry.Path == "diagrams" && entry.Type == "tree":
			diagramsTreePresent = true
		case version == 2 && strings.HasPrefix(entry.Path, "diagrams/"):
			relative := strings.TrimPrefix(entry.Path, "diagrams/")
			if strings.Contains(relative, "/") || !strings.HasSuffix(relative, ".yaml") || entry.Type != "blob" || !ordinaryFileMode(entry.Mode) {
				return nil, nil, errors.New("accepted tree contains an invalid Diagram path")
			}
			diagramEntries = append(diagramEntries, entry)
		default:
			return nil, nil, fmt.Errorf("accepted tree contains unsupported path %q", entry.Path)
		}
	}
	if componentsTreePresent && len(componentEntries) == 0 {
		return nil, nil, errors.New("accepted tree contains an empty components directory")
	}
	if version == 2 && (!diagramsTreePresent || len(diagramEntries) == 0) {
		return nil, nil, errors.New("accepted tree must contain one or more Diagrams")
	}
	return componentEntries, diagramEntries, nil
}

func ordinaryFileMode(mode string) bool { return mode == "100644" || mode == "100755" }

func (manager *Manager) loadDiagrams(ctx context.Context, storePath string, entries []treeEntry, rootValue string, components []component) ([]diagram, uuid.UUID, error) {
	root, err := uuid.Parse(rootValue)
	if err != nil {
		return nil, uuid.Nil, errors.New("root_diagram is not a valid UUID")
	}
	componentIDs := make(map[uuid.UUID]struct{}, len(components))
	for _, component := range components {
		componentIDs[component.id] = struct{}{}
	}
	diagrams := make([]diagram, 0, len(entries))
	diagramIDs := make(map[uuid.UUID]struct{}, len(entries))
	for _, entry := range entries {
		contents, err := manager.git.readBlob(ctx, storePath, entry.Object)
		if err != nil {
			return nil, uuid.Nil, fmt.Errorf("read Diagram %q: %v", entry.Path, err)
		}
		parsed, err := parseDiagram(entry.Path, contents)
		if err != nil {
			return nil, uuid.Nil, fmt.Errorf("Diagram %q: %v", entry.Path, err)
		}
		if _, duplicate := diagramIDs[parsed.id]; duplicate {
			return nil, uuid.Nil, fmt.Errorf("duplicate Diagram ID %s", parsed.id)
		}
		diagramIDs[parsed.id] = struct{}{}
		diagrams = append(diagrams, parsed)
	}
	if _, exists := diagramIDs[root]; !exists {
		return nil, uuid.Nil, errors.New("root_diagram does not resolve to a Diagram")
	}

	homeCounts := make(map[uuid.UUID]int, len(components))
	parentCounts := make(map[uuid.UUID]int, len(diagrams))
	children := make(map[uuid.UUID][]uuid.UUID, len(diagrams))
	for _, current := range diagrams {
		seen := make(map[uuid.UUID]struct{}, len(current.appearances))
		for _, appearance := range current.appearances {
			if _, exists := componentIDs[appearance.component]; !exists {
				return nil, uuid.Nil, fmt.Errorf("Diagram %q contains an unknown Component appearance", current.path)
			}
			if _, duplicate := seen[appearance.component]; duplicate {
				return nil, uuid.Nil, fmt.Errorf("Diagram %q contains a Component more than once", current.path)
			}
			seen[appearance.component] = struct{}{}
			if appearance.role == "home" {
				homeCounts[appearance.component]++
			}
			if appearance.hasDetailLink {
				if _, exists := diagramIDs[appearance.detailDiagram]; !exists {
					return nil, uuid.Nil, fmt.Errorf("Diagram %q links to an unknown detail Diagram", current.path)
				}
				parentCounts[appearance.detailDiagram]++
				children[current.id] = append(children[current.id], appearance.detailDiagram)
			}
		}
	}
	for componentID := range componentIDs {
		if homeCounts[componentID] != 1 {
			return nil, uuid.Nil, fmt.Errorf("Component %s must have exactly one home appearance", componentID)
		}
	}
	if parentCounts[root] != 0 {
		return nil, uuid.Nil, errors.New("root Diagram must not have a parent anchor")
	}
	for diagramID := range diagramIDs {
		if diagramID != root && parentCounts[diagramID] != 1 {
			return nil, uuid.Nil, fmt.Errorf("non-root Diagram %s must have exactly one parent anchor", diagramID)
		}
	}
	visiting := make(map[uuid.UUID]bool, len(diagrams))
	visited := make(map[uuid.UUID]bool, len(diagrams))
	var visit func(uuid.UUID) error
	visit = func(id uuid.UUID) error {
		if visiting[id] {
			return errors.New("Diagram hierarchy contains a cycle")
		}
		if visited[id] {
			return nil
		}
		visiting[id] = true
		for _, child := range children[id] {
			if err := visit(child); err != nil {
				return err
			}
		}
		visiting[id] = false
		visited[id] = true
		return nil
	}
	if err := visit(root); err != nil {
		return nil, uuid.Nil, err
	}
	if len(visited) != len(diagrams) {
		return nil, uuid.Nil, errors.New("every Diagram must be reachable from root")
	}
	return diagrams, root, nil
}

// NewComponentChange assigns creation-time identity and filename while leaving
// the accepted snapshot untouched. Existing pending paths participate in the
// collision check because all changes form one candidate Architecture.
func (manager *Manager) NewComponentChange(base Snapshot, changes []ComponentChange, title, description string) ComponentChange {
	used := make(map[string]struct{}, len(base.components)+len(changes))
	for _, component := range base.components {
		used[component.path] = struct{}{}
	}
	for _, change := range changes {
		used[change.Path] = struct{}{}
	}
	slug := componentFilenameSlug(title)
	path := "components/" + slug + ".md"
	for suffix := 2; ; suffix++ {
		if _, exists := used[path]; !exists {
			break
		}
		path = fmt.Sprintf("components/%s-%d.md", slug, suffix)
	}
	return ComponentChange{
		ID:                 uuid.NewString(),
		Title:              title,
		Description:        description,
		Path:               path,
		New:                true,
		TitleChanged:       true,
		DescriptionChanged: true,
	}
}

func componentFilenameSlug(title string) string {
	var result strings.Builder
	separator := false
	for _, character := range strings.ToLower(strings.TrimSpace(title)) {
		switch {
		case character >= 'a' && character <= 'z', character >= '0' && character <= '9':
			if separator && result.Len() > 0 {
				result.WriteByte('-')
			}
			separator = false
			result.WriteRune(character)
		case unicode.IsLetter(character), unicode.IsDigit(character):
			if separator && result.Len() > 0 {
				result.WriteByte('-')
			}
			separator = false
			result.WriteRune(character)
		default:
			separator = true
		}
	}
	value := strings.Trim(result.String(), "-")
	if value == "" {
		return "component"
	}
	return value
}

// ConstructCandidate is the single I2.2 candidate construction and validation
// path. It starts from the exact loaded base tree, writes only changed/new
// blobs, and validates the complete resulting tree through the same loader used
// for accepted Architecture.
func (manager *Manager) ConstructCandidate(ctx context.Context, base Snapshot, changes []ComponentChange) (Candidate, error) {
	storePath, err := manager.StorePath(base.storeID.String())
	if err != nil {
		return Candidate{}, err
	}
	entries, err := manager.git.treeEntries(ctx, storePath, base.revision)
	if err != nil {
		return Candidate{}, fmt.Errorf("construct candidate from accepted base: %w", err)
	}

	byPath := make(map[string]treeEntry, len(entries))
	var manifest treeEntry
	for _, entry := range entries {
		if entry.Path == "architecture.yaml" {
			manifest = entry
		}
		if entry.Type == "blob" {
			byPath[entry.Path] = entry
		}
	}
	if manifest.Path == "" {
		return Candidate{}, fmt.Errorf("%w: architecture identity is missing", ErrInvalid)
	}

	baseByID := make(map[string]component, len(base.components))
	candidateIDs := make(map[string]struct{}, len(base.components)+len(changes))
	for _, component := range base.components {
		baseByID[component.id.String()] = component
		candidateIDs[component.id.String()] = struct{}{}
	}
	for _, change := range changes {
		if change.New {
			candidateIDs[change.ID] = struct{}{}
		}
	}
	seenIDs := make(map[string]struct{}, len(changes))
	for _, change := range changes {
		if _, duplicate := seenIDs[change.ID]; duplicate {
			return Candidate{}, fmt.Errorf("%w: component is changed more than once", ErrInvalid)
		}
		seenIDs[change.ID] = struct{}{}
		if strings.TrimSpace(change.Title) == "" {
			return Candidate{}, &ComponentValidationError{ComponentID: change.ID, Err: ErrTitleRequired}
		}
		if strings.ContainsAny(change.Title, "\r\n") {
			return Candidate{}, &ComponentValidationError{ComponentID: change.ID, Err: ErrTitleOneLine}
		}
		if change.RelationshipsChanged || change.New {
			for relationshipIndex, relationship := range change.Relationships {
				if strings.TrimSpace(relationship.Label) == "" {
					return Candidate{}, &ComponentValidationError{
						ComponentID: change.ID, RelationshipPosition: relationshipIndex + 1, RelationshipField: "label", Err: ErrRelationshipLabelRequired,
					}
				}
				if strings.TrimSpace(relationship.TargetID) == "" {
					return Candidate{}, &ComponentValidationError{
						ComponentID: change.ID, RelationshipPosition: relationshipIndex + 1, RelationshipField: "target", Err: ErrRelationshipTargetRequired,
					}
				}
				if _, err := uuid.Parse(relationship.TargetID); err != nil {
					return Candidate{}, &ComponentValidationError{
						ComponentID: change.ID, RelationshipPosition: relationshipIndex + 1, RelationshipField: "target", Err: ErrRelationshipTargetRequired,
					}
				}
				if _, exists := candidateIDs[relationship.TargetID]; !exists {
					return Candidate{}, &ComponentValidationError{
						ComponentID: change.ID, RelationshipPosition: relationshipIndex + 1, RelationshipField: "target", Err: ErrRelationshipTargetRequired,
					}
				}
			}
		}

		var source []byte
		mode := "100644"
		if change.New {
			if _, exists := baseByID[change.ID]; exists {
				return Candidate{}, fmt.Errorf("%w: new component identity already exists", ErrInvalid)
			}
			if !validNewComponentPath(change.Path) {
				return Candidate{}, fmt.Errorf("%w: new component path is invalid", ErrInvalid)
			}
			if _, exists := byPath[change.Path]; exists {
				return Candidate{}, fmt.Errorf("%w: new component path already exists", ErrInvalid)
			}
			if _, err := uuid.Parse(change.ID); err != nil {
				return Candidate{}, fmt.Errorf("%w: new component identity is invalid", ErrInvalid)
			}
			source, err = newComponentSource(change)
			if err != nil {
				return Candidate{}, fmt.Errorf("serialize new component metadata: %w", err)
			}
		} else {
			accepted, exists := baseByID[change.ID]
			if !exists || accepted.path != change.Path {
				return Candidate{}, fmt.Errorf("%w: changed component does not belong to the accepted base", ErrInvalid)
			}
			mode = accepted.mode
			source, err = editedComponentSource(accepted, change)
			if err != nil {
				return Candidate{}, fmt.Errorf("serialize changed component metadata: %w", err)
			}
			if bytes.Equal(source, accepted.source) {
				continue
			}
		}
		blob, err := manager.git.writeBlob(ctx, storePath, source)
		if err != nil {
			return Candidate{}, fmt.Errorf("write candidate component: %w", err)
		}
		byPath[change.Path] = treeEntry{Mode: mode, Type: "blob", Object: blob, Path: change.Path}
	}

	componentPaths := make([]string, 0, len(byPath))
	for path := range byPath {
		if strings.HasPrefix(path, "components/") {
			componentPaths = append(componentPaths, path)
		}
	}
	sort.Strings(componentPaths)
	var componentTree string
	if len(componentPaths) > 0 {
		var treeSource strings.Builder
		for _, path := range componentPaths {
			entry := byPath[path]
			fmt.Fprintf(&treeSource, "%s blob %s\t%s\n", entry.Mode, entry.Object, strings.TrimPrefix(path, "components/"))
		}
		componentTree, err = manager.git.makeTree(ctx, storePath, []byte(treeSource.String()))
		if err != nil {
			return Candidate{}, fmt.Errorf("construct candidate component tree: %w", err)
		}
	}
	rootSource := fmt.Sprintf("%s blob %s\tarchitecture.yaml\n", manifest.Mode, manifest.Object)
	if componentTree != "" {
		rootSource += "040000 tree " + componentTree + "\tcomponents\n"
	}
	tree, err := manager.git.makeTree(ctx, storePath, []byte(rootSource))
	if err != nil {
		return Candidate{}, fmt.Errorf("construct candidate tree: %w", err)
	}
	snapshot, err := manager.load(ctx, storePath, base.storeID, tree)
	if err != nil {
		return Candidate{}, err
	}
	return Candidate{tree: tree, snapshot: snapshot}, nil
}

// AcceptedRevision observes only the authoritative accepted ref for the
// snapshot's private store. It does not load or fall back to another ref.
func (manager *Manager) AcceptedRevision(ctx context.Context, snapshot Snapshot) (string, bool, error) {
	storePath, err := manager.StorePath(snapshot.storeID.String())
	if err != nil {
		return "", false, err
	}
	return manager.git.resolveRef(ctx, storePath, acceptedRef)
}

// CandidateDiff returns review evidence for the exact base and candidate
// trees. The bytes are predictable presentation, not canonical state.
func (manager *Manager) CandidateDiff(ctx context.Context, base Snapshot, candidate Candidate) ([]byte, error) {
	if base.storeID != candidate.snapshot.storeID {
		return nil, fmt.Errorf("candidate belongs to another Architecture store")
	}
	storePath, err := manager.StorePath(base.storeID.String())
	if err != nil {
		return nil, err
	}
	diff, err := manager.git.diffTrees(ctx, storePath, base.revision, candidate.tree)
	if err != nil {
		return nil, err
	}
	return []byte(presentUnifiedDiff(diff)), nil
}

func presentUnifiedDiff(diff []byte) string {
	var presented strings.Builder
	for len(diff) > 0 {
		value, size := utf8.DecodeRune(diff)
		if value == utf8.RuneError && size == 1 {
			fmt.Fprintf(&presented, "\\x%02x", diff[0])
			diff = diff[1:]
			continue
		}
		diff = diff[size:]
		switch {
		case value == '\n' || value == '\t':
			presented.WriteRune(value)
		case value == '\\':
			presented.WriteString("\\\\")
		case unicode.IsPrint(value):
			presented.WriteRune(value)
		case value <= 0xff:
			fmt.Fprintf(&presented, "\\x%02x", value)
		default:
			fmt.Fprintf(&presented, "\\u{%x}", value)
		}
	}
	return presented.String()
}

// CreateSuccessor creates a non-canonical commit object. Authority changes
// only if AdvanceAccepted subsequently succeeds.
func (manager *Manager) CreateSuccessor(ctx context.Context, base Snapshot, candidate Candidate) (string, error) {
	if base.storeID != candidate.snapshot.storeID {
		return "", fmt.Errorf("candidate belongs to another Architecture store")
	}
	storePath, err := manager.StorePath(base.storeID.String())
	if err != nil {
		return "", err
	}
	return manager.git.makeSuccessorCommit(ctx, storePath, candidate.tree, base.revision)
}

// AdvanceAccepted performs the mandatory compare-and-swap. A nil result is the
// acceptance success boundary; callers observe the ref only to classify a
// failed update, never by parsing Git's diagnostic text.
func (manager *Manager) AdvanceAccepted(ctx context.Context, base Snapshot, successor string) error {
	storePath, err := manager.StorePath(base.storeID.String())
	if err != nil {
		return err
	}
	return manager.git.updateRef(ctx, storePath, acceptedRef, successor, base.revision)
}

func validNewComponentPath(path string) bool {
	if !strings.HasPrefix(path, "components/") || !strings.HasSuffix(path, ".md") {
		return false
	}
	relative := strings.TrimPrefix(path, "components/")
	return relative != "" && !strings.Contains(relative, "/")
}

func newComponentSource(change ComponentChange) ([]byte, error) {
	frontmatter, err := marshalComponentFrontmatter(change.ID, change.Relationships)
	if err != nil {
		return nil, err
	}
	return []byte(frontmatter + "# " + escapeMarkdownTitle(change.Title) + "\n" + change.Description), nil
}

func editedComponentSource(accepted component, change ComponentChange) ([]byte, error) {
	heading := accepted.source[accepted.headingStart:accepted.headingEnd]
	if change.TitleChanged && change.Title != accepted.title {
		heading = replacementHeading(accepted, change.Title)
	}
	body := accepted.body
	if change.DescriptionChanged {
		body = []byte(change.Description)
	}
	source := make([]byte, 0, len(accepted.source)+len(change.Title)+len(change.Description))
	source = append(source, accepted.source[:accepted.headingStart]...)
	source = append(source, heading...)
	source = append(source, body...)
	if change.RelationshipsChanged {
		frontmatter, err := marshalComponentFrontmatter(change.ID, change.Relationships)
		if err != nil {
			return nil, err
		}
		rewritten := make([]byte, 0, len(frontmatter)+len(source)-accepted.markdownStart)
		rewritten = append(rewritten, frontmatter...)
		rewritten = append(rewritten, source[accepted.markdownStart:]...)
		return rewritten, nil
	}
	return source, nil
}

func marshalComponentFrontmatter(id string, relationships []AuthoringRelationship) (string, error) {
	metadata := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	metadata.Content = append(metadata.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "id"},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: id, Style: yaml.DoubleQuotedStyle},
	)
	if len(relationships) > 0 {
		sequence := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, relationship := range relationships {
			item := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			item.Content = append(item.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "target"},
				&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: relationship.TargetID, Style: yaml.DoubleQuotedStyle},
				&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "label"},
				&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: relationship.Label, Style: yaml.DoubleQuotedStyle},
			)
			sequence.Content = append(sequence.Content, item)
		}
		metadata.Content = append(metadata.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "relationships"},
			sequence,
		)
	}
	var contents bytes.Buffer
	encoder := yaml.NewEncoder(&contents)
	encoder.SetIndent(2)
	if err := encoder.Encode(metadata); err != nil {
		return "", err
	}
	return "---\n" + contents.String() + "---\n", nil
}

func replacementHeading(accepted component, title string) []byte {
	lineEnding := headingLineEnding(accepted.source[accepted.headingStart:accepted.headingEnd])
	escaped := escapeMarkdownTitle(title)
	if accepted.headingStyle == headingSetext {
		return []byte(escaped + lineEnding + "=" + lineEnding)
	}
	return []byte("# " + escaped + lineEnding)
}

func headingLineEnding(block []byte) string {
	if bytes.Contains(block, []byte("\r\n")) {
		return "\r\n"
	}
	if bytes.Contains(block, []byte("\n")) {
		return "\n"
	}
	return "\n"
}

func escapeMarkdownTitle(title string) string {
	var escaped strings.Builder
	for _, character := range title {
		if character == '&' {
			escaped.WriteString("&amp;")
			continue
		}
		if character < utf8.RuneSelf && util.IsPunct(byte(character)) {
			escaped.WriteByte('\\')
		}
		escaped.WriteRune(character)
	}
	return escaped.String()
}

type componentFrontmatter struct {
	ID            string                      `yaml:"id"`
	Relationships []componentRelationshipYAML `yaml:"relationships,omitempty"`
}

type componentRelationshipYAML struct {
	Target string `yaml:"target"`
	Label  string `yaml:"label"`
}

var componentMarkdown = goldmark.New(
	goldmark.WithExtensions(
		extension.Table,
		extension.TaskList,
		extension.Strikethrough,
		extension.Linkify,
	),
)

func parseComponent(path string, contents []byte) (component, error) {
	if !utf8.Valid(contents) {
		return component{}, errors.New("source is not valid UTF-8")
	}
	frontmatter, markdownSource, err := splitComponentFrontmatter(contents)
	if err != nil {
		return component{}, err
	}
	metadata, err := parseComponentFrontmatter(frontmatter)
	if err != nil {
		return component{}, err
	}
	componentID, err := uuid.Parse(metadata.ID)
	if err != nil {
		return component{}, errors.New("id is not a valid UUID")
	}

	document := componentMarkdown.Parser().Parse(text.NewReader(markdownSource))
	first := document.FirstChild()
	heading, ok := first.(*ast.Heading)
	if !ok || heading.Level != 1 {
		return component{}, errors.New("first Markdown block must be a level-one heading")
	}
	titleBytes, err := plainHeadingText(heading, markdownSource)
	if err != nil {
		return component{}, err
	}
	title := strings.TrimSpace(string(titleBytes))
	if title == "" {
		return component{}, errors.New("level-one heading title is empty")
	}
	bodyStart, err := headingBlockEnd(markdownSource, heading)
	if err != nil {
		return component{}, err
	}

	relationships := make([]componentRelationship, len(metadata.Relationships))
	for index, relationship := range metadata.Relationships {
		target, err := uuid.Parse(relationship.Target)
		if err != nil {
			return component{}, fmt.Errorf("relationship %d target is not a valid UUID", index+1)
		}
		relationships[index] = componentRelationship{target: target, label: relationship.Label}
	}
	body := append([]byte(nil), markdownSource[bodyStart:]...)
	headingStart, style, err := headingBlockStart(markdownSource, heading)
	if err != nil {
		return component{}, err
	}
	markdownStart := len(contents) - len(markdownSource)
	return component{
		id:            componentID,
		path:          path,
		title:         title,
		body:          body,
		relationships: relationships,
		markdownStart: markdownStart,
		headingStart:  markdownStart + headingStart,
		headingEnd:    markdownStart + bodyStart,
		headingStyle:  style,
	}, nil
}

func plainHeadingText(heading *ast.Heading, source []byte) ([]byte, error) {
	var result bytes.Buffer
	err := ast.Walk(heading, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch value := node.(type) {
		case *ast.CodeSpan:
			result.Write(value.Text(source))
			return ast.WalkSkipChildren, nil
		case *ast.AutoLink:
			result.Write(value.Label(source))
			return ast.WalkSkipChildren, nil
		case *ast.RawHTML:
			result.Write(value.Segments.Value(source))
			return ast.WalkSkipChildren, nil
		case *ast.Text:
			result.Write(resolveMarkdownText(value.Value(source)))
			if value.SoftLineBreak() || value.HardLineBreak() {
				result.WriteByte(' ')
			}
		case *ast.String:
			if value.IsCode() || value.IsRaw() {
				result.Write(value.Value)
			} else {
				result.Write(resolveMarkdownText(value.Value))
			}
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return nil, fmt.Errorf("read level-one heading title: %w", err)
	}
	return result.Bytes(), nil
}

func resolveMarkdownText(source []byte) []byte {
	value := util.UnescapePunctuations(source)
	value = util.ResolveNumericReferences(value)
	return util.ResolveEntityNames(value)
}

func splitComponentFrontmatter(contents []byte) ([]byte, []byte, error) {
	firstLine, next, ok := sourceLine(contents, 0)
	if !ok || string(firstLine) != "---" {
		return nil, nil, errors.New("required YAML frontmatter is missing")
	}
	frontmatterStart := next
	for offset := next; offset <= len(contents); {
		lineStart := offset
		line, following, exists := sourceLine(contents, offset)
		if !exists {
			break
		}
		if string(line) == "---" {
			return contents[frontmatterStart:lineStart], contents[following:], nil
		}
		if following <= offset {
			break
		}
		offset = following
	}
	return nil, nil, errors.New("YAML frontmatter closing delimiter is missing")
}

func sourceLine(source []byte, offset int) ([]byte, int, bool) {
	if offset < 0 || offset > len(source) || offset == len(source) {
		return nil, offset, false
	}
	end := bytes.IndexByte(source[offset:], '\n')
	if end < 0 {
		line := source[offset:]
		return bytes.TrimSuffix(line, []byte{'\r'}), len(source), true
	}
	line := source[offset : offset+end]
	return bytes.TrimSuffix(line, []byte{'\r'}), offset + end + 1, true
}

func parseComponentFrontmatter(contents []byte) (componentFrontmatter, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(contents))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		return componentFrontmatter{}, fmt.Errorf("parse frontmatter: %w", err)
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return componentFrontmatter{}, errors.New("frontmatter contains multiple YAML documents")
		}
		return componentFrontmatter{}, fmt.Errorf("parse frontmatter: %w", err)
	}
	if len(document.Content) != 1 {
		return componentFrontmatter{}, errors.New("frontmatter must contain one mapping")
	}
	if err := validateComponentFrontmatterYAML(document.Content[0]); err != nil {
		return componentFrontmatter{}, err
	}
	var value componentFrontmatter
	if err := document.Content[0].Decode(&value); err != nil {
		return componentFrontmatter{}, fmt.Errorf("parse frontmatter: %w", err)
	}
	return value, nil
}

func validateComponentFrontmatterYAML(root *yaml.Node) error {
	if root.Kind != yaml.MappingNode || root.ShortTag() != "!!map" {
		return errors.New("frontmatter must contain a mapping")
	}
	required := map[string]bool{"id": false}
	seenRelationships := false
	for index := 0; index < len(root.Content); index += 2 {
		key := root.Content[index]
		value := root.Content[index+1]
		if key.Kind != yaml.ScalarNode || key.ShortTag() != "!!str" {
			return errors.New("frontmatter field names must be strings")
		}
		switch key.Value {
		case "id":
			if required["id"] {
				return errors.New("frontmatter contains duplicate field \"id\"")
			}
			required["id"] = true
			if value.Kind != yaml.ScalarNode || value.ShortTag() != "!!str" {
				return errors.New("frontmatter field id must be a string")
			}
		case "relationships":
			if seenRelationships {
				return errors.New("frontmatter contains duplicate field \"relationships\"")
			}
			seenRelationships = true
			if value.Kind != yaml.SequenceNode || value.ShortTag() != "!!seq" {
				return errors.New("frontmatter field relationships must be a sequence")
			}
			for relationshipIndex, relationship := range value.Content {
				if err := validateRelationshipYAML(relationship); err != nil {
					return fmt.Errorf("relationship %d: %w", relationshipIndex+1, err)
				}
			}
		default:
			return fmt.Errorf("frontmatter contains unknown field %q", key.Value)
		}
	}
	if !required["id"] {
		return errors.New("frontmatter is missing field \"id\"")
	}
	return nil
}

func validateRelationshipYAML(root *yaml.Node) error {
	if root.Kind != yaml.MappingNode || root.ShortTag() != "!!map" {
		return errors.New("item must be a mapping")
	}
	required := map[string]bool{"target": false, "label": false}
	for index := 0; index < len(root.Content); index += 2 {
		key := root.Content[index]
		value := root.Content[index+1]
		if key.Kind != yaml.ScalarNode || key.ShortTag() != "!!str" {
			return errors.New("field names must be strings")
		}
		if _, known := required[key.Value]; !known {
			return fmt.Errorf("contains unknown field %q", key.Value)
		}
		if required[key.Value] {
			return fmt.Errorf("contains duplicate field %q", key.Value)
		}
		required[key.Value] = true
		if value.Kind != yaml.ScalarNode || value.ShortTag() != "!!str" {
			return fmt.Errorf("field %s must be a string", key.Value)
		}
	}
	for field, present := range required {
		if !present {
			return fmt.Errorf("is missing field %q", field)
		}
	}
	label := relationshipField(root, "label")
	if strings.TrimSpace(label) == "" {
		return errors.New("label is empty")
	}
	return nil
}

func relationshipField(root *yaml.Node, name string) string {
	for index := 0; index < len(root.Content); index += 2 {
		if root.Content[index].Value == name {
			return root.Content[index+1].Value
		}
	}
	return ""
}

func headingBlockEnd(source []byte, heading *ast.Heading) (int, error) {
	lines := heading.Lines()
	if lines == nil || lines.Len() == 0 {
		return 0, errors.New("level-one heading has no source location")
	}
	first := lines.At(0)
	lineStart := bytes.LastIndexByte(source[:first.Start], '\n') + 1
	lineEnd := endOfSourceLine(source, first.Stop)
	line := bytes.TrimLeft(source[lineStart:lineEnd], " \t")
	if (len(line) == 1 && line[0] == '#') || (len(line) > 1 && line[0] == '#' && (line[1] == ' ' || line[1] == '\t' || line[1] == '\r' || line[1] == '\n')) {
		return lineEnd, nil
	}
	last := lines.At(lines.Len() - 1)
	contentEnd := endOfSourceLine(source, last.Stop)
	underlineEnd := endOfLineStartingAt(source, contentEnd)
	if underlineEnd == contentEnd {
		return 0, errors.New("Setext level-one heading is incomplete")
	}
	return underlineEnd, nil
}

func headingBlockStart(source []byte, heading *ast.Heading) (int, headingStyle, error) {
	lines := heading.Lines()
	if lines == nil || lines.Len() == 0 {
		return 0, headingATX, errors.New("level-one heading has no source location")
	}
	first := lines.At(0)
	lineStart := bytes.LastIndexByte(source[:first.Start], '\n') + 1
	lineEnd := endOfSourceLine(source, first.Stop)
	line := bytes.TrimLeft(source[lineStart:lineEnd], " \t")
	if (len(line) == 1 && line[0] == '#') || (len(line) > 1 && line[0] == '#' && (line[1] == ' ' || line[1] == '\t' || line[1] == '\r' || line[1] == '\n')) {
		return lineStart, headingATX, nil
	}
	return lineStart, headingSetext, nil
}

func endOfSourceLine(source []byte, offset int) int {
	if offset > len(source) {
		return len(source)
	}
	if offset > 0 && source[offset-1] == '\n' {
		return offset
	}
	if newline := bytes.IndexByte(source[offset:], '\n'); newline >= 0 {
		return offset + newline + 1
	}
	return len(source)
}

func endOfLineStartingAt(source []byte, offset int) int {
	if offset >= len(source) {
		return len(source)
	}
	if newline := bytes.IndexByte(source[offset:], '\n'); newline >= 0 {
		return offset + newline + 1
	}
	return len(source)
}

type manifest struct {
	Format      string          `yaml:"format"`
	Version     int             `yaml:"version"`
	StoreID     string          `yaml:"store_id"`
	Project     manifestProject `yaml:"project"`
	RootDiagram string          `yaml:"root_diagram,omitempty"`
}

type manifestProject struct {
	Name       string `yaml:"name"`
	SourceHint string `yaml:"source_hint"`
}

func marshalManifest(value manifest) ([]byte, error) {
	if err := validateManifest(value); err != nil {
		return nil, err
	}
	return yaml.Marshal(value)
}

func parseManifest(contents []byte) (manifest, error) {
	if !utf8.Valid(contents) {
		return manifest{}, errors.New("architecture.yaml is not valid UTF-8")
	}
	decoder := yaml.NewDecoder(bytes.NewReader(contents))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		return manifest{}, fmt.Errorf("parse architecture.yaml: %w", err)
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return manifest{}, errors.New("architecture.yaml contains multiple YAML documents")
		}
		return manifest{}, fmt.Errorf("parse architecture.yaml: %w", err)
	}
	if len(document.Content) != 1 {
		return manifest{}, errors.New("architecture.yaml must contain one mapping")
	}
	version, err := manifestVersion(document.Content[0])
	if err != nil {
		return manifest{}, err
	}
	if version != 1 && version != 2 {
		return manifest{}, fmt.Errorf("%w: unsupported Architecture format version", ErrUnsupported)
	}
	if err := validateManifestYAML(document.Content[0], version); err != nil {
		return manifest{}, err
	}
	var value manifest
	if err := document.Content[0].Decode(&value); err != nil {
		return manifest{}, fmt.Errorf("parse architecture.yaml: %w", err)
	}
	if err := validateManifest(value); err != nil {
		return manifest{}, err
	}
	return value, nil
}

func manifestVersion(root *yaml.Node) (int, error) {
	if root.Kind != yaml.MappingNode || root.ShortTag() != "!!map" {
		return 0, errors.New("architecture.yaml must contain a mapping")
	}
	found := false
	version := 0
	for index := 0; index < len(root.Content); index += 2 {
		key := root.Content[index]
		value := root.Content[index+1]
		if key.Kind == yaml.ScalarNode && key.ShortTag() == "!!str" && key.Value == "version" {
			if found {
				return 0, errors.New("architecture.yaml contains duplicate field \"version\"")
			}
			found = true
			if value.Kind != yaml.ScalarNode || value.ShortTag() != "!!int" {
				return 0, errors.New("architecture.yaml field version must be an integer")
			}
			if err := value.Decode(&version); err != nil {
				return 0, errors.New("architecture.yaml field version must be an integer")
			}
		}
	}
	if !found {
		return 0, errors.New("architecture.yaml is missing field \"version\"")
	}
	return version, nil
}

func validateManifestYAML(root *yaml.Node, version int) error {
	if root.Kind != yaml.MappingNode || root.ShortTag() != "!!map" {
		return errors.New("architecture.yaml must contain a mapping")
	}
	required := map[string]bool{
		"format": false, "version": false, "store_id": false, "project": false,
	}
	if version == 2 {
		required["root_diagram"] = false
	}
	for index := 0; index < len(root.Content); index += 2 {
		key := root.Content[index]
		value := root.Content[index+1]
		if key.Kind != yaml.ScalarNode || key.ShortTag() != "!!str" {
			return errors.New("architecture.yaml field names must be strings")
		}
		if _, known := required[key.Value]; !known {
			return fmt.Errorf("architecture.yaml contains unknown field %q", key.Value)
		}
		if required[key.Value] {
			return fmt.Errorf("architecture.yaml contains duplicate field %q", key.Value)
		}
		required[key.Value] = true
		switch key.Value {
		case "format", "store_id", "root_diagram":
			if value.Kind != yaml.ScalarNode || value.ShortTag() != "!!str" {
				return fmt.Errorf("architecture.yaml field %s must be a string", key.Value)
			}
		case "version":
			if value.Kind != yaml.ScalarNode || value.ShortTag() != "!!int" {
				return errors.New("architecture.yaml field version must be an integer")
			}
		case "project":
			if err := validateManifestProjectYAML(value); err != nil {
				return err
			}
		}
	}
	for field, present := range required {
		if !present {
			return fmt.Errorf("architecture.yaml is missing field %q", field)
		}
	}
	return nil
}

func validateManifestProjectYAML(project *yaml.Node) error {
	if project.Kind != yaml.MappingNode || project.ShortTag() != "!!map" {
		return errors.New("architecture.yaml field project must be a mapping")
	}
	required := map[string]bool{"name": false, "source_hint": false}
	for index := 0; index < len(project.Content); index += 2 {
		key := project.Content[index]
		value := project.Content[index+1]
		if key.Kind != yaml.ScalarNode || key.ShortTag() != "!!str" {
			return errors.New("architecture.yaml project field names must be strings")
		}
		if _, known := required[key.Value]; !known {
			return fmt.Errorf("architecture.yaml project contains unknown field %q", key.Value)
		}
		if required[key.Value] {
			return fmt.Errorf("architecture.yaml project contains duplicate field %q", key.Value)
		}
		required[key.Value] = true
		if value.Kind != yaml.ScalarNode || value.ShortTag() != "!!str" {
			return fmt.Errorf("architecture.yaml field project.%s must be a string", key.Value)
		}
	}
	for field, present := range required {
		if !present {
			return fmt.Errorf("architecture.yaml project is missing field %q", field)
		}
	}
	return nil
}

func validateManifest(value manifest) error {
	if value.Format != "workbraid-architecture" {
		return fmt.Errorf("%w: unsupported Architecture format", ErrUnsupported)
	}
	if value.Version != 1 && value.Version != 2 {
		return fmt.Errorf("%w: unsupported Architecture format version", ErrUnsupported)
	}
	if _, err := uuid.Parse(value.StoreID); err != nil {
		return errors.New("store_id is not a valid UUID")
	}
	if strings.TrimSpace(value.Project.Name) == "" {
		return errors.New("project.name is empty")
	}
	if strings.TrimSpace(value.Project.SourceHint) == "" {
		return errors.New("project.source_hint is empty")
	}
	if value.Version == 1 && value.RootDiagram != "" {
		return errors.New("format v1 must not contain root_diagram")
	}
	if value.Version == 2 {
		if _, err := uuid.Parse(value.RootDiagram); err != nil {
			return errors.New("root_diagram is not a valid UUID")
		}
	}
	return nil
}

type diagramYAML struct {
	ID          string                  `yaml:"id"`
	Title       string                  `yaml:"title"`
	Appearances []diagramAppearanceYAML `yaml:"appearances,omitempty"`
}

type diagramAppearanceYAML struct {
	Component     string `yaml:"component"`
	Role          string `yaml:"role"`
	DetailDiagram string `yaml:"detail_diagram,omitempty"`
}

func parseDiagram(path string, contents []byte) (diagram, error) {
	if !utf8.Valid(contents) {
		return diagram{}, errors.New("Diagram is not valid UTF-8")
	}
	decoder := yaml.NewDecoder(bytes.NewReader(contents))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		return diagram{}, fmt.Errorf("parse YAML: %w", err)
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return diagram{}, errors.New("Diagram contains multiple YAML documents")
		}
		return diagram{}, fmt.Errorf("parse YAML: %w", err)
	}
	if len(document.Content) != 1 {
		return diagram{}, errors.New("Diagram must contain one mapping")
	}
	if err := validateDiagramYAML(document.Content[0]); err != nil {
		return diagram{}, err
	}
	var value diagramYAML
	if err := document.Content[0].Decode(&value); err != nil {
		return diagram{}, fmt.Errorf("parse YAML: %w", err)
	}
	id, err := uuid.Parse(value.ID)
	if err != nil {
		return diagram{}, errors.New("Diagram id is not a valid UUID")
	}
	if strings.TrimSpace(value.Title) == "" {
		return diagram{}, errors.New("Diagram title is empty")
	}
	result := diagram{id: id, path: path, title: value.Title, appearances: make([]diagramAppearance, len(value.Appearances))}
	for index, item := range value.Appearances {
		componentID, err := uuid.Parse(item.Component)
		if err != nil {
			return diagram{}, fmt.Errorf("appearance %d Component is not a valid UUID", index+1)
		}
		if item.Role != "home" && item.Role != "reference" {
			return diagram{}, fmt.Errorf("appearance %d role must be home or reference", index+1)
		}
		appearance := diagramAppearance{component: componentID, role: item.Role}
		if item.DetailDiagram != "" {
			if item.Role != "home" {
				return diagram{}, fmt.Errorf("appearance %d reference cannot link to a detail Diagram", index+1)
			}
			detailID, err := uuid.Parse(item.DetailDiagram)
			if err != nil {
				return diagram{}, fmt.Errorf("appearance %d detail_diagram is not a valid UUID", index+1)
			}
			appearance.detailDiagram = detailID
			appearance.hasDetailLink = true
		}
		result.appearances[index] = appearance
	}
	return result, nil
}

func validateDiagramYAML(root *yaml.Node) error {
	if root.Kind != yaml.MappingNode || root.ShortTag() != "!!map" {
		return errors.New("Diagram must contain a mapping")
	}
	required := map[string]bool{"id": false, "title": false}
	seenAppearances := false
	for index := 0; index < len(root.Content); index += 2 {
		key := root.Content[index]
		value := root.Content[index+1]
		if key.Kind != yaml.ScalarNode || key.ShortTag() != "!!str" {
			return errors.New("Diagram field names must be strings")
		}
		if key.Value != "id" && key.Value != "title" && key.Value != "appearances" {
			return fmt.Errorf("Diagram contains unknown field %q", key.Value)
		}
		if key.Value == "appearances" {
			if seenAppearances {
				return errors.New("Diagram contains duplicate field \"appearances\"")
			}
			seenAppearances = true
			if value.Kind != yaml.SequenceNode || value.ShortTag() != "!!seq" {
				return errors.New("Diagram field appearances must be a sequence")
			}
			for appearanceIndex, appearance := range value.Content {
				if err := validateDiagramAppearanceYAML(appearance, appearanceIndex+1); err != nil {
					return err
				}
			}
			continue
		}
		if required[key.Value] {
			return fmt.Errorf("Diagram contains duplicate field %q", key.Value)
		}
		required[key.Value] = true
		if value.Kind != yaml.ScalarNode || value.ShortTag() != "!!str" {
			return fmt.Errorf("Diagram field %s must be a string", key.Value)
		}
	}
	for field, present := range required {
		if !present {
			return fmt.Errorf("Diagram is missing field %q", field)
		}
	}
	return nil
}

func validateDiagramAppearanceYAML(value *yaml.Node, position int) error {
	if value.Kind != yaml.MappingNode || value.ShortTag() != "!!map" {
		return fmt.Errorf("Diagram appearance %d must be a mapping", position)
	}
	required := map[string]bool{"component": false, "role": false}
	seenDetail := false
	for index := 0; index < len(value.Content); index += 2 {
		key := value.Content[index]
		field := value.Content[index+1]
		if key.Kind != yaml.ScalarNode || key.ShortTag() != "!!str" {
			return fmt.Errorf("Diagram appearance %d field names must be strings", position)
		}
		if key.Value != "component" && key.Value != "role" && key.Value != "detail_diagram" {
			return fmt.Errorf("Diagram appearance %d contains unknown field %q", position, key.Value)
		}
		if key.Value == "detail_diagram" {
			if seenDetail {
				return fmt.Errorf("Diagram appearance %d contains duplicate field %q", position, key.Value)
			}
			seenDetail = true
		} else {
			if required[key.Value] {
				return fmt.Errorf("Diagram appearance %d contains duplicate field %q", position, key.Value)
			}
			required[key.Value] = true
		}
		if field.Kind != yaml.ScalarNode || field.ShortTag() != "!!str" {
			return fmt.Errorf("Diagram appearance %d field %s must be a string", position, key.Value)
		}
		if key.Value == "detail_diagram" && field.Value == "" {
			return fmt.Errorf("Diagram appearance %d field detail_diagram must not be empty", position)
		}
	}
	for field, present := range required {
		if !present {
			return fmt.Errorf("Diagram appearance %d is missing field %q", position, field)
		}
	}
	return nil
}
