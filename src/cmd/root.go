package cmd

import (
	"context"
	"embed"
	"fmt"
	"os"
	"path/filepath"

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

func loadConfig() (*Config, error) {
	//TODO si fichier non trouvé, use embed
	path := filepath.Join(os.Getenv("HOME"), ".config", "manjaro-branch-check.yaml")
	file, err := os.Open(path)
	if err != nil {
		// for test: rm ~/.config/manjaro-branch-check.yaml
		conf, _ := embedFS.ReadFile("config.yaml")
		//f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
		f, err := os.Create(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		f.WriteString(string(conf))
		fmt.Printf("Create file Config : %s\n", path)
		//return nil, err
	}
	file.Close()

	var config Config
	file, err = os.Open(path)
	if err != nil {
		fmt.Printf("Config file bad ?? %s\n", path)
		return nil, err
	}
	defer file.Close()
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "mbc",
	Short: "Play with Manjaro Repos",
	CompletionOptions: cobra.CompletionOptions{
		HiddenDefaultCmd: true, // hides cmd
		// DisableDefaultCmd: true, // removes cmd
	},
	Long: `Manjaro Multi Branch packages Navigator
use four branches with same config (archlinux, unstable, testing, stable)

Which packages are new to a branch? (diff)
Which packages disappear? (diff)
What are the version differences between branches? (info, version)
`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		conf, err := loadConfig()
		if err != nil {
			fmt.Println("Error loading configuration:", err)
			return
		}
		ctx := context.Background()
		ctx = context.WithValue(ctx, "configVars", *conf)
		ctx = context.WithValue(ctx, "cacheDir", filepath.Join(os.Getenv("HOME"), ".cache", "manjaro-branch-check"))
		cmd.SetContext(ctx)
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
