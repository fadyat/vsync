package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// newTag creates a new tag in the git repository
// with semVer version based on the latest tag and
// commit messages.
func newTag(gitPath, tagPrefix string, triggers *Triggers) error {
	changes := getUnreleasedChanges(gitPath, triggers)
	tag := bumpVersion(latestTag(gitPath), changes)

	fmt.Println("New tag:", tag)
	return nil
}

// latestTag returns the latest tag in the git repository,
// or empty string if there are no tags.
func latestTag(gitPath string) string {
	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
	cmd.Dir = gitPath

	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(out))
}

// getUnreleasedChanges returns a list of commits, which
// aren't included in any tag yet.
func getUnreleasedChanges(gitPath string, triggers *Triggers) []string {
	return nil
}

// bumpVersion returns a new version based on the latest tag
// and commits.
func bumpVersion(version string, commits []string) string {
	return ""
}

func updateChangelog(gitPath, changelogPath string) error {
	changes := getUnreleasedChanges(gitPath, nil)
	tag := bumpVersion(latestTag(gitPath), changes)

	fmt.Println("Updating changelog...")
	fmt.Println("New version:", tag)
	fmt.Println("Changes:", changes)

	return nil
}

func doCommit(gitPath, msgTemplate string) error {
	// verify that are only changelog changes
	// otherwise, abort commit

	return nil
}
