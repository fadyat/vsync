package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type action struct {
	name string
	run  func() error
}

func writeToTopOfFile(filePath, content string) error {
	f, err := os.OpenFile(filepath.Clean(filePath), os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	defer func() {
		if err = f.Close(); err != nil {
			err = fmt.Errorf("failed to close file: %w", err)
		}
	}()

	stat, err := f.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	fileSize := stat.Size()
	buf := make([]byte, fileSize)
	_, err = f.Read(buf)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	_, err = f.WriteAt([]byte(content+"\n"), 0)
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	_, err = f.WriteAt(buf, int64(len(content)+1))
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	return nil
}

func markdownChangelog(tag, changelogPath string, changes []string) (err error) {
	var content strings.Builder
	content.WriteString(fmt.Sprintf("## [%s]\n\n", tag))

	for _, change := range changes {
		content.WriteString(fmt.Sprintf("- %s\n", change))
	}

	return writeToTopOfFile(changelogPath, content.String())
}

type gitWrapper struct {
	api             vcs
	changeLogWriter func(tag, changelogPath string, changes []string) error
}

func (g *gitWrapper) newTag(tagPrefix string, triggers *Triggers) error {
	uncommittedChanges := g.api.uncommittedChanges()
	if len(uncommittedChanges) > 0 {
		return ErrUncommittedChanges
	}

	changes := g.api.unreleasedChanges()
	latestTag := strings.TrimPrefix(g.api.latestTag(), tagPrefix)
	tag, err := g.bumpVersion(latestTag, changes, triggers)
	if err != nil {
		return err
	}

	tag = tagPrefix + tag
	return g.api.newTag(tag)
}

func (g *gitWrapper) bumpVersion(
	version string,
	commits []string,
	triggers *Triggers,
) (string, error) {
	// 0 - major, 1 - minor, 2 - patch, 3 - nothing to bump
	var priority = 3

	var havePrefix = func(s string, prefixes []string) bool {
		for _, prefix := range prefixes {
			if strings.HasPrefix(s, prefix) {
				return true
			}
		}

		return false
	}

	for _, commit := range commits {
		if havePrefix(commit, triggers.Major) {
			priority = min(priority, 0)
		}

		if havePrefix(commit, triggers.Minor) {
			priority = min(priority, 1)
		}

		if havePrefix(commit, triggers.Patch) {
			priority = min(priority, 2)
		}
	}

	return g.matchNextVersion(version, priority)
}

func (g *gitWrapper) matchNextVersion(version string, priority int) (string, error) {
	if priority == 3 {
		return "", ErrNothingToBump
	}

	// it's a new project with no tags
	if version == "" {
		version = "0.0.0"
	}

	parts := strings.Split(version, ".")
	if len(parts) < 3 {
		return "", fmt.Errorf("%w: %s", ErrInvalidSemVer, version)
	}

	part := parts[priority]
	num, err := strconv.Atoi(part)
	if err != nil {
		return "", err
	}

	parts[priority] = strconv.Itoa(num + 1)
	return strings.Join(parts, "."), nil
}

func (g *gitWrapper) updateChangelog(tagPrefix, changelogPath string, triggers *Triggers) error {
	uncommittedChanges := g.api.uncommittedChanges()
	if len(uncommittedChanges) > 0 {
		return ErrUncommittedChanges
	}

	changes := g.api.unreleasedChanges()
	latestTag := strings.TrimPrefix(g.api.latestTag(), tagPrefix)
	tag, err := g.bumpVersion(latestTag, changes, triggers)
	if err != nil {
		return err
	}

	tag = tagPrefix + tag
	return g.changeLogWriter(tag, changelogPath, changes)
}

func (g *gitWrapper) commit(autocommitMessage, changelogPath string) error {
	uncommittedChanges := g.api.uncommittedChanges()

	if len(uncommittedChanges) == 0 {
		return ErrNothingToCommit
	}

	if !strings.Contains(strings.Join(uncommittedChanges, "\n"), changelogPath) {
		return ErrChangeLogNotUpdated
	}

	if len(uncommittedChanges) > 1 {
		return ErrMultipleChanges
	}

	return g.api.acommit(autocommitMessage)
}
