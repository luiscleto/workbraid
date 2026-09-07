package web

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"workbraid/internal/agentapi"
	"workbraid/internal/architecture"
)

func shapeNoteAction(action string) bool {
	switch action {
	case "shapes", "notes", "set-shape", "restore-default-shape", "add-note", "edit-note", "delete-note":
		return true
	}
	return false
}
func validShapeNoteFields(data []byte, action string, browser bool) bool {
	required := map[string]bool{"store_id": true, "diagram_id": true}
	optional := map[string]bool{}
	if action == "shapes" || action == "notes" {
		optional["change_set_id"] = true
	} else if browser {
		for _, k := range []string{"project_slug", "expected_revision", "change_set_id", "expected_pending_generation", "pending_generation_observed"} {
			optional[k] = true
		}
	} else {
		required["change_set_id"], required["generation"] = true, true
	}
	switch action {
	case "set-shape":
		required["component_id"], required["shape"] = true, true
	case "restore-default-shape":
		required["component_id"] = true
	case "add-note":
		required["text"] = true
	case "edit-note":
		for _, k := range []string{"note_id", "text", "x", "y", "width", "height"} {
			required[k] = true
		}
	case "delete-note":
		required["note_id"] = true
	}
	d := json.NewDecoder(bytes.NewReader(data))
	token, e := d.Token()
	if e != nil || token != json.Delim('{') {
		return false
	}
	seen := map[string]bool{}
	for d.More() {
		token, e = d.Token()
		k, ok := token.(string)
		if e != nil || !ok || seen[k] || !required[k] && !optional[k] {
			return false
		}
		seen[k] = true
		var raw json.RawMessage
		if d.Decode(&raw) != nil {
			return false
		}
		if string(raw) == "null" {
			if browser && k == "expected_pending_generation" {
				continue
			}
			return false
		}
		switch k {
		case "x", "y", "width", "height", "generation", "expected_pending_generation":
			var n int
			if json.Unmarshal(raw, &n) != nil {
				return false
			}
		case "pending_generation_observed":
			var b bool
			if json.Unmarshal(raw, &b) != nil {
				return false
			}
		default:
			var s string
			if json.Unmarshal(raw, &s) != nil {
				return false
			}
		}
	}
	for k := range required {
		if !seen[k] {
			return false
		}
	}
	_, e = d.Token()
	return e == nil && ensureJSONEnd(d) == nil
}

type shapeNoteInput struct {
	DiagramID, ComponentID, NoteID, Text, Shape string
	X, Y, Width, Height                         int
}

func (h *Handler) shapeNoteLocked(ctx context.Context, base architecture.Snapshot, pending *pendingChangeSet, action string, v shapeNoteInput) (*pendingChangeSet, bool, string, string) {
	current := base
	if pending != nil {
		if pending.candidate == nil {
			return pending, false, "", changeValidationBlocked
		}
		object, exists, e := h.architecture.ObserveActiveChangeSet(ctx, pending.storeID, pending.id)
		if e != nil {
			return pending, false, "", changeOperationFailed
		}
		if !exists || object != pending.refObject {
			return pending, false, "", "change_set_state_mismatch"
		}
		current = pending.candidate.Snapshot()
	}
	if _, e := uuid.Parse(v.DiagramID); e != nil {
		return pending, false, "", "invalid_request"
	}
	if !current.HasDiagram(v.DiagramID) {
		return pending, false, "", changeTargetNotFound
	}
	var shape *string
	var note *architecture.NoteValue
	id := v.NoteID
	if strings.Contains(action, "shape") {
		if _, e := uuid.Parse(v.ComponentID); e != nil {
			return pending, false, "", "invalid_request"
		}
		before, exists := current.NodeShape(v.DiagramID, v.ComponentID)
		if !exists {
			return pending, false, "", changeTargetIneligible
		}
		if action == "set-shape" {
			if !architecture.ValidShape(v.Shape) {
				return pending, false, "", "invalid_request"
			}
			shape = &v.Shape
		}
		if before == nil && shape == nil || before != nil && shape != nil && *before == *shape {
			return pending, true, "", ""
		}
	} else {
		notes, _ := current.DiagramNotes(v.DiagramID)
		var before *architecture.NoteValue
		if action != "add-note" {
			parsed, e := uuid.Parse(id)
			if e != nil {
				return pending, false, "", "invalid_request"
			}
			id = parsed.String()
			for _, n := range notes {
				if n.ID == id {
					copy := n.NoteValue
					before = &copy
				}
			}
		}
		switch action {
		case "add-note":
			n, e := current.NewDiagramNote(v.DiagramID, v.Text)
			if e != nil {
				return pending, false, "", "invalid_request"
			}
			id, note = n.ID, &n.NoteValue
		case "edit-note":
			if before == nil {
				return pending, false, "", changeTargetNotFound
			}
			note = &architecture.NoteValue{Text: v.Text, X: v.X, Y: v.Y, Width: v.Width, Height: v.Height}
			if !architecture.ValidNote(*note) {
				return pending, false, "", "invalid_request"
			}
			if *before == *note {
				return pending, true, id, ""
			}
		case "delete-note":
			if before == nil {
				return pending, true, id, ""
			}
		}
	}
	proposed := h.ensurePendingLocked(base, clonePending(pending))
	c := architecture.UpgradePresentation(current, h.durableChangeSet(proposed).Composition)
	if strings.Contains(action, "shape") {
		c = architecture.SetNodeShape(c, v.DiagramID, v.ComponentID, shape)
	} else {
		c = architecture.SetDiagramNote(base, c, v.DiagramID, id, note)
	}
	proposed.nodeShapes, proposed.diagramNotes, proposed.nodeSizes, proposed.architectureVersion = c.NodeShapes, c.DiagramNotes, c.NodeSizes, c.ArchitectureVersion
	h.rebuildPendingLocked(ctx, base, proposed)
	if pendingOperationFailed(proposed) {
		return pending, false, "", changeOperationFailed
	}
	if proposed.candidate == nil {
		return pending, false, "", changeValidationBlocked
	}
	return proposed, false, id, ""
}
func (h *Handler) browserShapeNote(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Origin") != h.expectedOrigin {
		writeJSON(w, http.StatusForbidden, errorResponse{Code: errorOriginMismatch})
		return
	}
	action := path.Base(r.URL.Path)
	data, e := io.ReadAll(http.MaxBytesReader(w, r.Body, maxRequestBody))
	if e != nil || !utf8.Valid(data) || !validShapeNoteFields(data, action, true) {
		writeJSON(w, http.StatusBadRequest, errorResponse{Code: "invalid_request"})
		return
	}
	var v struct {
		diagramMutationRequest
		NoteID string `json:"note_id"`
		Text   string `json:"text"`
		Shape  string `json:"shape"`
	}
	if json.Unmarshal(data, &v) != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Code: "invalid_request"})
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	base, pending, ok := h.writableV2StateLocked(w, v.diagramMutationRequest)
	if !ok {
		return
	}
	in := shapeNoteInput{DiagramID: v.DiagramID, ComponentID: v.ComponentID, NoteID: v.NoteID, Text: v.Text, Shape: v.Shape}
	if v.X != nil {
		in.X = *v.X
	}
	if v.Y != nil {
		in.Y = *v.Y
	}
	if v.Width != nil {
		in.Width = *v.Width
	}
	if v.Height != nil {
		in.Height = *v.Height
	}
	proposed, unchanged, _, code := h.shapeNoteLocked(r.Context(), base, pending, action, in)
	if code != "" {
		writeJSON(w, http.StatusConflict, errorResponse{Code: code})
		return
	}
	if unchanged {
		writeJSON(w, http.StatusOK, h.responseForLoadedProjectLocked(*h.loadedSnapshot, pending, h.loadedStale, h.acceptedDiff))
		return
	}
	h.persistBrowserChangeSetMutationLocked(w, r.Context(), v.ChangeSetID, proposed)
}
func (h *Handler) agentShapeNoteMutation(w http.ResponseWriter, r *http.Request, state agentapi.StatePreconditions, v shapeNoteInput) {
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	base, pending, e := h.checkAgentStateLocked(state)
	if e != nil {
		h.writeAgentDomainErrorLocked(w, http.StatusConflict, e)
		return
	}
	proposed, unchanged, id, code := h.shapeNoteLocked(r.Context(), base, pending, path.Base(r.URL.Path), v)
	if code != "" {
		h.writeAgentErrorLocked(w, http.StatusConflict, code, agentMessage(code), nil)
		return
	}
	if !unchanged && !h.persistAgentMutationLocked(r.Context(), state.ChangeSetID, proposed) {
		h.writeAgentErrorLocked(w, http.StatusConflict, "change_set_state_mismatch", agentMessage("change_set_state_mismatch"), nil)
		return
	}
	h.writeAgentSuccessLocked(w, http.StatusOK, agentMutationResult(proposed, map[string]any{"change_set": h.agentChangeSetProjectionLocked(proposed), "diagram_id": v.DiagramID, "note_id": id, "unchanged": unchanged}))
}
func (h *Handler) agentSetShape(w http.ResponseWriter, r *http.Request) {
	v, ok := decodeAgentRequest[agentapi.DiagramSetShapeRequest](h, w, r)
	if ok {
		h.agentShapeNoteMutation(w, r, v.StatePreconditions, shapeNoteInput{DiagramID: v.DiagramID, ComponentID: v.ComponentID, Shape: v.Shape})
	}
}
func (h *Handler) agentDefaultShape(w http.ResponseWriter, r *http.Request) {
	v, ok := decodeAgentRequest[agentapi.DiagramRestoreDefaultSizeRequest](h, w, r)
	if ok {
		h.agentShapeNoteMutation(w, r, v.StatePreconditions, shapeNoteInput{DiagramID: v.DiagramID, ComponentID: v.ComponentID})
	}
}
func (h *Handler) agentAddNote(w http.ResponseWriter, r *http.Request) {
	v, ok := decodeAgentRequest[agentapi.DiagramAddNoteRequest](h, w, r)
	if ok {
		h.agentShapeNoteMutation(w, r, v.StatePreconditions, shapeNoteInput{DiagramID: v.DiagramID, Text: v.Text})
	}
}
func (h *Handler) agentEditNote(w http.ResponseWriter, r *http.Request) {
	v, ok := decodeAgentRequest[agentapi.DiagramEditNoteRequest](h, w, r)
	if ok {
		h.agentShapeNoteMutation(w, r, v.StatePreconditions, shapeNoteInput{DiagramID: v.DiagramID, NoteID: v.NoteID, Text: v.Text, X: v.X, Y: v.Y, Width: v.Width, Height: v.Height})
	}
}
func (h *Handler) agentDeleteNote(w http.ResponseWriter, r *http.Request) {
	v, ok := decodeAgentRequest[agentapi.DiagramDeleteNoteRequest](h, w, r)
	if ok {
		h.agentShapeNoteMutation(w, r, v.StatePreconditions, shapeNoteInput{DiagramID: v.DiagramID, NoteID: v.NoteID})
	}
}
