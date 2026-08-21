package main

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"go.yaml.in/yaml/v4"
)

func TestOriginForLoopbackAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		address string
		want    string
		wantErr bool
	}{
		{name: "IPv4", address: "127.0.0.1:8080", want: "http://127.0.0.1:8080"},
		{name: "IPv6", address: "[::1]:8080", want: "http://[::1]:8080"},
		{name: "all interfaces", address: "0.0.0.0:8080", wantErr: true},
		{name: "host name", address: "localhost:8080", wantErr: true},
		{name: "dynamic port", address: "127.0.0.1:0", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := originForLoopbackAddress(test.address)
			if test.wantErr {
				if err == nil {
					t.Fatalf("originForLoopbackAddress(%q) unexpectedly succeeded", test.address)
				}
				return
			}
			if err != nil {
				t.Fatalf("originForLoopbackAddress(%q): %v", test.address, err)
			}
			if got != test.want {
				t.Fatalf("originForLoopbackAddress(%q) = %q, want %q", test.address, got, test.want)
			}
		})
	}
}

func TestRealProcessRestartReopensExactAcceptedComponents(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and restarts the real WorkBraid process")
	}

	binary := filepath.Join(t.TempDir(), "workbraid")
	build := exec.Command("go", "build", "-buildvcs=false", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build WorkBraid: %v\n%s", err, output)
	}
	dataDirectory := t.TempDir()
	uiDirectory := t.TempDir()
	if err := os.WriteFile(filepath.Join(uiDirectory, "index.html"), []byte("<main>WorkBraid</main>"), 0o600); err != nil {
		t.Fatal(err)
	}
	source := createProcessTestSourceRepository(t)
	sourceBefore := snapshotProcessTestSource(t, source)

	processA := startWorkBraidProcess(t, binary, dataDirectory, uiDirectory)
	opened := postProcessJSON(t, processA.origin, "/api/projects/open", source)
	if opened.status != http.StatusOK || !strings.Contains(opened.body, `"known":false`) {
		t.Fatalf("initial open status=%d body=%s", opened.status, opened.body)
	}
	initialized := postProcessJSON(t, processA.origin, "/api/projects/initialize", source)
	if initialized.status != http.StatusOK {
		t.Fatalf("initialize status=%d body=%s", initialized.status, initialized.body)
	}
	var first struct {
		Revision       string `json:"revision"`
		State          string `json:"state"`
		ComponentCount int    `json:"component_count"`
	}
	if err := json.Unmarshal([]byte(initialized.body), &first); err != nil {
		t.Fatal(err)
	}
	if first.Revision == "" || first.State != "empty" || first.ComponentCount != 0 {
		t.Fatalf("unexpected initialized Architecture: %+v", first)
	}
	processA.stop(t)
	if sourceAfterShutdown := snapshotProcessTestSource(t, source); sourceAfterShutdown != sourceBefore {
		t.Fatalf("source repository changed before restart\nbefore:\n%s\nafter process A:\n%s", sourceBefore, sourceAfterShutdown)
	}

	storeID := processTestStoreID(t, filepath.Join(dataDirectory, "workbraid.db"), filepath.Clean(source))
	storePath := filepath.Join(dataDirectory, "architecture", storeID+".git")
	if accepted := processTestGit(t, source, "--git-dir", storePath, "show-ref", "--verify", "--hash", "refs/heads/accepted"); accepted != first.Revision {
		t.Fatalf("accepted=%q initialized=%q", accepted, first.Revision)
	}
	acceptedComponents := processTestAdvanceAcceptedComponents(t, storePath, first.Revision)
	t.Logf("accepted component fixture revision: %s", acceptedComponents)

	processB := startWorkBraidProcess(t, binary, dataDirectory, uiDirectory)
	reopened := postProcessJSON(t, processB.origin, "/api/projects/open", source)
	if reopened.status != http.StatusOK {
		t.Fatalf("reopen status=%d body=%s", reopened.status, reopened.body)
	}
	var second struct {
		Revision        string   `json:"revision"`
		State           string   `json:"state"`
		ComponentCount  int      `json:"component_count"`
		ComponentTitles []string `json:"component_titles"`
	}
	if err := json.Unmarshal([]byte(reopened.body), &second); err != nil {
		t.Fatal(err)
	}
	if second.Revision != acceptedComponents || second.State != "ready" || second.ComponentCount != 2 || strings.Join(second.ComponentTitles, "|") != "API|Worker" {
		t.Fatalf("restarted component Architecture=%+v accepted=%q", second, acceptedComponents)
	}
	processB.stop(t)
	if accepted := processTestGit(t, source, "--git-dir", storePath, "show-ref", "--verify", "--hash", "refs/heads/accepted"); accepted != acceptedComponents {
		t.Fatalf("component load changed accepted from %q to %q", acceptedComponents, accepted)
	}
	if storeIDAfter := processTestStoreID(t, filepath.Join(dataDirectory, "workbraid.db"), filepath.Clean(source)); storeIDAfter != storeID {
		t.Fatalf("restart changed architecture link from %q to %q", storeID, storeIDAfter)
	}

	if sourceAfter := snapshotProcessTestSource(t, source); sourceAfter != sourceBefore {
		t.Fatalf("source repository changed across process restart\nbefore:\n%s\nafter:\n%s", sourceBefore, sourceAfter)
	}
}

func processTestAdvanceAcceptedComponents(t *testing.T, storePath, parent string) string {
	t.Helper()
	manifest := []byte(processTestGit(t, storePath, "--git-dir", storePath, "show", parent+":architecture.yaml") + "\n")
	var parsedManifest struct {
		RootDiagram string `yaml:"root_diagram"`
	}
	if err := yaml.Unmarshal(manifest, &parsedManifest); err != nil || parsedManifest.RootDiagram == "" {
		t.Fatalf("parse bootstrap root Diagram: root=%q err=%v", parsedManifest.RootDiagram, err)
	}
	const apiID = "11111111-1111-4111-8111-111111111111"
	const workerID = "22222222-2222-4222-8222-222222222222"
	api := []byte("---\nid: \"" + apiID + "\"\nrelationships:\n  - target: \"" + workerID + "\"\n    label: calls\n---\n# API\n\nAPI body\n")
	worker := []byte("---\nid: \"" + workerID + "\"\nrelationships: []\n---\nWorker\n======\nWorker body\n")
	manifestBlob := processTestGitInput(t, storePath, manifest, "--git-dir", storePath, "hash-object", "-w", "--stdin")
	apiBlob := processTestGitInput(t, storePath, api, "--git-dir", storePath, "hash-object", "-w", "--stdin")
	workerBlob := processTestGitInput(t, storePath, worker, "--git-dir", storePath, "hash-object", "-w", "--stdin")
	componentTree := processTestGitInput(t, storePath, []byte(
		"100644 blob "+apiBlob+"\tapi.md\n100755 blob "+workerBlob+"\tworker.md\n",
	), "--git-dir", storePath, "mktree")
	rootDiagram := []byte("id: \"" + parsedManifest.RootDiagram + "\"\ntitle: Project\nappearances:\n  - component: \"" + apiID + "\"\n    role: home\n  - component: \"" + workerID + "\"\n    role: home\n")
	rootDiagramBlob := processTestGitInput(t, storePath, rootDiagram, "--git-dir", storePath, "hash-object", "-w", "--stdin")
	diagramTree := processTestGitInput(t, storePath, []byte("100644 blob "+rootDiagramBlob+"\troot.yaml\n"), "--git-dir", storePath, "mktree")
	rootTree := processTestGitInput(t, storePath, []byte(
		"100644 blob "+manifestBlob+"\tarchitecture.yaml\n040000 tree "+componentTree+"\tcomponents\n040000 tree "+diagramTree+"\tdiagrams\n",
	), "--git-dir", storePath, "mktree")
	commit := processTestGitInput(t, storePath, []byte("Add accepted components\n"),
		"-c", "user.name=WorkBraid Test", "-c", "user.email=test@workbraid.invalid",
		"--git-dir", storePath, "commit-tree", rootTree, "-p", parent)
	processTestGit(t, storePath, "--git-dir", storePath, "update-ref", "refs/heads/accepted", commit, parent)
	return commit
}

type workBraidProcess struct {
	command *exec.Cmd
	done    chan error
	logs    *bytes.Buffer
	origin  string
	stopped bool
}

func startWorkBraidProcess(t *testing.T, binary, dataDirectory, uiDirectory string) *workBraidProcess {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	origin := "http://" + address
	logs := &bytes.Buffer{}
	command := exec.Command(binary, "-listen", address, "-data-dir", dataDirectory, "-ui-dir", uiDirectory)
	command.Stdout = logs
	command.Stderr = logs
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	running := &workBraidProcess{command: command, done: make(chan error, 1), logs: logs, origin: origin}
	go func() { running.done <- command.Wait() }()
	t.Cleanup(func() { running.stop(t) })

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		response, err := http.Get(origin + "/")
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return running
			}
		}
		select {
		case err := <-running.done:
			running.stopped = true
			t.Fatalf("WorkBraid exited before readiness: %v\n%s", err, logs.String())
		default:
		}
		time.Sleep(25 * time.Millisecond)
	}
	running.stop(t)
	t.Fatalf("WorkBraid did not become ready\n%s", logs.String())
	return nil
}

func (process *workBraidProcess) stop(t *testing.T) {
	t.Helper()
	if process == nil || process.stopped {
		return
	}
	process.stopped = true
	if err := process.command.Process.Signal(os.Interrupt); err != nil {
		t.Fatalf("stop WorkBraid: %v", err)
	}
	select {
	case <-process.done:
		return
	case <-time.After(5 * time.Second):
		if err := process.command.Process.Kill(); err != nil {
			t.Fatalf("kill WorkBraid after stop timeout: %v", err)
		}
		<-process.done
	}
}

type processResponse struct {
	status int
	body   string
}

func postProcessJSON(t *testing.T, origin, path, sourceRoot string) processResponse {
	t.Helper()
	body, err := json.Marshal(map[string]string{"source_root": sourceRoot})
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPost, origin+path, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", origin)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var responseBody bytes.Buffer
	if _, err := responseBody.ReadFrom(response.Body); err != nil {
		t.Fatal(err)
	}
	return processResponse{status: response.StatusCode, body: responseBody.String()}
}

func createProcessTestSourceRepository(t *testing.T) string {
	t.Helper()
	repository := t.TempDir()
	processTestGit(t, repository, "init", "--quiet")
	if err := os.WriteFile(filepath.Join(repository, "tracked.txt"), []byte("tracked\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	processTestGit(t, repository, "add", "tracked.txt")
	processTestGit(t, repository, "-c", "user.name=WorkBraid Test", "-c", "user.email=test@workbraid.invalid", "commit", "--quiet", "-m", "source fixture")
	if err := os.WriteFile(filepath.Join(repository, "untracked.txt"), []byte("untracked\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return repository
}

func snapshotProcessTestSource(t *testing.T, repository string) string {
	t.Helper()
	head := processTestGit(t, repository, "rev-parse", "HEAD")
	status := processTestGit(t, repository, "status", "--short")
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

func processTestStoreID(t *testing.T, databasePath, sourceRoot string) string {
	t.Helper()
	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var storeID string
	if err := db.QueryRow(`SELECT store_id FROM source_architecture_associations WHERE normalized_source_root = ?`, sourceRoot).Scan(&storeID); err != nil {
		t.Fatal(err)
	}
	return storeID
}

func processTestGit(t *testing.T, directory string, arguments ...string) string {
	t.Helper()
	command := exec.Command("git", arguments...)
	command.Dir = directory
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", arguments, err, output)
	}
	return strings.TrimSpace(string(output))
}

func processTestGitInput(t *testing.T, directory string, input []byte, arguments ...string) string {
	t.Helper()
	command := exec.Command("git", arguments...)
	command.Dir = directory
	command.Stdin = bytes.NewReader(input)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", arguments, err, output)
	}
	return strings.TrimSpace(string(output))
}
