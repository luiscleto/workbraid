package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"reflect"
	"testing"

	"workbraid/internal/architecture"
)

func printRead(t *testing.T, handler http.Handler, base architectureResponse, id, mode, token string) *httptest.ResponseRecorder {
	t.Helper()
	q := url.Values{"project_slug": {base.ProjectSlug}, "store_id": {base.StoreID}, "change_set_id": {id}}
	if mode != "" {
		q.Set(mode, token)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/api/architecture/print?"+q.Encode(), nil))
	return response
}

func TestPrintUsesExactRecordsWithoutPreparingReviewOrSelectingProject(t *testing.T) {
	f := newNativeRefreshFixture(t, true)
	active := testActiveChangeSet(f.state)
	s := active.refObject
	first := printRead(t, f.handler, f.base, active.id, "reviewed_state", s)
	if first.Code != 200 {
		t.Fatal(first.Body.String())
	}
	var printed struct {
		Review reviewResponse `json:"review"`
		State  string         `json:"state"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &printed); err != nil {
		t.Fatal(err)
	}
	expected := reviewResponseForBinding(active.id, *active.review, active.baseSnapshot)
	expected.ReviewedState = s
	if !reflect.DeepEqual(printed.Review, expected) || printed.State != s {
		t.Fatal("print differs from exact Review")
	}
	refs := git(t, "--git-dir", f.storePath, "show-ref")
	for _, mode := range []struct{ key, value string }{{"", ""}, {"reviewed_state", "bad"}, {"applied_state", s}, {"review_id", "missing"}} {
		if r := printRead(t, f.handler, f.base, active.id, mode.key, mode.value); r.Code == 200 {
			t.Fatalf("invalid locator succeeded: %+v", mode)
		}
	}
	// A fresh Handler has no selected project; printing must not create one.
	restarted, handler := newHandler(testOrigin, testUI(t), filepath.Dir(filepath.Dir(f.storePath)))
	if r := printRead(t, handler, f.base, active.id, "reviewed_state", s); r.Code != 200 {
		t.Fatal(r.Body.String())
	}
	if restarted.loadedProject != nil || restarted.loadedSnapshot != nil {
		t.Fatal("print selected project")
	}
	if git(t, "--git-dir", f.storePath, "show-ref") != refs {
		t.Fatal("print changed refs")
	}
	// An unprepared generation cannot become reviewed merely by opening print.
	context := f.state.currentArchitectureResponseLocked()
	selectActiveChangeSetForTest(&context, active.id)
	changed := decodeArchitectureResponse(t, postJSONRequest(t, f.handler, "/api/architecture/components/edit", observedComponentMutation(context, componentMutationRequest{ComponentID: f.component, Description: "New generation.\n", DescriptionChanged: true})))
	if changed.Changes == nil {
		t.Fatal("mutation missing")
	}
	current := testActiveChangeSet(f.state)
	refs = git(t, "--git-dir", f.storePath, "show-ref")
	if r := printRead(t, f.handler, f.base, active.id, "reviewed_state", s); r.Code != 409 {
		t.Fatal("old S retargeted")
	}
	if r := printRead(t, f.handler, f.base, active.id, "reviewed_state", current.refObject); r.Code != 409 {
		t.Fatal("unprepared print succeeded")
	}
	if current.review != nil || git(t, "--git-dir", f.storePath, "show-ref") != refs {
		t.Fatal("print prepared Review")
	}
}

func TestPrintAppliedReceiptAndImmutableSubmissionSurviveLifecycle(t *testing.T) {
	for _, action := range []string{"apply", "discard", "iterate"} {
		t.Run(action, func(t *testing.T) {
			f := newNativeRefreshFixture(t, true)
			active := testActiveChangeSet(f.state)
			s := active.refObject
			binding := active.review
			submitted := decodeArchitectureResponse(t, postJSONRequest(t, f.handler, "/api/architecture/review-submissions/submit", reviewSubmissionRequest{ProjectSlug: f.base.ProjectSlug, StoreID: f.base.StoreID, ChangeSetID: active.id, ReviewedState: s, BaseRevision: binding.baseRevision, CandidateTree: binding.candidateTree, Generation: active.generation, Verdict: "approve", Author: "Print test", Comments: []architecture.ReviewCommentInput{}}))
			if submitted.SubmittedReview == nil {
				t.Fatal("submission missing")
			}
			rid := submitted.SubmittedReview.ID
			original := printRead(t, f.handler, f.base, active.id, "review_id", rid)
			if original.Code != 200 {
				t.Fatal(original.Body.String())
			}
			context := f.state.currentArchitectureResponseLocked()
			selectActiveChangeSetForTest(&context, active.id)
			switch action {
			case "apply":
				result := postJSONRequest(t, f.handler, "/api/architecture/accept", acceptChangesRequest{ProjectSlug: f.base.ProjectSlug, StoreID: f.base.StoreID, ChangeSetID: active.id, BaseRevision: binding.baseRevision, CandidateTree: binding.candidateTree, Generation: active.generation})
				if result.Code != 200 {
					t.Fatal(result.Body.String())
				}
				applied := f.state.changeSets[active.id]
				if applied == nil || applied.lifecycle != "applied" {
					t.Fatal("applied receipt absent")
				}
				r := printRead(t, f.handler, f.base, active.id, "applied_state", applied.refObject)
				var p struct {
					State   string
					Applied string `json:"applied_revision"`
					Review  reviewResponse
				}
				if r.Code != 200 {
					t.Fatal(r.Body.String())
				}
				json.Unmarshal(r.Body.Bytes(), &p)
				if p.State != applied.refObject || p.Applied != applied.appliedRevision || p.Review.ReviewedState != "" || p.Review.BaseRevision != binding.baseRevision {
					t.Fatal("applied context fabricated")
				}
			case "discard":
				r := postJSONRequest(t, f.handler, "/api/architecture/discard", observedAction(context))
				if r.Code != 200 {
					t.Fatal(r.Body.String())
				}
			case "iterate":
				r := postJSONRequest(t, f.handler, "/api/architecture/components/edit", observedComponentMutation(context, componentMutationRequest{ComponentID: f.component, Description: "Later.\n", DescriptionChanged: true}))
				if r.Code != 200 {
					t.Fatal(r.Body.String())
				}
			}
			if r := printRead(t, f.handler, f.base, active.id, "reviewed_state", s); r.Code == 200 {
				t.Fatal("old active state survived movement")
			}
			git(t, "--git-dir", f.storePath, "gc", "--prune=now")
			_, handler := newHandler(testOrigin, testUI(t), filepath.Dir(filepath.Dir(f.storePath)))
			refs := git(t, "--git-dir", f.storePath, "show-ref")
			later := printRead(t, handler, f.base, active.id, "review_id", rid)
			var a, b map[string]any
			json.Unmarshal(original.Body.Bytes(), &a)
			json.Unmarshal(later.Body.Bytes(), &b)
			if later.Code != 200 || !reflect.DeepEqual(a["review"], b["review"]) || a["proposal_markdown"] != b["proposal_markdown"] {
				t.Fatal("historical print changed after lifecycle/GC/restart")
			}
			if git(t, "--git-dir", f.storePath, "show-ref") != refs {
				t.Fatal("historical print changed refs")
			}
		})
	}
}
