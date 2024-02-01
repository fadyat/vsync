package main

import (
	"fmt"
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
	acommit(message string) error
	uncommittedChanges() []string
}

type git struct {
	path string
}

func (g *git) unreleasedChanges() []string {
	latestTag, part := g.latestTag(), "HEAD"
	if latestTag != "" {
		part = latestTag + "..HEAD"
	}

	cmd := exec.Command("git", "log", "--pretty=format:%s", part)
	cmd.Dir = g.path

	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	return strings.Split(string(out), "\n")
}

func (g *git) uncommittedChanges() []string {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = filepath.Join(g.path, "..")

	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	var (
		split   = strings.Split(string(out), "\n")
		changes = make([]string, 0, len(split))
	)
	for _, line := range split {
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

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git --version failed: %s", string(out))
	}

	return string(out), nil
}

func (g *git) newTag(tag string) error {
	cmd := exec.Command("git", "tag", tag)
	cmd.Dir = g.path

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git tag failed: %s", string(out))
	}

	return nil
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

func (g *git) add(path string) error {
	cmd := exec.Command("git", "add", path)
	cmd.Dir = filepath.Join(g.path, "..")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git add failed: %s", string(out))
	}

	return nil
}

func (g *git) acommit(message string) error {
	if err := g.add("."); err != nil {
		return err
	}

	cmd := exec.Command("git", "commit", "-am", message)
	cmd.Dir = filepath.Join(g.path, "..")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git commit failed: %s", string(out))
	}

	return nil
}
