package architecture

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type gitRunner struct{}

type treeEntry struct {
	Mode   string
	Type   string
	Object string
	Path   string
}

type refEntry struct {
	Name   string
	Object string
}

func (gitRunner) initBare(ctx context.Context, repository string) error {
	_, err := runGit(ctx, nil, "init", "--bare", "--quiet", "--template=", repository)
	return err
}

func (gitRunner) isBare(ctx context.Context, repository string) (bool, error) {
	output, err := runGit(ctx, nil, "--git-dir", repository, "rev-parse", "--is-bare-repository")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(output)) == "true", nil
}

func (gitRunner) resolveRef(ctx context.Context, repository, ref string) (string, bool, error) {
	output, err := runGit(ctx, nil, "--git-dir", repository, "for-each-ref", "--format=%(refname)%00%(objectname)", ref)
	if err != nil {
		return "", false, err
	}
	for _, record := range bytes.Split(output, []byte{'\n'}) {
		name, object, found := bytes.Cut(record, []byte{0})
		if found && string(name) == ref {
			return strings.TrimSpace(string(object)), true, nil
		}
	}
	return "", false, nil
}

func (gitRunner) refs(ctx context.Context, repository string) ([]string, error) {
	output, err := runGit(ctx, nil, "--git-dir", repository, "for-each-ref", "--format=%(refname)")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(output)) == "" {
		return nil, nil
	}
	return strings.Fields(string(output)), nil
}

func (gitRunner) refsUnder(ctx context.Context, repository, prefix string) ([]refEntry, error) {
	output, err := runGit(ctx, nil, "--git-dir", repository, "for-each-ref", "--format=%(refname)%00%(objectname)", prefix)
	if err != nil {
		return nil, err
	}
	var refs []refEntry
	for _, record := range bytes.Split(output, []byte{'\n'}) {
		if len(record) == 0 {
			continue
		}
		name, object, found := bytes.Cut(record, []byte{0})
		if !found {
			return nil, errors.New("Git returned a malformed ref entry")
		}
		refs = append(refs, refEntry{Name: string(name), Object: strings.TrimSpace(string(object))})
	}
	return refs, nil
}

func (gitRunner) writeBlob(ctx context.Context, repository string, contents []byte) (string, error) {
	output, err := runGit(ctx, contents, "--git-dir", repository, "hash-object", "-w", "--stdin")
	return strings.TrimSpace(string(output)), err
}

func (gitRunner) makeBootstrapTree(ctx context.Context, repository, blob string) (string, error) {
	line := []byte("100644 blob " + blob + "\tarchitecture.yaml\n")
	output, err := runGit(ctx, line, "--git-dir", repository, "mktree")
	return strings.TrimSpace(string(output)), err
}

func (gitRunner) makeTree(ctx context.Context, repository string, entries []byte) (string, error) {
	output, err := runGit(ctx, entries, "--git-dir", repository, "mktree")
	return strings.TrimSpace(string(output)), err
}

func (gitRunner) makeBootstrapCommit(ctx context.Context, repository, tree string) (string, error) {
	output, err := runGit(ctx, []byte("Initialize Architecture\n"), "--git-dir", repository, "commit-tree", tree)
	return strings.TrimSpace(string(output)), err
}

func (gitRunner) makeSuccessorCommit(ctx context.Context, repository, tree, parent string) (string, error) {
	output, err := runGit(ctx, []byte("Update Architecture\n"), "--git-dir", repository, "commit-tree", tree, "-p", parent)
	return strings.TrimSpace(string(output)), err
}

func (gitRunner) makeStateCommit(ctx context.Context, repository, tree, parent string) (string, error) {
	output, err := runGit(ctx, []byte("Record Architecture change set\n"), "--git-dir", repository, "commit-tree", tree, "-p", parent)
	return strings.TrimSpace(string(output)), err
}

func (gitRunner) makeReviewCommit(ctx context.Context, repository, tree, parent string) (string, error) {
	output, err := runGit(ctx, []byte("Record Architecture review\n"), "--git-dir", repository, "commit-tree", tree, "-p", parent)
	return strings.TrimSpace(string(output)), err
}

func (gitRunner) createRef(ctx context.Context, repository, ref, object string) error {
	_, err := runGit(ctx, nil, "--git-dir", repository, "update-ref", ref, object, zeroObject)
	return err
}

func (gitRunner) updateRef(ctx context.Context, repository, ref, object, expected string) error {
	_, err := runGit(ctx, nil, "--git-dir", repository, "update-ref", ref, object, expected)
	return err
}

func (gitRunner) deleteRef(ctx context.Context, repository, ref, expected string) error {
	_, err := runGit(ctx, nil, "--git-dir", repository, "update-ref", "-d", ref, expected)
	return err
}

func (gitRunner) acceptChangeSet(ctx context.Context, repository, accepted, successor, activeRef, activeObject, appliedRef, appliedObject string) error {
	input := strings.Join([]string{
		"start",
		"update " + acceptedRef + " " + successor + " " + accepted,
		"delete " + activeRef + " " + activeObject,
		"create " + appliedRef + " " + appliedObject,
		"prepare",
		"commit",
		"",
	}, "\n")
	_, err := runGit(ctx, []byte(input), "--git-dir", repository, "update-ref", "--stdin")
	return err
}

func (gitRunner) createReview(ctx context.Context, repository, activeRef, activeObject, reviewRef, reviewObject string) error {
	input := strings.Join([]string{
		"start",
		"verify " + activeRef + " " + activeObject,
		"create " + reviewRef + " " + reviewObject,
		"prepare",
		"commit",
		"",
	}, "\n")
	_, err := runGit(ctx, []byte(input), "--git-dir", repository, "update-ref", "--stdin")
	return err
}

func (gitRunner) reconcileChangeSet(ctx context.Context, repository, accepted, activeRef, oldState, newState string) error {
	input := strings.Join([]string{"start", "verify " + acceptedRef + " " + accepted, "update " + activeRef + " " + newState + " " + oldState, "prepare", "commit", ""}, "\n")
	_, err := runGit(ctx, []byte(input), "--git-dir", repository, "update-ref", "--stdin")
	return err
}

func (gitRunner) diffTrees(ctx context.Context, repository, baseTree, candidateTree string) ([]byte, error) {
	return runGit(ctx, nil,
		"--git-dir", repository,
		"diff-tree", "--no-commit-id", "-r", "-p",
		"--no-ext-diff", "--no-textconv", "--no-color", "--text",
		"--src-prefix=a/", "--dst-prefix=b/",
		baseTree, candidateTree,
	)
}

func (gitRunner) treeEntries(ctx context.Context, repository, revision string) ([]treeEntry, error) {
	output, err := runGit(ctx, nil, "--git-dir", repository, "ls-tree", "-r", "-t", "-z", "--full-tree", revision)
	if err != nil {
		return nil, err
	}
	var entries []treeEntry
	for _, record := range bytes.Split(output, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		metadata, path, found := bytes.Cut(record, []byte{'\t'})
		if !found {
			return nil, errors.New("Git returned a malformed tree entry")
		}
		fields := strings.Fields(string(metadata))
		if len(fields) != 3 {
			return nil, errors.New("Git returned malformed tree metadata")
		}
		entries = append(entries, treeEntry{Mode: fields[0], Type: fields[1], Object: fields[2], Path: string(path)})
	}
	return entries, nil
}

func (gitRunner) directTreeEntries(ctx context.Context, repository, revision string) ([]treeEntry, error) {
	output, err := runGit(ctx, nil, "--git-dir", repository, "ls-tree", "-z", "--full-tree", revision)
	if err != nil {
		return nil, err
	}
	var entries []treeEntry
	for _, record := range bytes.Split(output, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		metadata, path, found := bytes.Cut(record, []byte{'\t'})
		if !found {
			return nil, errors.New("Git returned a malformed tree entry")
		}
		fields := strings.Fields(string(metadata))
		if len(fields) != 3 {
			return nil, errors.New("Git returned malformed tree metadata")
		}
		entries = append(entries, treeEntry{Mode: fields[0], Type: fields[1], Object: fields[2], Path: string(path)})
	}
	return entries, nil
}

func (gitRunner) objectType(ctx context.Context, repository, object string) (string, error) {
	output, err := runGit(ctx, nil, "--git-dir", repository, "cat-file", "-t", object)
	return strings.TrimSpace(string(output)), err
}

func (gitRunner) commitParent(ctx context.Context, repository, commit string) (string, error) {
	output, err := runGit(ctx, nil, "--git-dir", repository, "rev-list", "--parents", "-n", "1", commit)
	if err != nil {
		return "", err
	}
	fields := strings.Fields(string(output))
	if len(fields) != 2 || fields[0] != commit {
		return "", errors.New("state commit must have exactly one parent")
	}
	return fields[1], nil
}

func (gitRunner) commitTree(ctx context.Context, repository, commit string) (string, error) {
	output, err := runGit(ctx, nil, "--git-dir", repository, "rev-parse", commit+"^{tree}")
	return strings.TrimSpace(string(output)), err
}

func (gitRunner) readBlob(ctx context.Context, repository, object string) ([]byte, error) {
	return runGit(ctx, nil, "--git-dir", repository, "cat-file", "blob", object)
}

func runGit(ctx context.Context, input []byte, arguments ...string) ([]byte, error) {
	fixed := []string{
		"-c", "core.hooksPath=/dev/null",
		"-c", "commit.gpgSign=false",
		"-c", "tag.gpgSign=false",
		"-c", "color.ui=false",
		"-c", "core.pager=cat",
		"-c", "diff.external=",
	}
	command := exec.CommandContext(ctx, "git", append(fixed, arguments...)...)
	command.Env = controlledGitEnvironment()
	command.Stdin = bytes.NewReader(input)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return stdout.Bytes(), fmt.Errorf("git command failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

func controlledGitEnvironment() []string {
	environment := []string{
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=" + os.DevNull,
		"GIT_NO_REPLACE_OBJECTS=1",
		"GIT_TERMINAL_PROMPT=0",
		"GCM_INTERACTIVE=Never",
		"GIT_PAGER=cat",
		"GIT_EDITOR=true",
		"GIT_SEQUENCE_EDITOR=true",
		"GIT_AUTHOR_NAME=WorkBraid",
		"GIT_AUTHOR_EMAIL=architecture@workbraid.invalid",
		"GIT_COMMITTER_NAME=WorkBraid",
		"GIT_COMMITTER_EMAIL=architecture@workbraid.invalid",
		"LC_ALL=C",
	}
	for _, name := range []string{"PATH", "TMPDIR", "SystemRoot"} {
		if value := os.Getenv(name); value != "" {
			environment = append(environment, name+"="+value)
		}
	}
	return environment
}
