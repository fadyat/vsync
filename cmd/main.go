package main

import (
	"errors"
	"github.com/BurntSushi/toml"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/spf13/cobra"
	"log"
	"log/slog"
	"os"
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
	TagsPrefix        string `toml:"tags_prefix"`
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
			Tags:              false,
			Changelog:         true,
			AutoCommit:        false,
			TagsPrefix:        "v",
			AutoCommitMessage: "chore[VSync]: changelog updated",
		},
	}
}

func overwriteConfig(c *config, path string) {
	if err := cleanenv.ReadConfig(path, c); err != nil {
		slog.Debug("config file not found, using default values")
	}
}

func overwriteGit(gitCli *git, path string) {
	gitCli.path = path
}

type vsyncFlags struct {
	configPath    string
	changelogPath string
	gitPath       string
}

var (
	ErrGitRepoNotFound     = errors.New("git repository not found")
	ErrGitNotFound         = errors.New("git-cli not found")
	ErrAutoCommitMessage   = errors.New("autocommit message can't be empty")
	ErrNothingToCommit     = errors.New("nothing to autocommit, generate tags or changelog")
	ErrChangeLogNotUpdated = errors.New("changelog not updated")
	ErrMultipleChanges     = errors.New("multiple changes, please commit manually")
	ErrUncommittedChanges  = errors.New("uncommitted changes, commit or stash them")
	ErrNothingToBump       = errors.New("nothing to bump based on commits")
	ErrInvalidSemVer       = errors.New("invalid semVer format of latest tag")
)

func main() {
	log.SetFlags(0)

	var (
		cfg     = defaultConfig()
		flags   = &vsyncFlags{}
		actions []*action
		gitCli  = &git{}
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
			overwriteGit(gitCli, flags.gitPath)

			if err = gitCli.verify(); err != nil {
				return err
			}

			if cfg.Generator.AutoCommit && cfg.Generator.AutoCommitMessage == "" {
				return ErrAutoCommitMessage
			}

			if cfg.Generator.AutoCommit && !(cfg.Generator.Tags || cfg.Generator.Changelog) {
				return ErrNothingToCommit
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			gw := &gitWrapper{
				api:             gitCli,
				changeLogWriter: markdownChangelog,
			}

			if cfg.Generator.Changelog {
				actions = append(actions, &action{
					name: "update changelog",
					run: func() error {
						return gw.updateChangelog(
							cfg.Generator.TagsPrefix, flags.changelogPath, cfg.Triggers,
						)
					},
				})
			}

			if cfg.Generator.AutoCommit {
				actions = append(actions, &action{
					name: "autocommit",
					run: func() error {
						return gw.commit(cfg.Generator.AutoCommitMessage, flags.changelogPath)
					},
				})
			}

			if cfg.Generator.Tags {
				actions = append(actions, &action{
					name: "new tag",
					run: func() error {
						return gw.newTag(
							cfg.Generator.TagsPrefix, cfg.Triggers,
						)
					},
				})
			}

			for _, a := range actions {
				log.Printf("Running %q action", a.name)
				if err := a.run(); err != nil {
					log.Printf("Failed to run %q action: %v", a.name, err)
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
	vsync.PersistentFlags().StringVarP(&cfg.Generator.TagsPrefix, "tags-prefix", "p", cfg.Generator.TagsPrefix, "tags prefix")
	vsync.PersistentFlags().StringVarP(&cfg.Generator.AutoCommitMessage, "autocommit-message", "m", cfg.Generator.AutoCommitMessage, "autocommit message")

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
