package architecture

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"go.yaml.in/yaml/v4"
)

const reviewsRefPrefix = "refs/workbraid/reviews/"

var (
	ErrReviewRequestInvalid = errors.New("review submission is invalid")
	ErrReviewAnchorInvalid  = errors.New("review anchor is invalid")
	ErrReviewInvalidated    = errors.New("review binding was invalidated")
	ErrReviewNotAllowed     = errors.New("review submission is not allowed")
)

type ReviewBinding struct {
	BaseRevision  string `json:"base_revision" yaml:"base_revision"`
	CandidateTree string `json:"candidate_tree" yaml:"candidate_tree"`
	Generation    uint64 `json:"generation" yaml:"generation"`
}

// ReviewAnchorValidationError identifies the submitted comment whose exact
// reviewed-snapshot anchor failed validation. CommentIndex is one-based for
// direct use by browser and agent clients.
type ReviewAnchorValidationError struct {
	CommentIndex int
	Reason       string
}

func (err *ReviewAnchorValidationError) Error() string {
	return fmt.Sprintf("review anchor is invalid: comment %d: %s", err.CommentIndex, err.Reason)
}

func (err *ReviewAnchorValidationError) Unwrap() error { return ErrReviewAnchorInvalid }

type ReviewAnchor struct {
	Kind              string `json:"kind" yaml:"kind"`
	Side              string `json:"side,omitempty" yaml:"side,omitempty"`
	ComponentID       string `json:"component_id,omitempty" yaml:"component_id,omitempty"`
	DiagramID         string `json:"diagram_id,omitempty" yaml:"diagram_id,omitempty"`
	Aspect            string `json:"aspect,omitempty" yaml:"aspect,omitempty"`
	DetailDiagramID   string `json:"detail_diagram_id,omitempty" yaml:"detail_diagram_id,omitempty"`
	SourceComponentID string `json:"source_component_id,omitempty" yaml:"source_component_id,omitempty"`
	TargetComponentID string `json:"target_component_id,omitempty" yaml:"target_component_id,omitempty"`
	Label             string `json:"label,omitempty" yaml:"label,omitempty"`
	Occurrence        int    `json:"occurrence,omitempty" yaml:"occurrence,omitempty"`
	StartLine         int    `json:"start_line,omitempty" yaml:"start_line,omitempty"`
	EndLine           int    `json:"end_line,omitempty" yaml:"end_line,omitempty"`
}

type ReviewCommentInput struct {
	Body   string       `json:"body"`
	Anchor ReviewAnchor `json:"anchor"`
}

type ReviewSubmissionInput struct {
	ChangeSetID   string
	ReviewedState string
	Binding       ReviewBinding
	Verdict       string
	Author        string
	Body          string
	Comments      []ReviewCommentInput
	SubmittedAt   time.Time
}

type ReviewComment struct {
	ID     string       `json:"id"`
	Body   string       `json:"body"`
	Anchor ReviewAnchor `json:"anchor"`
}

type ReviewSubmission struct {
	ID             string          `json:"id"`
	ChangeSetID    string          `json:"change_set_id"`
	ReviewedState  string          `json:"reviewed_state"`
	Binding        ReviewBinding   `json:"binding"`
	Verdict        string          `json:"verdict"`
	Author         string          `json:"author"`
	SubmittedAt    time.Time       `json:"submitted_at"`
	Body           string          `json:"body"`
	Comments       []ReviewComment `json:"comments"`
	RefObject      string          `json:"-"`
	ReviewedChange ChangeSet       `json:"-"`
}

type UnavailableReview struct {
	ChangeSetID string `json:"change_set_id,omitempty"`
	ReviewID    string `json:"review_id,omitempty"`
	Ref         string `json:"ref"`
	Reason      string `json:"reason"`
}

type reviewMetadata struct {
	Format        string        `yaml:"format"`
	Version       int           `yaml:"version"`
	ID            string        `yaml:"id"`
	ChangeSetID   string        `yaml:"change_set_id"`
	ReviewedState string        `yaml:"reviewed_state"`
	Binding       ReviewBinding `yaml:"binding"`
	Verdict       string        `yaml:"verdict"`
	Author        string        `yaml:"author"`
	SubmittedAt   string        `yaml:"submitted_at"`
	Comments      []string      `yaml:"comments"`
}

type reviewCommentMetadata struct {
	Format  string       `yaml:"format"`
	Version int          `yaml:"version"`
	ID      string       `yaml:"id"`
	Anchor  ReviewAnchor `yaml:"anchor"`
}

func reviewRef(changeSetID, reviewID string) string {
	return reviewsRefPrefix + changeSetID + "/" + reviewID
}

func (manager *Manager) SubmitReview(ctx context.Context, storeID string, input ReviewSubmissionInput) (ReviewSubmission, error) {
	if !canonicalUUID(input.ChangeSetID) || !validObjectID(input.ReviewedState) || !validReviewBinding(input.Binding) {
		return ReviewSubmission{}, ErrReviewRequestInvalid
	}
	author := strings.TrimSpace(input.Author)
	if author == "" || strings.ContainsAny(author, "\r\n") || !utf8.ValidString(author) || !utf8.ValidString(input.Body) {
		return ReviewSubmission{}, ErrReviewRequestInvalid
	}
	if input.Verdict != "comment" && input.Verdict != "approve" && input.Verdict != "request_changes" {
		return ReviewSubmission{}, ErrReviewRequestInvalid
	}
	if input.Verdict == "comment" && strings.TrimSpace(input.Body) == "" && len(input.Comments) == 0 {
		return ReviewSubmission{}, ErrReviewRequestInvalid
	}
	storePath, err := manager.StorePath(storeID)
	if err != nil {
		return ReviewSubmission{}, err
	}
	reviewed, err := manager.loadChangeSet(ctx, storePath, storeID, "active", input.ChangeSetID, "", input.ReviewedState)
	if err != nil || reviewed.Review == nil || reviewed.Candidate == nil {
		return ReviewSubmission{}, ErrReviewInvalidated
	}
	if reviewed.Review.BaseRevision != input.Binding.BaseRevision || reviewed.Review.CandidateTree != input.Binding.CandidateTree || reviewed.Review.Generation != input.Binding.Generation {
		return ReviewSubmission{}, ErrReviewInvalidated
	}
	comments := make([]ReviewComment, len(input.Comments))
	for index, value := range input.Comments {
		if !utf8.ValidString(value.Body) || strings.TrimSpace(value.Body) == "" {
			return ReviewSubmission{}, ErrReviewRequestInvalid
		}
		if err := validateReviewAnchor(reviewed, value.Anchor); err != nil {
			return ReviewSubmission{}, &ReviewAnchorValidationError{CommentIndex: index + 1, Reason: err.Error()}
		}
		comments[index] = ReviewComment{ID: uuid.NewString(), Body: value.Body, Anchor: value.Anchor}
	}
	submittedAt := input.SubmittedAt.UTC()
	if submittedAt.IsZero() {
		submittedAt = time.Now().UTC()
	}
	submittedAt = submittedAt.Truncate(time.Second)
	review := ReviewSubmission{
		ID: uuid.NewString(), ChangeSetID: input.ChangeSetID, ReviewedState: input.ReviewedState,
		Binding: input.Binding, Verdict: input.Verdict, Author: author, SubmittedAt: submittedAt,
		Body: input.Body, Comments: comments, ReviewedChange: reviewed,
	}
	object, err := manager.writeReview(ctx, storePath, review)
	if err != nil {
		return ReviewSubmission{}, err
	}
	if err := manager.git.createReview(ctx, storePath, changeSetRef("active", input.ChangeSetID), input.ReviewedState, reviewRef(input.ChangeSetID, review.ID), object); err != nil {
		return ReviewSubmission{}, ErrReviewInvalidated
	}
	review.RefObject = object
	return review, nil
}

func (manager *Manager) writeReview(ctx context.Context, storePath string, review ReviewSubmission) (string, error) {
	commentIDs := make([]string, len(review.Comments))
	var commentTree strings.Builder
	for index, comment := range review.Comments {
		commentIDs[index] = comment.ID
		metadata, err := yaml.Marshal(reviewCommentMetadata{Format: "workbraid-review-comment", Version: 1, ID: comment.ID, Anchor: comment.Anchor})
		if err != nil {
			return "", err
		}
		metadataBlob, err := manager.git.writeBlob(ctx, storePath, metadata)
		if err != nil {
			return "", err
		}
		bodyBlob, err := manager.git.writeBlob(ctx, storePath, []byte(comment.Body))
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&commentTree, "100644 blob %s\t%s.md\n100644 blob %s\t%s.yaml\n", bodyBlob, comment.ID, metadataBlob, comment.ID)
	}
	metadata, err := yaml.Marshal(reviewMetadata{
		Format: "workbraid-review", Version: 1, ID: review.ID, ChangeSetID: review.ChangeSetID,
		ReviewedState: review.ReviewedState, Binding: review.Binding, Verdict: review.Verdict,
		Author: review.Author, SubmittedAt: review.SubmittedAt.UTC().Format(time.RFC3339), Comments: commentIDs,
	})
	if err != nil {
		return "", err
	}
	metadataBlob, err := manager.git.writeBlob(ctx, storePath, metadata)
	if err != nil {
		return "", err
	}
	bodyBlob, err := manager.git.writeBlob(ctx, storePath, []byte(review.Body))
	if err != nil {
		return "", err
	}
	var root strings.Builder
	fmt.Fprintf(&root, "100644 blob %s\tbody.md\n", bodyBlob)
	if len(review.Comments) > 0 {
		commentsTree, err := manager.git.makeTree(ctx, storePath, []byte(commentTree.String()))
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&root, "040000 tree %s\tcomments\n", commentsTree)
	}
	fmt.Fprintf(&root, "100644 blob %s\treview.yaml\n", metadataBlob)
	tree, err := manager.git.makeTree(ctx, storePath, []byte(root.String()))
	if err != nil {
		return "", err
	}
	return manager.git.makeReviewCommit(ctx, storePath, tree, review.ReviewedState)
}

func (manager *Manager) LoadReviews(ctx context.Context, storeID, onlyChangeSetID string) ([]ReviewSubmission, []UnavailableReview, error) {
	storePath, err := manager.StorePath(storeID)
	if err != nil {
		return nil, nil, err
	}
	prefix := reviewsRefPrefix
	if onlyChangeSetID != "" {
		prefix += onlyChangeSetID + "/"
	}
	refs, err := manager.git.refsUnder(ctx, storePath, prefix)
	if err != nil {
		return nil, nil, err
	}
	counts := make(map[string]int)
	for _, ref := range refs {
		parts := strings.Split(strings.TrimPrefix(ref.Name, reviewsRefPrefix), "/")
		if len(parts) == 2 {
			counts[parts[1]]++
		}
	}
	var valid []ReviewSubmission
	var unavailable []UnavailableReview
	for _, ref := range refs {
		parts := strings.Split(strings.TrimPrefix(ref.Name, reviewsRefPrefix), "/")
		if len(parts) != 2 || !canonicalUUID(parts[0]) || !canonicalUUID(parts[1]) || counts[parts[len(parts)-1]] != 1 {
			unavailable = append(unavailable, UnavailableReview{Ref: ref.Name, Reason: "invalid or conflicting review ref identity"})
			continue
		}
		review, err := manager.loadReview(ctx, storePath, storeID, parts[0], parts[1], ref.Object)
		if err != nil {
			unavailable = append(unavailable, UnavailableReview{ChangeSetID: parts[0], ReviewID: parts[1], Ref: ref.Name, Reason: err.Error()})
			continue
		}
		valid = append(valid, review)
	}
	sort.Slice(valid, func(i, j int) bool {
		if !valid[i].SubmittedAt.Equal(valid[j].SubmittedAt) {
			return valid[i].SubmittedAt.After(valid[j].SubmittedAt)
		}
		return valid[i].ID < valid[j].ID
	})
	sort.Slice(unavailable, func(i, j int) bool { return unavailable[i].Ref < unavailable[j].Ref })
	return valid, unavailable, nil
}

func (manager *Manager) loadReview(ctx context.Context, storePath, storeID, changeSetID, reviewID, object string) (ReviewSubmission, error) {
	if kind, err := manager.git.objectType(ctx, storePath, object); err != nil || kind != "commit" {
		return ReviewSubmission{}, errors.New("review ref does not name a commit")
	}
	parent, err := manager.git.commitParent(ctx, storePath, object)
	if err != nil {
		return ReviewSubmission{}, err
	}
	entries, err := manager.git.directTreeEntries(ctx, storePath, object)
	if err != nil {
		return ReviewSubmission{}, err
	}
	byPath := map[string]treeEntry{}
	for _, entry := range entries {
		if _, exists := byPath[entry.Path]; exists {
			return ReviewSubmission{}, errors.New("review tree contains duplicate path")
		}
		byPath[entry.Path] = entry
	}
	for path, entry := range byPath {
		if path == "review.yaml" || path == "body.md" {
			if entry.Type != "blob" || entry.Mode != "100644" {
				return ReviewSubmission{}, errors.New("review document has invalid mode")
			}
		} else if path == "comments" {
			if entry.Type != "tree" || entry.Mode != "040000" {
				return ReviewSubmission{}, errors.New("review comments entry is invalid")
			}
		} else {
			return ReviewSubmission{}, fmt.Errorf("review tree contains unknown path %q", path)
		}
	}
	if byPath["review.yaml"].Object == "" || byPath["body.md"].Object == "" {
		return ReviewSubmission{}, errors.New("review tree is incomplete")
	}
	metadataBytes, err := manager.git.readBlob(ctx, storePath, byPath["review.yaml"].Object)
	if err != nil {
		return ReviewSubmission{}, err
	}
	metadata, err := parseReviewMetadata(metadataBytes)
	if err != nil {
		return ReviewSubmission{}, err
	}
	if metadata.ID != reviewID || metadata.ChangeSetID != changeSetID || metadata.ReviewedState != parent {
		return ReviewSubmission{}, errors.New("review identity or parent does not match its ref")
	}
	body, err := manager.git.readBlob(ctx, storePath, byPath["body.md"].Object)
	if err != nil || !utf8.Valid(body) {
		return ReviewSubmission{}, errors.New("review body is not valid UTF-8")
	}
	reviewed, err := manager.loadChangeSet(ctx, storePath, storeID, "active", changeSetID, "", parent)
	if err != nil || reviewed.Review == nil || reviewed.Candidate == nil {
		return ReviewSubmission{}, errors.New("reviewed Change Set state is unavailable")
	}
	if reviewed.Review.BaseRevision != metadata.Binding.BaseRevision || reviewed.Review.CandidateTree != metadata.Binding.CandidateTree || reviewed.Review.Generation != metadata.Binding.Generation {
		return ReviewSubmission{}, errors.New("review binding does not match its reviewed state")
	}
	comments, err := manager.loadReviewComments(ctx, storePath, byPath["comments"], metadata.Comments, reviewed)
	if err != nil {
		return ReviewSubmission{}, err
	}
	if metadata.Verdict == "comment" && strings.TrimSpace(string(body)) == "" && len(comments) == 0 {
		return ReviewSubmission{}, errors.New("comment review is empty")
	}
	return ReviewSubmission{ID: metadata.ID, ChangeSetID: metadata.ChangeSetID, ReviewedState: parent, Binding: metadata.Binding,
		Verdict: metadata.Verdict, Author: metadata.Author, SubmittedAt: metadata.submittedTime(), Body: string(body), Comments: comments,
		RefObject: object, ReviewedChange: reviewed}, nil
}

func (manager *Manager) loadReviewComments(ctx context.Context, storePath string, commentsEntry treeEntry, ids []string, reviewed ChangeSet) ([]ReviewComment, error) {
	if len(ids) == 0 {
		if commentsEntry.Object != "" {
			return nil, errors.New("empty review contains a comments tree")
		}
		return nil, nil
	}
	if commentsEntry.Object == "" || commentsEntry.Type != "tree" {
		return nil, errors.New("review comments tree is missing")
	}
	entries, err := manager.git.directTreeEntries(ctx, storePath, commentsEntry.Object)
	if err != nil {
		return nil, err
	}
	byPath := make(map[string]treeEntry, len(entries))
	for _, entry := range entries {
		if _, exists := byPath[entry.Path]; exists || entry.Type != "blob" || entry.Mode != "100644" {
			return nil, errors.New("review comments tree is malformed")
		}
		byPath[entry.Path] = entry
	}
	if len(byPath) != len(ids)*2 {
		return nil, errors.New("review comments tree has unmatched paths")
	}
	seen := map[string]bool{}
	comments := make([]ReviewComment, len(ids))
	for index, id := range ids {
		if !canonicalUUID(id) || seen[id] {
			return nil, errors.New("review comments contain duplicate or invalid identity")
		}
		seen[id] = true
		metadataEntry, bodyEntry := byPath[id+".yaml"], byPath[id+".md"]
		if metadataEntry.Object == "" || bodyEntry.Object == "" {
			return nil, errors.New("review comment pair is missing")
		}
		metadataBytes, err := manager.git.readBlob(ctx, storePath, metadataEntry.Object)
		if err != nil {
			return nil, err
		}
		metadata, err := parseReviewCommentMetadata(metadataBytes)
		if err != nil || metadata.ID != id {
			return nil, errors.New("review comment metadata is invalid")
		}
		body, err := manager.git.readBlob(ctx, storePath, bodyEntry.Object)
		if err != nil || !utf8.Valid(body) || strings.TrimSpace(string(body)) == "" {
			return nil, errors.New("review comment body is invalid")
		}
		if err := validateReviewAnchor(reviewed, metadata.Anchor); err != nil {
			return nil, fmt.Errorf("review comment anchor is invalid: %w", err)
		}
		comments[index] = ReviewComment{ID: id, Body: string(body), Anchor: metadata.Anchor}
	}
	return comments, nil
}

func parseReviewMetadata(contents []byte) (reviewMetadata, error) {
	root, err := oneYAMLMapping(contents, "review.yaml")
	if err != nil {
		return reviewMetadata{}, err
	}
	fields, err := validateClosedMapping(root, "review.yaml", map[string]string{
		"format": "!!str", "version": "!!int", "id": "!!str", "change_set_id": "!!str", "reviewed_state": "!!str",
		"binding": "!!map", "verdict": "!!str", "author": "!!str", "submitted_at": "!!str", "comments": "!!seq",
	}, nil)
	if err != nil {
		return reviewMetadata{}, err
	}
	if _, err := validateClosedMapping(fields["binding"], "review.yaml binding", map[string]string{"base_revision": "!!str", "candidate_tree": "!!str", "generation": "!!int"}, nil); err != nil {
		return reviewMetadata{}, err
	}
	for _, id := range fields["comments"].Content {
		if id.ShortTag() != "!!str" {
			return reviewMetadata{}, errors.New("review.yaml comment IDs must be strings")
		}
	}
	var metadata reviewMetadata
	if err := root.Decode(&metadata); err != nil {
		return metadata, err
	}
	if metadata.Format != "workbraid-review" || metadata.Version != 1 || !canonicalUUID(metadata.ID) || !canonicalUUID(metadata.ChangeSetID) || !validObjectID(metadata.ReviewedState) || !validReviewBinding(metadata.Binding) || strings.TrimSpace(metadata.Author) != metadata.Author || metadata.Author == "" || strings.ContainsAny(metadata.Author, "\r\n") {
		return metadata, errors.New("review.yaml values are invalid")
	}
	if metadata.Verdict != "comment" && metadata.Verdict != "approve" && metadata.Verdict != "request_changes" {
		return metadata, errors.New("review.yaml verdict is unsupported")
	}
	submittedAt, err := time.Parse(time.RFC3339, metadata.SubmittedAt)
	_, offset := submittedAt.Zone()
	if err != nil || offset != 0 {
		return metadata, errors.New("review.yaml submitted_at is invalid")
	}
	return metadata, nil
}

func (metadata reviewMetadata) submittedTime() time.Time {
	value, _ := time.Parse(time.RFC3339, metadata.SubmittedAt)
	return value
}

func parseReviewCommentMetadata(contents []byte) (reviewCommentMetadata, error) {
	root, err := oneYAMLMapping(contents, "review comment")
	if err != nil {
		return reviewCommentMetadata{}, err
	}
	fields, err := validateClosedMapping(root, "review comment", map[string]string{"format": "!!str", "version": "!!int", "id": "!!str", "anchor": "!!map"}, nil)
	if err != nil {
		return reviewCommentMetadata{}, err
	}
	if err := validateReviewAnchorYAML(fields["anchor"]); err != nil {
		return reviewCommentMetadata{}, err
	}
	var metadata reviewCommentMetadata
	if err := root.Decode(&metadata); err != nil {
		return metadata, err
	}
	if metadata.Format != "workbraid-review-comment" || metadata.Version != 1 || !canonicalUUID(metadata.ID) {
		return metadata, errors.New("review comment values are invalid")
	}
	return metadata, nil
}

func validateReviewAnchorYAML(root *yaml.Node) error {
	if root == nil || root.Kind != yaml.MappingNode {
		return errors.New("review anchor must be a mapping")
	}
	kind := ""
	for index := 0; index < len(root.Content); index += 2 {
		if root.Content[index].Value == "kind" {
			kind = root.Content[index+1].Value
		}
	}
	required := map[string]string{"kind": "!!str"}
	optional := map[string]string{}
	add := func(key, kind string, needed bool) {
		if needed {
			required[key] = kind
		} else {
			optional[key] = kind
		}
	}
	switch kind {
	case "proposal":
	case "proposal_markdown":
		add("start_line", "!!int", true)
		add("end_line", "!!int", true)
	case "component":
		add("side", "!!str", true)
		add("component_id", "!!str", true)
	case "component_markdown":
		add("side", "!!str", true)
		add("component_id", "!!str", true)
		add("start_line", "!!int", true)
		add("end_line", "!!int", true)
	case "diagram":
		add("side", "!!str", true)
		add("diagram_id", "!!str", true)
	case "composition":
		add("side", "!!str", true)
		add("diagram_id", "!!str", true)
		add("component_id", "!!str", true)
		add("aspect", "!!str", true)
		add("detail_diagram_id", "!!str", false)
	case "relationship":
		add("side", "!!str", true)
		add("source_component_id", "!!str", true)
		add("target_component_id", "!!str", true)
		add("label", "!!str", true)
		add("occurrence", "!!int", true)
	default:
		return errors.New("review anchor kind is unsupported")
	}
	_, err := validateClosedMapping(root, "review anchor", required, optional)
	return err
}

func validReviewBinding(binding ReviewBinding) bool {
	return validObjectID(binding.BaseRevision) && validObjectID(binding.CandidateTree)
}

func validateReviewAnchor(reviewed ChangeSet, anchor ReviewAnchor) error {
	if anchor.Kind == "proposal" {
		if anchor != (ReviewAnchor{Kind: "proposal"}) {
			return errors.New("proposal anchor has extra fields")
		}
		return nil
	}
	if anchor.Kind == "proposal_markdown" {
		if anchor.Side != "" || anchor.ComponentID != "" || anchor.DiagramID != "" || anchor.Aspect != "" || anchor.DetailDiagramID != "" || anchor.SourceComponentID != "" || anchor.TargetComponentID != "" || anchor.Label != "" || anchor.Occurrence != 0 || !validLineRange([]byte(reviewed.Proposal), anchor.StartLine, anchor.EndLine) {
			return errors.New("proposal Markdown range does not exist")
		}
		return nil
	}
	var snapshot Snapshot
	if anchor.Side == "before" {
		snapshot = reviewed.BaseSnapshot
	} else if anchor.Side == "with_changes" && reviewed.Candidate != nil {
		snapshot = reviewed.Candidate.Snapshot()
	} else {
		return errors.New("Architecture side is invalid")
	}
	switch anchor.Kind {
	case "component":
		if !canonicalUUID(anchor.ComponentID) || !snapshot.HasComponent(anchor.ComponentID) || anchor.StartLine != 0 || anchor.EndLine != 0 || anchor.DiagramID != "" || anchor.Aspect != "" || anchor.DetailDiagramID != "" || anchor.SourceComponentID != "" || anchor.TargetComponentID != "" || anchor.Label != "" || anchor.Occurrence != 0 {
			return errors.New("Component does not exist on that side")
		}
	case "component_markdown":
		source, ok := snapshot.ComponentMarkdownSource(anchor.ComponentID)
		if !ok || !validLineRange(source, anchor.StartLine, anchor.EndLine) || anchor.DiagramID != "" || anchor.Aspect != "" || anchor.DetailDiagramID != "" || anchor.SourceComponentID != "" || anchor.TargetComponentID != "" || anchor.Label != "" || anchor.Occurrence != 0 {
			return errors.New("Component Markdown range does not exist")
		}
	case "diagram":
		if !canonicalUUID(anchor.DiagramID) || !snapshot.HasDiagram(anchor.DiagramID) || anchor.ComponentID != "" || anchor.StartLine != 0 || anchor.EndLine != 0 || anchor.Aspect != "" || anchor.DetailDiagramID != "" || anchor.SourceComponentID != "" || anchor.TargetComponentID != "" || anchor.Label != "" || anchor.Occurrence != 0 {
			return errors.New("Diagram does not exist on that side")
		}
	case "composition":
		role, exists := snapshot.ComponentAppearanceRole(anchor.DiagramID, anchor.ComponentID)
		if !exists || (anchor.Aspect != "home" && anchor.Aspect != "reference" && anchor.Aspect != "detail") || anchor.StartLine != 0 || anchor.EndLine != 0 || anchor.SourceComponentID != "" || anchor.TargetComponentID != "" || anchor.Label != "" || anchor.Occurrence != 0 {
			return errors.New("composition fact does not exist")
		}
		if anchor.Aspect == "home" || anchor.Aspect == "reference" {
			if role != anchor.Aspect || anchor.DetailDiagramID != "" {
				return errors.New("composition fact does not exist")
			}
		} else if !snapshot.HasDetailLink(anchor.DiagramID, anchor.ComponentID, anchor.DetailDiagramID) {
			return errors.New("detail link does not exist")
		}
	case "relationship":
		if !canonicalUUID(anchor.SourceComponentID) || !canonicalUUID(anchor.TargetComponentID) || anchor.Occurrence < 1 || anchor.ComponentID != "" || anchor.DiagramID != "" || anchor.Aspect != "" || anchor.DetailDiagramID != "" || anchor.StartLine != 0 || anchor.EndLine != 0 {
			return errors.New("Relationship selector is invalid")
		}
		count := 0
		for _, component := range snapshot.AuthoringComponents() {
			if component.ID == anchor.SourceComponentID {
				for _, relationship := range component.Relationships {
					if relationship.TargetID == anchor.TargetComponentID && relationship.Label == anchor.Label {
						count++
					}
				}
			}
		}
		if count < anchor.Occurrence {
			return errors.New("Relationship occurrence does not exist")
		}
	default:
		return errors.New("review anchor kind is unsupported")
	}
	return nil
}

func validLineRange(source []byte, start, end int) bool {
	if start < 1 || end < start {
		return false
	}
	lines := 0
	if len(source) > 0 {
		lines = 1
		for index, value := range source {
			if value == '\n' && index != len(source)-1 {
				lines++
			}
		}
	}
	return end <= lines
}
