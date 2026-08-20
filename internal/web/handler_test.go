package web

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"go.yaml.in/yaml/v4"
	_ "modernc.org/sqlite"

	"workbraid/internal/associations"
)

const testOrigin = "http://127.0.0.1:8080"

func TestOpenProjectUsesRealSQLiteAndLeavesSourceRepositoryUntouched(t *testing.T) {
	db := openWebTestDatabase(t)
	repository := createSourceRepository(t)
	before := snapshotRepository(t, repository)
	handler := newTestHandler(t, db)

	response := postOpenProject(t, handler, testOrigin, repository)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("permissive CORS header = %q", got)
	}
	var body struct {
		SourceRoot  string `json:"source_root"`
		ProjectName string `json:"project_name"`
		Known       bool   `json:"known"`
		StoreID     string `json:"store_id"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.SourceRoot != filepath.Clean(repository) || body.ProjectName != filepath.Base(repository) || body.Known || body.StoreID != "" {
		t.Fatalf("unexpected response: %+v", body)
	}

	var count int
	if err := db.QueryRow(`SELECT count(*) FROM source_architecture_associations`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("open inserted %d associations, want 0", count)
	}
	after := snapshotRepository(t, repository)
	if before != after {
		t.Fatalf("source repository changed\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestInitializeProjectCreatesOneAcceptedBootstrapAndLeavesSourceUntouched(t *testing.T) {
	db := openWebTestDatabase(t)
	repository := createSourceRepository(t)
	before := snapshotRepository(t, repository)
	dataDirectory := t.TempDir()
	handler := NewHandler(db, testOrigin, t.TempDir(), dataDirectory)

	opened := postOpenProject(t, handler, testOrigin, repository)
	if opened.Code != http.StatusOK || !strings.Contains(opened.Body.String(), `"known":false`) {
		t.Fatalf("open status=%d body=%s", opened.Code, opened.Body.String())
	}
	assertAssociationCount(t, db, 0)
	if _, err := os.Stat(filepath.Join(dataDirectory, "architecture")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("opening created private Architecture state: %v", err)
	}

	initialized := postInitializeProject(t, handler, testOrigin, repository)
	if initialized.Code != http.StatusOK {
		t.Fatalf("initialize status=%d body=%s", initialized.Code, initialized.Body.String())
	}
	if got := initialized.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("permissive CORS header = %q", got)
	}
	var result struct {
		SourceRoot     string `json:"source_root"`
		ProjectName    string `json:"project_name"`
		State          string `json:"state"`
		Revision       string `json:"revision"`
		ComponentCount int    `json:"component_count"`
	}
	if err := json.Unmarshal(initialized.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.SourceRoot != filepath.Clean(repository) || result.ProjectName != filepath.Base(repository) || result.State != "empty" || result.Revision == "" || result.ComponentCount != 0 {
		t.Fatalf("unexpected initialization result: %+v", result)
	}

	storeID := associatedStoreID(t, db, filepath.Clean(repository))
	var tables string
	if err := db.QueryRow(`SELECT group_concat(name, ',') FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`).Scan(&tables); err != nil {
		t.Fatal(err)
	}
	if tables != "source_architecture_associations" {
		t.Fatalf("operational database tables = %q", tables)
	}
	if _, err := uuid.Parse(storeID); err != nil {
		t.Fatalf("associated ID is not UUID: %q", storeID)
	}
	storePath := filepath.Join(dataDirectory, "architecture", storeID+".git")
	if got := runGit(t, dataDirectory, "--git-dir", storePath, "rev-parse", "--is-bare-repository"); got != "true" {
		t.Fatalf("bare = %q", got)
	}
	if got := runGit(t, dataDirectory, "--git-dir", storePath, "show-ref", "--verify", "--hash", "refs/heads/accepted"); got != result.Revision {
		t.Fatalf("accepted=%q revision=%q", got, result.Revision)
	}
	if got := runGit(t, dataDirectory, "--git-dir", storePath, "rev-list", "--parents", "-n", "1", result.Revision); got != result.Revision {
		t.Fatalf("bootstrap has a parent: %q", got)
	}
	if got := runGit(t, dataDirectory, "--git-dir", storePath, "ls-tree", result.Revision); !strings.HasPrefix(got, "100644 blob ") || !strings.HasSuffix(got, "\tarchitecture.yaml") || strings.Contains(got, "\n") {
		t.Fatalf("unexpected bootstrap tree: %q", got)
	}
	manifestBytes := []byte(runGit(t, dataDirectory, "--git-dir", storePath, "show", result.Revision+":architecture.yaml"))
	var manifest struct {
		Format  string `yaml:"format"`
		Version int    `yaml:"version"`
		StoreID string `yaml:"store_id"`
		Project struct {
			Name       string `yaml:"name"`
			SourceHint string `yaml:"source_hint"`
		} `yaml:"project"`
	}
	if err := yaml.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Format != "workbraid-architecture" || manifest.Version != 1 || manifest.StoreID != storeID || manifest.Project.Name != filepath.Base(repository) || manifest.Project.SourceHint != filepath.Clean(repository) {
		t.Fatalf("unexpected manifest: %+v", manifest)
	}
	if after := snapshotRepository(t, repository); after != before {
		t.Fatalf("source repository changed\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestNewApplicationInstanceReopensExactAcceptedEmptyArchitecture(t *testing.T) {
	repository := createSourceRepository(t)
	sourceBefore := snapshotRepository(t, repository)
	dataDirectory := t.TempDir()
	databasePath := filepath.Join(dataDirectory, "workbraid.db")
	dbA := openWebDatabaseAt(t, databasePath)
	handlerA := NewHandler(dbA, testOrigin, t.TempDir(), dataDirectory)

	initialized := postInitializeProject(t, handlerA, testOrigin, repository)
	if initialized.Code != http.StatusOK {
		t.Fatalf("initialize status=%d body=%s", initialized.Code, initialized.Body.String())
	}
	var original struct {
		Revision       string `json:"revision"`
		State          string `json:"state"`
		ComponentCount int    `json:"component_count"`
	}
	if err := json.Unmarshal(initialized.Body.Bytes(), &original); err != nil {
		t.Fatal(err)
	}
	if original.Revision == "" || original.State != "empty" || original.ComponentCount != 0 {
		t.Fatalf("unexpected initialized Architecture: %+v", original)
	}
	storeID := associatedStoreID(t, dbA, filepath.Clean(repository))
	storePath := filepath.Join(dataDirectory, "architecture", storeID+".git")
	gitBefore := snapshotPrivateArchitecture(t, dataDirectory)
	associationsBefore := snapshotAssociations(t, dbA)
	if err := dbA.Close(); err != nil {
		t.Fatal(err)
	}

	// A new database connection, manager, handler, and in-memory snapshot prove
	// this is application reconstruction rather than reuse of handler state.
	dbB := openWebDatabaseAt(t, databasePath)
	handlerB := NewHandler(dbB, testOrigin, t.TempDir(), dataDirectory)
	reopened := postOpenProject(t, handlerB, testOrigin, repository)
	if reopened.Code != http.StatusOK {
		t.Fatalf("reopen status=%d body=%s", reopened.Code, reopened.Body.String())
	}
	var loaded struct {
		Revision       string `json:"revision"`
		State          string `json:"state"`
		ComponentCount int    `json:"component_count"`
	}
	if err := json.Unmarshal(reopened.Body.Bytes(), &loaded); err != nil {
		t.Fatal(err)
	}
	if loaded.Revision != original.Revision || loaded.State != "empty" || loaded.ComponentCount != 0 {
		t.Fatalf("reopened Architecture = %+v, original = %+v", loaded, original)
	}
	if accepted := runGit(t, dataDirectory, "--git-dir", storePath, "show-ref", "--verify", "--hash", "refs/heads/accepted"); accepted != loaded.Revision {
		t.Fatalf("accepted=%q reopened=%q", accepted, loaded.Revision)
	}
	if after := snapshotPrivateArchitecture(t, dataDirectory); after != gitBefore {
		t.Fatalf("reopen changed private Architecture\nbefore:\n%s\nafter:\n%s", gitBefore, after)
	}
	if after := snapshotAssociations(t, dbB); after != associationsBefore {
		t.Fatalf("reopen changed associations\nbefore:\n%s\nafter:\n%s", associationsBefore, after)
	}
	if sourceAfter := snapshotRepository(t, repository); sourceAfter != sourceBefore {
		t.Fatalf("source repository changed\nbefore:\n%s\nafter:\n%s", sourceBefore, sourceAfter)
	}
}

func TestNewApplicationInstanceReopensExactAcceptedComponents(t *testing.T) {
	repository := createSourceRepository(t)
	sourceBefore := snapshotRepository(t, repository)
	dataDirectory := t.TempDir()
	databasePath := filepath.Join(dataDirectory, "workbraid.db")
	dbA := openWebDatabaseAt(t, databasePath)
	handlerA := NewHandler(dbA, testOrigin, t.TempDir(), dataDirectory)

	initialized := postInitializeProject(t, handlerA, testOrigin, repository)
	if initialized.Code != http.StatusOK {
		t.Fatalf("initialize status=%d body=%s", initialized.Code, initialized.Body.String())
	}
	var initial struct {
		Revision string `json:"revision"`
	}
	if err := json.Unmarshal(initialized.Body.Bytes(), &initial); err != nil {
		t.Fatal(err)
	}
	storeID := associatedStoreID(t, dbA, filepath.Clean(repository))
	storePath := filepath.Join(dataDirectory, "architecture", storeID+".git")
	manifest := []byte(runGit(t, dataDirectory, "--git-dir", storePath, "show", initial.Revision+":architecture.yaml") + "\n")
	apiID := uuid.NewString()
	workerID := uuid.NewString()
	accepted := advanceAcceptedToComponents(t, storePath, initial.Revision, manifest, []testComponent{
		{
			path:   "arbitrary api.md",
			mode:   "100644",
			source: []byte("---\nid: \"" + apiID + "\"\nrelationships:\n  - target: \"" + workerID + "\"\n    label: calls\n---\n# API\n\nAPI body\n"),
		},
		{
			path:   "worker.md",
			mode:   "100755",
			source: []byte("---\nid: \"" + workerID + "\"\nrelationships:\n  - target: \"" + apiID + "\"\n    label: responds to\n---\nWorker\n======\nWorker body\n"),
		},
	})

	privateBefore := snapshotPrivateArchitecture(t, dataDirectory)
	associationsBefore := snapshotAssociations(t, dbA)
	openedA := postOpenProject(t, handlerA, testOrigin, repository)
	assertComponentInventoryResponse(t, openedA, accepted, []string{"API", "Worker"})
	openedBody := decodeArchitectureResponse(t, openedA)
	if openedBody.Components[0].Filename != "arbitrary api.md" || len(openedBody.Components[0].Relationships) != 1 || openedBody.Components[0].Relationships[0].TargetID != workerID {
		t.Fatalf("accepted projection omitted filename/relationships: %+v", openedBody.Components[0])
	}
	if after := snapshotPrivateArchitecture(t, dataDirectory); after != privateBefore {
		t.Fatalf("component open changed private Architecture\nbefore:\n%s\nafter:\n%s", privateBefore, after)
	}
	if after := snapshotAssociations(t, dbA); after != associationsBefore {
		t.Fatalf("component open changed associations\nbefore:\n%s\nafter:\n%s", associationsBefore, after)
	}
	if err := dbA.Close(); err != nil {
		t.Fatal(err)
	}

	dbB := openWebDatabaseAt(t, databasePath)
	handlerB := NewHandler(dbB, testOrigin, t.TempDir(), dataDirectory)
	reopened := postOpenProject(t, handlerB, testOrigin, repository)
	assertComponentInventoryResponse(t, reopened, accepted, []string{"API", "Worker"})
	if after := snapshotPrivateArchitecture(t, dataDirectory); after != privateBefore {
		t.Fatalf("component reopen changed private Architecture\nbefore:\n%s\nafter:\n%s", privateBefore, after)
	}
	if after := snapshotAssociations(t, dbB); after != associationsBefore {
		t.Fatalf("component reopen changed associations\nbefore:\n%s\nafter:\n%s", associationsBefore, after)
	}
	if sourceAfter := snapshotRepository(t, repository); sourceAfter != sourceBefore {
		t.Fatalf("component reopen changed source repository\nbefore:\n%s\nafter:\n%s", sourceBefore, sourceAfter)
	}
}

func assertComponentInventoryResponse(t *testing.T, response *httptest.ResponseRecorder, revision string, titles []string) {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("open status=%d body=%s", response.Code, response.Body.String())
	}
	var loaded struct {
		State           string   `json:"state"`
		Revision        string   `json:"revision"`
		ComponentCount  int      `json:"component_count"`
		ComponentTitles []string `json:"component_titles"`
		Components      []struct {
			ID          string `json:"id"`
			Title       string `json:"title"`
			Description string `json:"description"`
		} `json:"components"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &loaded); err != nil {
		t.Fatal(err)
	}
	if loaded.State != "ready" || loaded.Revision != revision || loaded.ComponentCount != len(titles) || strings.Join(loaded.ComponentTitles, "|") != strings.Join(titles, "|") {
		t.Fatalf("component inventory response = %+v", loaded)
	}
	if len(loaded.Components) != len(titles) {
		t.Fatalf("structured authoring components = %+v", loaded.Components)
	}
	for _, component := range loaded.Components {
		if _, err := uuid.Parse(component.ID); err != nil || component.Title == "" {
			t.Fatalf("structured authoring component = %+v", component)
		}
	}
}

func TestOpenProjectBoundedAcceptedStateFailuresAreReadOnly(t *testing.T) {
	tests := []struct {
		name       string
		wantStatus int
		wantCode   string
		arrange    func(t *testing.T, dataDirectory, storePath, storeID, revision string)
	}{
		{
			name:       "associated private location missing",
			wantStatus: http.StatusConflict,
			wantCode:   errorArchitectureUnavailable,
			arrange: func(t *testing.T, _, storePath, _, _ string) {
				if err := os.Rename(storePath, storePath+".fixture-away"); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name:       "missing accepted ignores plausible fallback",
			wantStatus: http.StatusConflict,
			wantCode:   errorArchitectureUnavailable,
			arrange: func(t *testing.T, dataDirectory, storePath, _, revision string) {
				runGit(t, dataDirectory, "--git-dir", storePath, "update-ref", "refs/heads/plausible", revision)
				runGit(t, dataDirectory, "--git-dir", storePath, "symbolic-ref", "HEAD", "refs/heads/plausible")
				runGit(t, dataDirectory, "--git-dir", storePath, "update-ref", "-d", "refs/heads/accepted", revision)
			},
		},
		{
			name:       "unsupported manifest version",
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   errorArchitectureUnsupported,
			arrange: func(t *testing.T, dataDirectory, storePath, _, revision string) {
				manifest := runGit(t, dataDirectory, "--git-dir", storePath, "show", revision+":architecture.yaml")
				manifest = strings.Replace(manifest, "version: 1", "version: 3", 1) + "\n"
				advanceAcceptedToManifest(t, storePath, revision, []byte(manifest), nil)
			},
		},
		{
			name:       "manifest identity mismatch",
			wantStatus: http.StatusConflict,
			wantCode:   errorArchitectureInvalid,
			arrange: func(t *testing.T, _ string, storePath, _ string, revision string) {
				manifest := []byte("format: workbraid-architecture\nversion: 1\nstore_id: \"" + uuid.NewString() + "\"\nproject:\n  name: Project\n  source_hint: /tmp/project\n")
				advanceAcceptedToManifest(t, storePath, revision, manifest, nil)
			},
		},
		{
			name:       "component with unresolved relationship is invalid",
			wantStatus: http.StatusConflict,
			wantCode:   errorArchitectureInvalid,
			arrange: func(t *testing.T, dataDirectory, storePath, _ string, revision string) {
				manifest := []byte(runGit(t, dataDirectory, "--git-dir", storePath, "show", revision+":architecture.yaml") + "\n")
				component := []byte("---\nid: \"" + uuid.NewString() + "\"\nrelationships:\n  - target: \"" + uuid.NewString() + "\"\n    label: calls\n---\n# Component\n")
				advanceAcceptedToManifest(t, storePath, revision, manifest, component)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := createSourceRepository(t)
			sourceBefore := snapshotRepository(t, repository)
			dataDirectory := t.TempDir()
			databasePath := filepath.Join(dataDirectory, "workbraid.db")
			dbA := openWebDatabaseAt(t, databasePath)
			handlerA := NewHandler(dbA, testOrigin, t.TempDir(), dataDirectory)
			initialized := postInitializeProject(t, handlerA, testOrigin, repository)
			if initialized.Code != http.StatusOK {
				t.Fatalf("initialize status=%d body=%s", initialized.Code, initialized.Body.String())
			}
			var initial struct {
				Revision string `json:"revision"`
			}
			if err := json.Unmarshal(initialized.Body.Bytes(), &initial); err != nil {
				t.Fatal(err)
			}
			storeID := associatedStoreID(t, dbA, filepath.Clean(repository))
			storePath := filepath.Join(dataDirectory, "architecture", storeID+".git")
			test.arrange(t, dataDirectory, storePath, storeID, initial.Revision)
			privateBefore := snapshotPrivateArchitecture(t, dataDirectory)
			associationsBefore := snapshotAssociations(t, dbA)
			if err := dbA.Close(); err != nil {
				t.Fatal(err)
			}

			dbB := openWebDatabaseAt(t, databasePath)
			handlerB := NewHandler(dbB, testOrigin, t.TempDir(), dataDirectory)
			response := postOpenProject(t, handlerB, testOrigin, repository)
			if response.Code != test.wantStatus || !strings.Contains(response.Body.String(), `"code":"`+test.wantCode+`"`) {
				t.Fatalf("open status=%d body=%s", response.Code, response.Body.String())
			}
			if strings.Contains(response.Body.String(), `"state":`) || strings.Contains(response.Body.String(), `"revision":`) || strings.Contains(response.Body.String(), `"component_titles":`) {
				t.Fatalf("failed open presented accepted Architecture: %s", response.Body.String())
			}
			if got := response.Header().Get("Access-Control-Allow-Origin"); got != "" {
				t.Fatalf("permissive CORS header = %q", got)
			}
			if privateAfter := snapshotPrivateArchitecture(t, dataDirectory); privateAfter != privateBefore {
				t.Fatalf("failed open changed private Architecture\nbefore:\n%s\nafter:\n%s", privateBefore, privateAfter)
			}
			if associationsAfter := snapshotAssociations(t, dbB); associationsAfter != associationsBefore {
				t.Fatalf("failed open changed associations\nbefore:\n%s\nafter:\n%s", associationsBefore, associationsAfter)
			}
			if sourceAfter := snapshotRepository(t, repository); sourceAfter != sourceBefore {
				t.Fatalf("failed open changed source repository\nbefore:\n%s\nafter:\n%s", sourceBefore, sourceAfter)
			}
		})
	}
}

func TestInitializeProjectRetainsAssociationAndRetriesSameStore(t *testing.T) {
	db := openWebTestDatabase(t)
	repository := createSourceRepository(t)
	before := snapshotRepository(t, repository)
	dataDirectory := t.TempDir()
	blockingPath := filepath.Join(dataDirectory, "architecture")
	if err := os.WriteFile(blockingPath, []byte("block"), 0o600); err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(db, testOrigin, t.TempDir(), dataDirectory)

	failed := postInitializeProject(t, handler, testOrigin, repository)
	if failed.Code != http.StatusInternalServerError || !strings.Contains(failed.Body.String(), `"code":"setup_incomplete"`) {
		t.Fatalf("failed status=%d body=%s", failed.Code, failed.Body.String())
	}
	storeID := associatedStoreID(t, db, filepath.Clean(repository))
	assertAssociationCount(t, db, 1)
	if err := os.Remove(blockingPath); err != nil {
		t.Fatal(err)
	}

	retried := postInitializeProject(t, handler, testOrigin, repository)
	if retried.Code != http.StatusOK {
		t.Fatalf("retry status=%d body=%s", retried.Code, retried.Body.String())
	}
	if got := associatedStoreID(t, db, filepath.Clean(repository)); got != storeID {
		t.Fatalf("retry replaced store ID %q with %q", storeID, got)
	}
	storePath := filepath.Join(dataDirectory, "architecture", storeID+".git")
	if commits := runGit(t, dataDirectory, "--git-dir", storePath, "rev-list", "--all", "--count"); commits != "1" {
		t.Fatalf("commit count = %s, want 1", commits)
	}
	if after := snapshotRepository(t, repository); after != before {
		t.Fatalf("source repository changed\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestConcurrentInitializationUsesOneAssociationAndStore(t *testing.T) {
	db := openWebTestDatabase(t)
	repository := createSourceRepository(t)
	dataDirectory := t.TempDir()
	handler := NewHandler(db, testOrigin, t.TempDir(), dataDirectory)

	start := make(chan struct{})
	responses := make(chan *httptest.ResponseRecorder, 2)
	var wait sync.WaitGroup
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			responses <- postInitializeProject(t, handler, testOrigin, repository)
		}()
	}
	close(start)
	wait.Wait()
	close(responses)
	for response := range responses {
		if response.Code != http.StatusOK {
			t.Fatalf("initialize status=%d body=%s", response.Code, response.Body.String())
		}
	}
	assertAssociationCount(t, db, 1)
	storeID := associatedStoreID(t, db, filepath.Clean(repository))
	entries, err := os.ReadDir(filepath.Join(dataDirectory, "architecture"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != storeID+".git" {
		t.Fatalf("private stores = %v, want only %s.git", entries, storeID)
	}
}

func TestInitializeProjectEnforcesOriginWithoutCORS(t *testing.T) {
	db := openWebTestDatabase(t)
	handler := newTestHandler(t, db)
	repository := t.TempDir()
	for _, origin := range []string{"", "http://attacker.example"} {
		response := postInitializeProject(t, handler, origin, repository)
		if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), `"code":"origin_mismatch"`) {
			t.Fatalf("origin %q: status=%d body=%s", origin, response.Code, response.Body.String())
		}
		if got := response.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Fatalf("origin %q: permissive CORS = %q", origin, got)
		}
	}
	assertAssociationCount(t, db, 0)
}

func TestInitializeProjectReportsInvalidAndUnsupportedAcceptedStates(t *testing.T) {
	for _, test := range []struct {
		name       string
		wantStatus int
		wantCode   string
		buildTree  func(t *testing.T, repository, manifestBlob string) string
	}{
		{
			name:       "identity mismatch is invalid",
			wantStatus: http.StatusConflict,
			wantCode:   "architecture_invalid",
			buildTree: func(t *testing.T, repository, _ string) string {
				wrongManifest := "format: workbraid-architecture\nversion: 1\nstore_id: \"" + uuid.NewString() + "\"\nproject:\n  name: Project\n  source_hint: /tmp/project\n"
				blob := runGitWithInput(t, repository, []byte(wrongManifest), "--git-dir", repository, "hash-object", "-w", "--stdin")
				return runGitWithInput(t, repository, []byte("100644 blob "+blob+"\tarchitecture.yaml\n"), "--git-dir", repository, "mktree")
			},
		},
		{
			name:       "empty components tree is invalid",
			wantStatus: http.StatusConflict,
			wantCode:   "architecture_invalid",
			buildTree: func(t *testing.T, repository, manifestBlob string) string {
				emptyComponentsTree := runGitWithInput(t, repository, nil, "--git-dir", repository, "mktree")
				root := "100644 blob " + manifestBlob + "\tarchitecture.yaml\n040000 tree " + emptyComponentsTree + "\tcomponents\n"
				return runGitWithInput(t, repository, []byte(root), "--git-dir", repository, "mktree")
			},
		},
		{
			name:       "non-string recovery hints are invalid",
			wantStatus: http.StatusConflict,
			wantCode:   "architecture_invalid",
			buildTree: func(t *testing.T, repository, _ string) string {
				storeID := strings.TrimSuffix(filepath.Base(repository), ".git")
				typedManifest := "format: workbraid-architecture\nversion: 1\nstore_id: \"" + storeID + "\"\nproject:\n  name: 123\n  source_hint: true\n"
				blob := runGitWithInput(t, repository, []byte(typedManifest), "--git-dir", repository, "hash-object", "-w", "--stdin")
				return runGitWithInput(t, repository, []byte("100644 blob "+blob+"\tarchitecture.yaml\n"), "--git-dir", repository, "mktree")
			},
		},
		{
			name:       "invalid component is rejected",
			wantStatus: http.StatusConflict,
			wantCode:   "architecture_invalid",
			buildTree: func(t *testing.T, repository, manifestBlob string) string {
				component := []byte("---\nid: \"" + uuid.NewString() + "\"\nrelationships:\n  - target: \"" + uuid.NewString() + "\"\n    label: calls\n---\n# Component\n")
				componentBlob := runGitWithInput(t, repository, component, "--git-dir", repository, "hash-object", "-w", "--stdin")
				componentTree := runGitWithInput(t, repository, []byte("100644 blob "+componentBlob+"\tcomponent.md\n"), "--git-dir", repository, "mktree")
				root := "100644 blob " + manifestBlob + "\tarchitecture.yaml\n040000 tree " + componentTree + "\tcomponents\n"
				return runGitWithInput(t, repository, []byte(root), "--git-dir", repository, "mktree")
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			db := openWebTestDatabase(t)
			source := t.TempDir()
			dataDirectory := t.TempDir()
			handler := NewHandler(db, testOrigin, t.TempDir(), dataDirectory)
			initialized := postInitializeProject(t, handler, testOrigin, source)
			if initialized.Code != http.StatusOK {
				t.Fatalf("initialize status=%d body=%s", initialized.Code, initialized.Body.String())
			}
			storeID := associatedStoreID(t, db, filepath.Clean(source))
			repository := filepath.Join(dataDirectory, "architecture", storeID+".git")
			oldRevision := runGit(t, dataDirectory, "--git-dir", repository, "show-ref", "--verify", "--hash", "refs/heads/accepted")
			manifestBlob := strings.Fields(runGit(t, dataDirectory, "--git-dir", repository, "ls-tree", oldRevision, "architecture.yaml"))[2]
			tree := test.buildTree(t, repository, manifestBlob)
			commit := runGitWithInput(t, repository, []byte("external accepted state\n"), "-c", "user.name=Test", "-c", "user.email=test@workbraid.invalid", "--git-dir", repository, "commit-tree", tree)
			runGit(t, dataDirectory, "--git-dir", repository, "update-ref", "refs/heads/accepted", commit, oldRevision)

			response := postInitializeProject(t, handler, testOrigin, source)
			if response.Code != test.wantStatus || !strings.Contains(response.Body.String(), `"code":"`+test.wantCode+`"`) {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			if accepted := runGit(t, dataDirectory, "--git-dir", repository, "show-ref", "--verify", "--hash", "refs/heads/accepted"); accepted != commit {
				t.Fatalf("failed load changed accepted from %q to %q", commit, accepted)
			}
		})
	}
}

func TestOpenProjectDoesNotInitializeAPreseededMissingArchitecture(t *testing.T) {
	db := openWebTestDatabase(t)
	repository := t.TempDir()
	const storeID = "a0b38e04-54bd-464d-8a8f-8f2e78e653ea"
	if _, err := db.Exec(
		`INSERT INTO source_architecture_associations(normalized_source_root, store_id) VALUES (?, ?)`,
		filepath.Clean(repository), storeID,
	); err != nil {
		t.Fatal(err)
	}

	dataDirectory := t.TempDir()
	response := postOpenProject(t, NewHandler(db, testOrigin, t.TempDir(), dataDirectory), testOrigin, repository)
	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"code":"architecture_unavailable"`) {
		t.Fatalf("response does not report unavailable Architecture: %s", response.Body.String())
	}
	storePath := filepath.Join(dataDirectory, "architecture", storeID+".git")
	if _, err := os.Stat(storePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("open created or changed missing private Architecture location: %v", err)
	}
	if got := associatedStoreID(t, db, filepath.Clean(repository)); got != storeID {
		t.Fatalf("open changed associated ID to %q", got)
	}
}

func TestOpenProjectReportsOperationalDatabaseFailure(t *testing.T) {
	db := openWebTestDatabase(t)
	handler := newTestHandler(t, db)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	response := postOpenProject(t, handler, testOrigin, t.TempDir())
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"code":"lookup_failed"`) {
		t.Fatalf("unexpected error body: %s", response.Body.String())
	}
}

func TestOpenProjectRejectsUnexpectedOrMissingOriginWithoutCORS(t *testing.T) {
	db := openWebTestDatabase(t)
	handler := newTestHandler(t, db)
	repository := t.TempDir()

	for _, origin := range []string{"", "http://attacker.example"} {
		response := postOpenProject(t, handler, origin, repository)
		if response.Code != http.StatusForbidden {
			t.Fatalf("origin %q: status = %d, want 403", origin, response.Code)
		}
		if got := response.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Fatalf("origin %q: permissive CORS header = %q", origin, got)
		}
		if !strings.Contains(response.Body.String(), `"code":"origin_mismatch"`) {
			t.Fatalf("origin %q: unexpected body %s", origin, response.Body.String())
		}
	}
}

func TestOpenProjectRejectsInvalidPaths(t *testing.T) {
	db := openWebTestDatabase(t)
	handler := newTestHandler(t, db)
	regularFile := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(regularFile, []byte("file"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		path string
		code string
	}{
		{name: "empty", path: "", code: "path_required"},
		{name: "relative", path: "relative/project", code: "path_relative"},
		{name: "missing", path: filepath.Join(t.TempDir(), "missing"), code: "path_missing"},
		{name: "regular file", path: regularFile, code: "path_not_directory"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := postOpenProject(t, handler, testOrigin, test.path)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			if !strings.Contains(response.Body.String(), `"code":"`+test.code+`"`) {
				t.Fatalf("body = %s, want code %q", response.Body.String(), test.code)
			}
		})
	}
}

func TestOpenProjectMapsMalformedJSONToGenericFailureCode(t *testing.T) {
	db := openWebTestDatabase(t)
	handler := newTestHandler(t, db)

	for _, body := range []string{`{"source_root":`, `{"source_root":"/tmp"} {}`} {
		request := httptest.NewRequest(http.MethodPost, "/api/projects/open", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Origin", testOrigin)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("body %q: status = %d, response = %s", body, response.Code, response.Body.String())
		}
		if !strings.Contains(response.Body.String(), `"code":"lookup_failed"`) {
			t.Fatalf("body %q: response = %s", body, response.Body.String())
		}
	}
}

func TestHandlerServesBuiltUI(t *testing.T) {
	db := openWebTestDatabase(t)
	uiDirectory := t.TempDir()
	if err := os.WriteFile(filepath.Join(uiDirectory, "index.html"), []byte("<main>WorkBraid</main>"), 0o600); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	NewHandler(db, testOrigin, uiDirectory, t.TempDir()).ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "WorkBraid") {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func postOpenProject(t *testing.T, handler http.Handler, origin, sourceRoot string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]string{"source_root": sourceRoot})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/projects/open", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if origin != "" {
		request.Header.Set("Origin", origin)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func postInitializeProject(t *testing.T, handler http.Handler, origin, sourceRoot string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]string{"source_root": sourceRoot})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/projects/initialize", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if origin != "" {
		request.Header.Set("Origin", origin)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func associatedStoreID(t *testing.T, db *sql.DB, sourceRoot string) string {
	t.Helper()
	var storeID string
	if err := db.QueryRow(`SELECT store_id FROM source_architecture_associations WHERE normalized_source_root = ?`, sourceRoot).Scan(&storeID); err != nil {
		t.Fatal(err)
	}
	return storeID
}

func assertAssociationCount(t *testing.T, db *sql.DB, want int) {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM source_architecture_associations`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("association count = %d, want %d", count, want)
	}
}

func snapshotAssociations(t *testing.T, db *sql.DB) string {
	t.Helper()
	rows, err := db.Query(`SELECT normalized_source_root, store_id FROM source_architecture_associations ORDER BY normalized_source_root`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var values []string
	for rows.Next() {
		var sourceRoot, storeID string
		if err := rows.Scan(&sourceRoot, &storeID); err != nil {
			t.Fatal(err)
		}
		values = append(values, sourceRoot+"\x00"+storeID)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return strings.Join(values, "\n")
}

func snapshotPrivateArchitecture(t *testing.T, dataDirectory string) string {
	t.Helper()
	root := filepath.Join(dataDirectory, "architecture")
	if _, err := os.Stat(root); errors.Is(err, os.ErrNotExist) {
		return "<missing>"
	} else if err != nil {
		t.Fatal(err)
	}
	var entries []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			entries = append(entries, "directory "+relative+" "+info.Mode().String())
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(contents)
		entries = append(entries, "file "+relative+" "+info.Mode().String()+" "+hex.EncodeToString(sum[:]))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(entries)
	return strings.Join(entries, "\n")
}

func advanceAcceptedToManifest(t *testing.T, storePath, oldRevision string, manifest, component []byte) string {
	t.Helper()
	manifestBlob := runGitWithInput(t, storePath, manifest, "--git-dir", storePath, "hash-object", "-w", "--stdin")
	rootEntries := "100644 blob " + manifestBlob + "\tarchitecture.yaml\n"
	if component != nil {
		componentBlob := runGitWithInput(t, storePath, component, "--git-dir", storePath, "hash-object", "-w", "--stdin")
		componentTree := runGitWithInput(t, storePath, []byte("100644 blob "+componentBlob+"\tcomponent.md\n"), "--git-dir", storePath, "mktree")
		rootEntries += "040000 tree " + componentTree + "\tcomponents\n"
	}
	tree := runGitWithInput(t, storePath, []byte(rootEntries), "--git-dir", storePath, "mktree")
	commit := runGitWithInput(t, storePath, []byte("external accepted state\n"),
		"-c", "user.name=Test", "-c", "user.email=test@workbraid.invalid",
		"--git-dir", storePath, "commit-tree", tree, "-p", oldRevision)
	runGit(t, storePath, "--git-dir", storePath, "update-ref", "refs/heads/accepted", commit, oldRevision)
	return commit
}

type testComponent struct {
	path   string
	mode   string
	source []byte
}

func advanceAcceptedToComponents(t *testing.T, storePath, oldRevision string, manifest []byte, components []testComponent) string {
	t.Helper()
	manifestBlob := runGitWithInput(t, storePath, manifest, "--git-dir", storePath, "hash-object", "-w", "--stdin")
	componentEntries := make([]string, len(components))
	for index, component := range components {
		blob := runGitWithInput(t, storePath, component.source, "--git-dir", storePath, "hash-object", "-w", "--stdin")
		componentEntries[index] = component.mode + " blob " + blob + "\t" + component.path
	}
	componentTree := runGitWithInput(t, storePath, []byte(strings.Join(componentEntries, "\n")+"\n"), "--git-dir", storePath, "mktree")
	rootEntries := "100644 blob " + manifestBlob + "\tarchitecture.yaml\n040000 tree " + componentTree + "\tcomponents\n"
	tree := runGitWithInput(t, storePath, []byte(rootEntries), "--git-dir", storePath, "mktree")
	commit := runGitWithInput(t, storePath, []byte("external accepted components\n"),
		"-c", "user.name=Test", "-c", "user.email=test@workbraid.invalid",
		"--git-dir", storePath, "commit-tree", tree, "-p", oldRevision)
	runGit(t, storePath, "--git-dir", storePath, "update-ref", "refs/heads/accepted", commit, oldRevision)
	return commit
}

func openWebTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	return openWebDatabaseAt(t, filepath.Join(t.TempDir(), "workbraid.db"))
}

func openWebDatabaseAt(t *testing.T, path string) *sql.DB {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	if err := associations.Initialize(db); err != nil {
		db.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func newTestHandler(t *testing.T, db *sql.DB) http.Handler {
	t.Helper()
	return NewHandler(db, testOrigin, t.TempDir(), t.TempDir())
}

func createSourceRepository(t *testing.T) string {
	t.Helper()
	repository := t.TempDir()
	runGit(t, repository, "init", "--quiet")
	if err := os.WriteFile(filepath.Join(repository, "tracked.txt"), []byte("tracked\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, repository, "add", "tracked.txt")
	runGit(t, repository, "-c", "user.name=WorkBraid Test", "-c", "user.email=test@workbraid.invalid", "commit", "--quiet", "-m", "test source")
	if err := os.WriteFile(filepath.Join(repository, "untracked.txt"), []byte("untracked\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return repository
}

func runGit(t *testing.T, directory string, arguments ...string) string {
	t.Helper()
	command := exec.CommandContext(context.Background(), "git", arguments...)
	command.Dir = directory
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", arguments, err, output)
	}
	return strings.TrimSpace(string(output))
}

func runGitWithInput(t *testing.T, directory string, input []byte, arguments ...string) string {
	t.Helper()
	command := exec.CommandContext(context.Background(), "git", arguments...)
	command.Dir = directory
	command.Stdin = bytes.NewReader(input)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", arguments, err, output)
	}
	return strings.TrimSpace(string(output))
}

func snapshotRepository(t *testing.T, repository string) string {
	t.Helper()
	head := runGit(t, repository, "rev-parse", "HEAD")
	status := runGit(t, repository, "status", "--short")
	var files []string
	err := filepath.WalkDir(repository, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(repository, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if relative == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(contents)
		files = append(files, relative+":"+hex.EncodeToString(sum[:]))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	return strings.Join([]string{head, status, strings.Join(files, "\n")}, "\n---\n")
}
