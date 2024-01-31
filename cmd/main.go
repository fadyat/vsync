package main

import (
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/spf13/cobra"
	"log"
	"log/slog"
	"os"
	"os/exec"
)

const (
	Version = "dev"
)

type Triggers struct {
	Major []string `toml:"major"`
	Minor []string `toml:"minor"`
	Patch []string `toml:"patch"`
}

type Generator struct {
	Tags              bool   `toml:"tags"`
	Changelog         bool   `toml:"changelog"`
	AutoCommit        bool   `toml:"autocommit"`
	AutoCommitMessage string `toml:"autocommit_message"`
}

type config struct {
	Triggers  *Triggers  `toml:"triggers"`
	Generator *Generator `toml:"generator"`
}

func defaultConfig() *config {
	return &config{
		Triggers: &Triggers{
			Major: []string{"break", "major"},
			Minor: []string{"feat", "feature", "minor"},
			Patch: []string{"fix", "perf", "ref", "docs", "style", "chore", "tests"},
		},
		Generator: &Generator{
			Tags:              true,
			Changelog:         true,
			AutoCommit:        false,
			AutoCommitMessage: "chore: release {{.Version}}",
		},
	}
}

func overwriteConfig(c *config, path string) {
	if err := cleanenv.ReadConfig(path, c); err != nil {
		slog.Debug("config file not found, using default values")
	}
}

type vsyncFlags struct {
	configPath    string
	changelogPath string
	gitPath       string
}

var (
	ErrGitRepoNotFound       = errors.New("git repository not found")
	ErrGitNotFound           = errors.New("git-cli not found")
	ErrChangeLogTagsOptions  = errors.New("changelog option can't be used without tags option")
	ErrAutoCommitTagsOptions = errors.New("autocommit option can't be used without tags option")
	ErrorAutoCommitMessage   = errors.New("autocommit message can't be empty")
)

func verifyGit(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("%w: %s", ErrGitRepoNotFound, path)
	}

	gitCmd := exec.Command("git", "--version")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", ErrGitNotFound, err)
	}

	return nil
}

func verifyChangelog(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("changelog file not found: %w", err)
	}

	return nil
}

func main() {
	log.SetFlags(0)

	var (
		cfg     = defaultConfig()
		flags   = &vsyncFlags{}
		actions []func() error
	)

	var vsync = &cobra.Command{
		Use:   "vsync",
		Short: "VSync is automatic semantic versioning tool for git.",
		Long: `VSync can help you to manage your project's version and changelog automatically.

It will increase your project's version automatically based on your git commit messages.
Tool is highly configurable and keywords can be changed to fit your needs.

All you need is to follow https://www.conventionalcommits.org/en/ standard for your commit messages.
VSync is inspired to automate https://semver.org/ and https://keepachangelog.com/en/ standards.
`,
		PreRunE: func(cmd *cobra.Command, args []string) (err error) {
			overwriteConfig(cfg, flags.configPath)

			if err = verifyGit(flags.gitPath); err != nil {
				return err
			}

			if !cfg.Generator.Tags && cfg.Generator.Changelog {
				return ErrChangeLogTagsOptions
			}

			if !cfg.Generator.Tags && cfg.Generator.AutoCommit {
				return ErrAutoCommitTagsOptions
			}

			if cfg.Generator.AutoCommit && cfg.Generator.AutoCommitMessage == "" {
				return ErrorAutoCommitMessage
			}

			if !cfg.Generator.Changelog {
				return nil
			}

			return verifyChangelog(flags.changelogPath)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfg.Generator.Tags {
				actions = append(actions, func() error {
					return newTag(flags.gitPath, cfg.Triggers)
				})
			}

			if cfg.Generator.Tags && cfg.Generator.Changelog {
				actions = append(actions, func() error {
					return updateChangelog(flags.gitPath, flags.changelogPath)
				})
			}

			if cfg.Generator.Tags && cfg.Generator.AutoCommit {
				actions = append(actions, func() error {
					return commit(flags.gitPath, cfg.Generator.AutoCommitMessage)
				})
			}

			return run(cfg, flags.gitPath, actions...)
		},
		SilenceUsage: true,
	}

	vsync.PersistentFlags().StringVarP(&flags.configPath, "config-path", "", "vsync.toml", "config file path")
	vsync.PersistentFlags().StringVarP(&flags.changelogPath, "changelog-path", "", "CHANGELOG.md", "changelog file path")
	vsync.PersistentFlags().StringVarP(&flags.gitPath, "git", "", ".git", "git repository path")

	vsync.PersistentFlags().BoolVarP(&cfg.Generator.Tags, "tags", "t", cfg.Generator.Tags, "generate tags based on commit messages")
	vsync.PersistentFlags().BoolVarP(&cfg.Generator.Changelog, "changelog", "c", cfg.Generator.Changelog, "generate changelog based on commit messages")
	vsync.PersistentFlags().BoolVarP(&cfg.Generator.AutoCommit, "autocommit", "a", cfg.Generator.AutoCommit, "autocommit changes")

	vsync.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the version number of VSync",
		Long:  `All software has versions. This is VSync's`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println("VSync version:", Version)
		},
	})

	vsync.AddCommand(&cobra.Command{
		Use:   "config",
		Short: "Print the configuration of VSync",
		Long:  `Print the configuration of VSync with passed flags`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return toml.NewEncoder(os.Stdout).Encode(cfg)
		},
	})

	_ = vsync.Execute()
}
