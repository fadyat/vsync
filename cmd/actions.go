package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func markdownChangelog(tag, changelogPath string, changes []string) (err error) {
	f, err := os.OpenFile(changelogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open changelog file: %w", err)
	}

	defer func() {
		if err = f.Close(); err != nil {
			err = fmt.Errorf("failed to close changelog file: %w", err)
		}
	}()

	tagHeader := fmt.Sprintf("## %s\n\n", tag)
	if _, err = f.WriteString(tagHeader); err != nil {
		return fmt.Errorf("failed to write to changelog file: %w", err)
	}

	for _, change := range changes {
		changePointer := fmt.Sprintf("- %s\n", change)
		if _, err = f.WriteString(changePointer); err != nil {
			return fmt.Errorf("failed to write to changelog file: %w", err)
		}
	}

	return nil
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

	fmt.Println(uncommittedChanges)
	return fmt.Errorf("aboba")

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

		if havePrefix(commit, triggers.Patch) && priority < 2 {
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

	return g.api.commit(autocommitMessage)
}
