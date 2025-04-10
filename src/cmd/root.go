package cmd

import (
	"context"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/leonelquinteros/gotext"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

//go:embed config.yaml
var embedFS embed.FS

type ctxkey int

const (
	AutoUpdate    int = 2
	ApplicationID     = "manjaro-branch-check"
)
const (
	ctxConfigVars ctxkey = iota
	ctxCacheDir
	ctxConfFilename
)

type Config struct {
	Branches []string `yaml:"branches"`
	Arch     []string `yaml:"arch"`
	Repos    []string `yaml:"repos"`
	Urls     []string `yaml:"urls"`
}

func (c Config) cache() string {
	return filepath.Join(os.Getenv("HOME"), ".cache", ApplicationID)
}

func (c Config) configFile() string {
	return filepath.Join(os.Getenv("HOME"), ".config", ApplicationID+".yaml")
}

type AppConfig struct {
	Ctx    context.Context
	Config Config
}

var AppState = &AppConfig{}

func loadConfig(confFilename string) (*Config, error) {
	file, err := os.Open(confFilename)
	if err != nil {
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
					os.Create(filePath)
					//return fmt.Errorf("local database corrupted! run command `update`")
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
		confFilename := Config{}.configFile()
		conf, err := loadConfig(confFilename)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error loading yaml configuration", confFilename)
			return err
		}
		ctx := context.Background()
		ctx = context.WithValue(ctx, ctxConfigVars, *conf)
		ctx = context.WithValue(ctx, ctxCacheDir, conf.cache())
		ctx = context.WithValue(ctx, ctxConfFilename, confFilename)
		cmd.SetContext(ctx)
		if !(strings.HasPrefix(cmd.Use, "help") || strings.HasPrefix(cmd.Use, "update")) {
			return cacheIsValid(*conf, ctx.Value(ctxCacheDir).(string))
		}
		return err
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Println("ERROR!", err)
		os.Exit(1)
	}
}

func setLocale() {
	exePath, err := os.Executable()
	if err != nil {
		panic(err)
	}
	dir := filepath.Join(filepath.Dir(exePath), "locale")
	gotext.Configure(dir, "fr", "default")
}

func init() {
	setLocale()
}
