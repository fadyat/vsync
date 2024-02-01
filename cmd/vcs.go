package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type vcs interface {
	verify() error
	version() (string, error)
	newTag(tag string) error
	latestTag() string
	unreleasedChanges() []string
	commit(message string) error
	uncommittedChanges() []string
}

type git struct {
	path string
}

func (g *git) unreleasedChanges() []string {
	cmd := exec.Command("git", "log", "--pretty=format:%s")
	cmd.Dir = g.path

	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	var changes []string
	for _, line := range strings.Split(string(out), "\n") {
		changes = append(changes, line)
	}

	return changes
}

func (g *git) uncommittedChanges() []string {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = filepath.Join(g.path, "..")

	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	var changes []string
	for _, line := range strings.Split(string(out), "\n") {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		changes = append(changes, parts[1])
	}

	return changes
}

func (g *git) verify() error {
	if _, err := os.Stat(g.path); err != nil {
		return ErrGitRepoNotFound
	}

	if _, err := exec.LookPath("git"); err != nil {
		return ErrGitNotFound
	}

	return nil
}

func (g *git) version() (string, error) {
	cmd := exec.Command("git", "--version")
	cmd.Dir = g.path

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(out), nil
}

func (g *git) newTag(tag string) error {
	cmd := exec.Command("git", "tag", tag)
	cmd.Dir = g.path

	return cmd.Run()
}

func (g *git) latestTag() string {
	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
	cmd.Dir = g.path

	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(out))
}

func (g *git) commit(message string) error {
	cmd := exec.Command("git", "commit", "-am", message)
	cmd.Dir = g.path

	return cmd.Run()
}
