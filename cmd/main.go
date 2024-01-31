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
	Tags           bool   `toml:"tags"`
	Changelog      bool   `toml:"changelog"`
	AutoCommit     bool   `toml:"autocommit"`
	CommitTemplate string `toml:"commit_template"`
	TagsPrefix     string `toml:"tags_prefix"`
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
			Tags:           false,
			Changelog:      true,
			AutoCommit:     false,
			CommitTemplate: "chore: release {{.Version}}",
			TagsPrefix:     "v",
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
	ErrGitRepoNotFound   = errors.New("git repository not found")
	ErrGitNotFound       = errors.New("git-cli not found")
	ErrAutoCommitMessage = errors.New("autocommit message can't be empty")
	ErrNothingToCommit   = errors.New("nothing to autocommit, generate tags or changelog")
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

			if cfg.Generator.AutoCommit && cfg.Generator.CommitTemplate == "" {
				return ErrAutoCommitMessage
			}

			if !cfg.Generator.Changelog {
				return nil
			}

			if cfg.Generator.AutoCommit && !(cfg.Generator.Tags || cfg.Generator.Changelog) {
				return ErrNothingToCommit
			}

			return verifyChangelog(flags.changelogPath)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfg.Generator.Tags {
				actions = append(actions, func() error {
					return newTag(flags.gitPath, cfg.Generator.TagsPrefix, cfg.Triggers)
				})
			}

			if cfg.Generator.Changelog {
				actions = append(actions, func() error {
					return updateChangelog(flags.gitPath, flags.changelogPath)
				})
			}

			if cfg.Generator.AutoCommit {
				actions = append(actions, func() error {
					return doCommit(flags.gitPath, cfg.Generator.CommitTemplate)
				})
			}

			for _, action := range actions {
				if err := action(); err != nil {
					return err
				}
			}

			return nil
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
