package cmd

import (
	"context"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

//go:embed config.yaml
var embedFS embed.FS

type Config struct {
	Branches []string `yaml:"branches"`
	Arch     []string `yaml:"arch"`
	Repos    []string `yaml:"repos"`
	Urls     []string `yaml:"urls"`
}

type AppConfig struct {
	Ctx    context.Context
	Config Config
}

var AppState = &AppConfig{}

func loadConfig(confFilename string) (*Config, error) {
	file, err := os.Open(confFilename)
	if err != nil {
		// for test: rm ~/.config/manjaro-branch-check.yaml
		conf, _ := embedFS.ReadFile("config.yaml")
		f, err := os.Create(confFilename)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		f.WriteString(string(conf))
		fmt.Printf("# Create file Config : %s\n", confFilename)
	}
	file.Close()

	var config Config
	file, err = os.Open(confFilename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func cacheIsValid(config Config, cacheDir string) error {
	branches := append(config.Branches, "archlinux")
	for _, branch := range branches {
		for _, repo := range config.Repos {
			for range config.Arch {
				dirPath := filepath.Join(cacheDir, branch, "sync")
				filePath := filepath.Join(dirPath, repo+".db")
				if _, err := os.Stat(filePath); err != nil {
					return fmt.Errorf("local database corrupted! run command `update`")
				}
			}
		}
	}
	return nil
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "mbc",
	Short: "Play with Manjaro Repos",
	CompletionOptions: cobra.CompletionOptions{
		HiddenDefaultCmd: true, // hides cmd
	},
	Long: `Manjaro Multi Branch packages Navigator
use four branches with same config (archlinux, unstable, testing, stable)

Which packages are new to a branch? (diff)
Which packages disappear? (diff)
What are the version differences between branches? (info, version)
`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		confFilename := filepath.Join(os.Getenv("HOME"), ".config", "manjaro-branch-check.yaml")
		conf, err := loadConfig(confFilename)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error loading yaml configuration", confFilename)
			return err
		}
		ctx := context.Background()
		ctx = context.WithValue(ctx, "configVars", *conf)
		ctx = context.WithValue(ctx, "cacheDir", filepath.Join(os.Getenv("HOME"), ".cache", "manjaro-branch-check"))
		ctx = context.WithValue(ctx, "confFilename", confFilename)
		cmd.SetContext(ctx)
		if !(strings.HasPrefix(cmd.Use, "help") || strings.HasPrefix(cmd.Use, "update")) {
			return cacheIsValid(*conf, ctx.Value("cacheDir").(string))
		}
		return err
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
		updateCmd.Run(cmd, []string{"silent"})
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Println("ERROR!", err)
		os.Exit(1)
	}
}

func init() {
}
