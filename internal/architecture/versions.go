package architecture

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrVersionMoved = errors.New("comparison version moved")
var ErrVersionUnavailable = errors.New("comparison version unavailable")

// VersionSelector addresses only an existing retention authority, never an
// arbitrary Git object. State is the active/receipt or reviewed-parent commit.
type VersionSelector struct {
	Kind        string `json:"kind"`
	Revision    string `json:"revision,omitempty"`
	ChangeSetID string `json:"change_set_id,omitempty"`
	State       string `json:"state,omitempty"`
	ReviewID    string `json:"review_id,omitempty"`
	Side        string `json:"side,omitempty"`
}

func exactUUID(s string) bool { id, err := uuid.Parse(s); return err == nil && id.String() == s }
func exactCommit(s string) bool {
	if len(s) != 40 {
		return false
	}
	for _, c := range s {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func (s VersionSelector) Validate() error {
	if s.Kind == "accepted" {
		if exactCommit(s.Revision) && s.ChangeSetID == "" && s.State == "" && s.ReviewID == "" && s.Side == "" {
			return nil
		}
	} else if (s.Kind == "proposal" || s.Kind == "applied" || s.Kind == "submitted_review") && s.Revision == "" && exactUUID(s.ChangeSetID) && exactCommit(s.State) && (s.Side == "base" || s.Side == "candidate") {
		if s.Kind == "submitted_review" && exactUUID(s.ReviewID) || s.Kind != "submitted_review" && s.ReviewID == "" {
			return nil
		}
	}
	return ErrInvalid
}

// Reject explicitly supplied foreign fields, nulls, duplicates and empty
// optional fields as well as unknown keys. Used by browser, CLI and MCP alike.
func (s *VersionSelector) UnmarshalJSON(data []byte) error {
	type plain VersionSelector
	d := json.NewDecoder(bytes.NewReader(data))
	t, err := d.Token()
	if err != nil || t != json.Delim('{') {
		return ErrInvalid
	}
	fields := map[string]json.RawMessage{}
	for d.More() {
		key, err := d.Token()
		if err != nil {
			return ErrInvalid
		}
		k := key.(string)
		if _, exists := fields[k]; exists {
			return ErrInvalid
		}
		var v json.RawMessage
		if d.Decode(&v) != nil || bytes.Equal(v, []byte("null")) || bytes.Equal(v, []byte(`""`)) {
			return ErrInvalid
		}
		fields[k] = v
	}
	if _, err := d.Token(); err != nil {
		return ErrInvalid
	}
	var p plain
	d = json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if d.Decode(&p) != nil {
		return ErrInvalid
	}
	*s = VersionSelector(p)
	return s.Validate()
}

type VersionInfo struct {
	Selector       VersionSelector `json:"selector"`
	Label          string          `json:"label"`
	Context        string          `json:"context,omitempty"`
	Revision       string          `json:"revision"`
	Generation     *uint64         `json:"generation,omitempty"`
	Document       *string         `json:"document,omitempty"`
	DocumentSource string          `json:"document_source,omitempty"`
	Unavailable    bool            `json:"unavailable,omitempty"`
}
type RetainedVersion struct {
	Info     VersionInfo
	Snapshot Snapshot
	refs     []refEntry
}
type VersionPageRequest struct {
	StoreID     string `json:"store_id"`
	Source      string `json:"source"`
	ChangeSetID string `json:"change_set_id,omitempty"`
	Cursor      string `json:"cursor,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}
type VersionPage struct {
	Versions         []VersionInfo          `json:"versions"`
	ReviewProposals  []ReviewProposalOption `json:"review_proposals,omitempty"`
	NextCursor       string                 `json:"next_cursor,omitempty"`
	AcceptedRevision string                 `json:"accepted_revision"`
}
type ReviewProposalOption struct {
	ChangeSetID string `json:"change_set_id"`
	Label       string `json:"label"`
}

// This continuation is specific to the four version lists; it is not an
// authority token. Every value is revalidated against its source on each read.
type versionCursor struct {
	Store    string `json:"store"`
	Source   string `json:"source"`
	Proposal string `json:"proposal,omitempty"`
	Tip      string `json:"tip"`
	Offset   int    `json:"offset,omitempty"`
	After    string `json:"after,omitempty"`
}

func (c *versionCursor) UnmarshalJSON(data []byte) error {
	d := json.NewDecoder(bytes.NewReader(data))
	t, err := d.Token()
	if err != nil || t != json.Delim('{') {
		return ErrInvalid
	}
	seen := map[string]bool{}
	for d.More() {
		k, e := d.Token()
		if e != nil {
			return ErrInvalid
		}
		key := k.(string)
		if seen[key] {
			return ErrInvalid
		}
		seen[key] = true
		var raw json.RawMessage
		if d.Decode(&raw) != nil || bytes.Equal(raw, []byte("null")) {
			return ErrInvalid
		}
	}
	if _, err = d.Token(); err != nil {
		return ErrInvalid
	}
	if d.Decode(new(any)) != io.EOF {
		return ErrInvalid
	}
	type plain versionCursor
	d = json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	var p plain
	if d.Decode(&p) != nil || !seen["store"] || !seen["source"] || !seen["tip"] {
		return ErrInvalid
	}
	*c = versionCursor(p)
	return nil
}

func (m *Manager) verifyVersionRefs(ctx context.Context, path string, refs []refEntry) error {
	for _, ref := range refs {
		value, _, err := m.git.resolveRef(ctx, path, ref.Name)
		if err != nil {
			return ErrVersionUnavailable
		}
		if value != ref.Object {
			return ErrVersionMoved
		}
	}
	return nil
}

func (m *Manager) retainedVersion(ctx context.Context, accepted Snapshot, s VersionSelector) (RetainedVersion, error) {
	v := RetainedVersion{Info: VersionInfo{Selector: s}}
	if s.Validate() != nil {
		return v, ErrInvalid
	}
	path, _ := m.StorePath(accepted.StoreID())
	v.refs = []refEntry{{Name: acceptedRef, Object: accepted.Revision()}}
	if s.Kind == "accepted" {
		// Establish reachability before the general historical loader is called.
		if _, err := runGit(ctx, nil, "--git-dir", path, "merge-base", "--is-ancestor", s.Revision, accepted.Revision()); err != nil {
			return v, ErrVersionUnavailable
		}
		snap, err := m.LoadRevision(ctx, accepted, s.Revision)
		if err != nil {
			return v, ErrVersionUnavailable
		}
		v.Snapshot = snap
		v.Info.Revision = snap.Revision()
		date, err := runGit(ctx, nil, "--git-dir", path, "show", "-s", "--format=%cI", s.Revision, "--")
		if err != nil {
			return v, ErrVersionUnavailable
		}
		stamp, err := time.Parse(time.RFC3339, strings.TrimSpace(string(date)))
		if err != nil {
			return v, ErrVersionUnavailable
		}
		v.Info.Label = "Accepted ancestry · " + stamp.Format("2 Jan 2006, 15:04:05")
		if s.Revision == accepted.Revision() {
			v.Info.Label = "Current Accepted"
		}
		return v, nil
	}
	var change ChangeSet
	if s.Kind == "submitted_review" {
		ref := reviewRef(s.ChangeSetID, s.ReviewID)
		object, present, err := m.git.resolveRef(ctx, path, ref)
		if err != nil || !present {
			return v, ErrVersionUnavailable
		}
		v.refs = append(v.refs, refEntry{Name: ref, Object: object})
		review, err := m.loadReview(ctx, path, accepted.StoreID(), s.ChangeSetID, s.ReviewID, object)
		if err != nil {
			return v, ErrVersionUnavailable
		}
		if review.ReviewedState != s.State {
			return v, ErrVersionMoved
		}
		change = review.ReviewedChange
		v.Info.Context = fmt.Sprintf("Submitted review by %s · %s · generation %d", review.Author, review.SubmittedAt.Format("2 Jan 2006 15:04"), change.Generation)
	} else {
		prefix, other, lifecycle := activeChangeSetPrefix, appliedChangeSetPrefix, "active"
		if s.Kind == "applied" {
			prefix, other, lifecycle = other, prefix, "applied"
		}
		ref := prefix + s.ChangeSetID
		object, present, err := m.git.resolveRef(ctx, path, ref)
		if err != nil {
			return v, ErrVersionUnavailable
		}
		if !present || object != s.State {
			return v, ErrVersionMoved
		}
		otherObject, _, err := m.git.resolveRef(ctx, path, other+s.ChangeSetID)
		if err != nil || otherObject != "" {
			return v, ErrVersionUnavailable
		}
		v.refs = append(v.refs, refEntry{Name: ref, Object: object}, refEntry{Name: other + s.ChangeSetID, Object: ""})
		change, err = m.loadChangeSet(ctx, path, accepted.StoreID(), lifecycle, s.ChangeSetID, ref, object)
		if err != nil {
			return v, ErrVersionUnavailable
		}
		v.Info.Context = fmt.Sprintf("%s proposal · generation %d", lifecycle, change.Generation)
	}
	v.Info.Generation = &change.Generation
	v.Info.Document = &change.Proposal
	v.Info.DocumentSource = s.ChangeSetID + ":" + s.State
	v.Snapshot = change.BaseSnapshot
	if s.Side == "candidate" {
		if change.Candidate == nil {
			return v, ErrVersionUnavailable
		}
		v.Snapshot = change.Candidate.Snapshot()
	}
	v.Info.Label = change.Name + " · Base version"
	if s.Side == "candidate" {
		v.Info.Label = change.Name + " · Proposed version"
	}
	v.Info.Revision = v.Snapshot.Revision()
	return v, nil
}

func (m *Manager) CompareVersions(ctx context.Context, store string, before, after VersionSelector) (Snapshot, RetainedVersion, RetainedVersion, string, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	var a, b RetainedVersion
	if !exactUUID(store) || before.Validate() != nil || after.Validate() != nil {
		return Snapshot{}, a, b, "", ErrInvalid
	}
	accepted, err := m.LoadAccepted(ctx, store)
	if err != nil {
		return accepted, a, b, "", ErrVersionUnavailable
	}
	a, err = m.retainedVersion(ctx, accepted, before)
	if err != nil {
		return accepted, a, b, "", err
	}
	b, err = m.retainedVersion(ctx, accepted, after)
	if err != nil {
		return accepted, a, b, "", err
	}
	path, _ := m.StorePath(store)
	diff, err := m.git.diffTrees(ctx, path, a.Snapshot.Revision(), b.Snapshot.Revision())
	if err != nil {
		return accepted, a, b, "", ErrVersionUnavailable
	}
	if err = m.verifyVersionRefs(ctx, path, append(a.refs, b.refs...)); err != nil {
		return accepted, a, b, "", err
	}
	return accepted, a, b, presentUnifiedDiff(diff), nil
}

// Stream lexically ordered ref names, retaining only one page. Cancel Git
// after the lookahead row. A request deadline bounds scanning past the cursor;
// no whole-catalog slice or reconstruction precedes pagination.
func versionRefPage(ctx context.Context, path, prefix, after string, limit int) ([]refEntry, bool, error) {
	child, cancel := context.WithCancel(ctx)
	defer cancel()
	cmd := gitCommand(child, "--git-dir", path, "for-each-ref", "--sort=refname", "--format=%(refname)%00%(objectname)", prefix)
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, false, err
	}
	if err = cmd.Start(); err != nil {
		return nil, false, err
	}
	scan := bufio.NewScanner(pipe)
	scan.Buffer(make([]byte, 4096), 65536)
	rows := []refEntry{}
	more := false
	for scan.Scan() {
		name, object, ok := strings.Cut(scan.Text(), "\x00")
		if !ok {
			cancel()
			_ = cmd.Wait()
			return nil, false, ErrVersionUnavailable
		}
		if name <= after {
			continue
		}
		if len(rows) == limit {
			more = true
			cancel()
			break
		}
		rows = append(rows, refEntry{Name: name, Object: object})
	}
	scanErr := scan.Err()
	waitErr := cmd.Wait()
	if ctx.Err() != nil || scanErr != nil || !more && waitErr != nil {
		return nil, false, ErrVersionUnavailable
	}
	return rows, more, nil
}

func (m *Manager) VersionCatalog(ctx context.Context, req VersionPageRequest) (VersionPage, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	page := VersionPage{Versions: []VersionInfo{}}
	if !exactUUID(req.StoreID) || req.Limit < 0 || req.Limit > 50 {
		return page, ErrInvalid
	}
	if req.Limit == 0 {
		req.Limit = 25
	}
	prefix := ""
	switch req.Source {
	case "accepted":
	case "proposal":
		prefix = activeChangeSetPrefix
	case "applied":
		prefix = appliedChangeSetPrefix
	case "submitted_review":
		if !exactUUID(req.ChangeSetID) {
			return page, ErrInvalid
		}
		prefix = reviewsRefPrefix + req.ChangeSetID + "/"
	case "review_proposals":
		prefix = reviewsRefPrefix
	default:
		return page, ErrInvalid
	}
	if req.Source != "submitted_review" && req.ChangeSetID != "" {
		return page, ErrInvalid
	}
	accepted, err := m.LoadAccepted(ctx, req.StoreID)
	if err != nil {
		return page, ErrVersionUnavailable
	}
	page.AcceptedRevision = accepted.Revision()
	c := versionCursor{Store: req.StoreID, Source: req.Source, Proposal: req.ChangeSetID, Tip: accepted.Revision()}
	if req.Cursor != "" {
		if len(req.Cursor) > 2048 {
			return page, ErrInvalid
		}
		data, e := base64.RawURLEncoding.DecodeString(req.Cursor)
		if e != nil {
			return page, ErrInvalid
		}
		d := json.NewDecoder(bytes.NewReader(data))
		d.DisallowUnknownFields()
		if d.Decode(&c) != nil || d.Decode(new(any)) != io.EOF || c.Store != req.StoreID || c.Source != req.Source || c.Proposal != req.ChangeSetID || c.Offset < 0 || c.Offset > 100000000 || !exactCommit(c.Tip) {
			return page, ErrInvalid
		}
		if c.Tip != accepted.Revision() {
			return page, ErrVersionMoved
		}
		if req.Source == "accepted" && c.After != "" || req.Source != "accepted" && (c.Offset != 0 || !strings.HasPrefix(c.After, prefix) || strings.ContainsAny(c.After, "\x00\n\r")) {
			return page, ErrInvalid
		}
	}
	path, _ := m.StorePath(req.StoreID)
	refs := []refEntry{{Name: acceptedRef, Object: accepted.Revision()}}
	more := false
	if req.Source == "accepted" {
		out, e := runGit(ctx, nil, "--git-dir", path, "rev-list", "--date-order", "--max-count="+strconv.Itoa(req.Limit+1), "--skip="+strconv.Itoa(c.Offset), accepted.Revision(), "--")
		if e != nil {
			return page, ErrVersionUnavailable
		}
		commits := strings.Fields(string(out))
		if len(commits) > req.Limit {
			more = true
			commits = commits[:req.Limit]
		}
		for _, commit := range commits {
			s := VersionSelector{Kind: "accepted", Revision: commit}
			v, e := m.retainedVersion(ctx, accepted, s)
			if e != nil {
				page.Versions = append(page.Versions, VersionInfo{Selector: s, Label: "Unavailable historical version · " + commit[:8], Unavailable: true})
				continue
			}
			page.Versions = append(page.Versions, v.Info)
		}
		c.Offset += len(commits)
	} else {
		rows, hasMore, e := versionRefPage(ctx, path, prefix, c.After, req.Limit)
		if e != nil {
			return page, ErrVersionUnavailable
		}
		more = hasMore
		for _, row := range rows {
			c.After = row.Name
			refs = append(refs, row)
			if req.Source == "review_proposals" {
				parts := strings.Split(strings.TrimPrefix(row.Name, reviewsRefPrefix), "/")
				if len(parts) != 2 || !exactUUID(parts[0]) || !exactUUID(parts[1]) {
					continue
				}
				found := false
				for _, p := range page.ReviewProposals {
					if p.ChangeSetID == parts[0] {
						found = true
					}
				}
				if found {
					continue
				}
				name := "Unavailable review · " + parts[0][:8]
				if review, e := m.loadReview(ctx, path, req.StoreID, parts[0], parts[1], row.Object); e == nil {
					name = review.ReviewedChange.Name
				}
				page.ReviewProposals = append(page.ReviewProposals, ReviewProposalOption{ChangeSetID: parts[0], Label: name})
				continue
			}
			s := VersionSelector{Kind: req.Source, State: row.Object, Side: "base"}
			if req.Source == "submitted_review" {
				parts := strings.Split(strings.TrimPrefix(row.Name, reviewsRefPrefix), "/")
				if len(parts) == 2 {
					s.ChangeSetID, s.ReviewID = parts[0], parts[1]
				}
				if exactUUID(s.ChangeSetID) && exactUUID(s.ReviewID) {
					s.State, _ = m.git.commitParent(ctx, path, row.Object)
				}
			} else {
				s.ChangeSetID = strings.TrimPrefix(row.Name, prefix)
			}
			v, e := m.retainedVersion(ctx, accepted, s)
			if e != nil {
				page.Versions = append(page.Versions, VersionInfo{Selector: s, Label: "Unavailable " + strings.ReplaceAll(req.Source, "_", " ") + " version", Unavailable: true})
				continue
			}
			refs = append(refs, v.refs...)
			// Catalog carries provenance, not complete proposal documents.
			v.Info.Document = nil
			page.Versions = append(page.Versions, v.Info)
			s.Side = "candidate"
			v, e = m.retainedVersion(ctx, accepted, s)
			if e != nil {
				page.Versions = append(page.Versions, VersionInfo{Selector: s, Label: strings.TrimSuffix(page.Versions[len(page.Versions)-1].Label, "Base version") + "Proposed version unavailable", Unavailable: true})
				continue
			}
			refs = append(refs, v.refs...)
			v.Info.Document = nil
			page.Versions = append(page.Versions, v.Info)
		}
	}
	if ctx.Err() != nil {
		return page, ErrVersionUnavailable
	}
	if err = m.verifyVersionRefs(ctx, path, refs); err != nil {
		return page, err
	}
	if more {
		data, _ := json.Marshal(c)
		page.NextCursor = base64.RawURLEncoding.EncodeToString(data)
	}
	return page, nil
}
