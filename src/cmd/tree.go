package cmd

import (
	"bufio"
	"fmt"
	"io"
	"mbc/cmd/alpm"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"os"
	"path/filepath"
)

var (
	GitBranch string
	Version   string
	BuildDate string
	GitID     string
	Project   string
)

func checkVersion(filename, version string) (bool, error) {
	inFile, err := os.Open(filename)
	if err != nil {
		return false, err
	}
	defer inFile.Close()

	scanner := bufio.NewScanner(inFile)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 5 {
			return false, nil
		}
		if line[1:] == version {
			return true, nil
		}
	}
	return false, nil
}

func setCompletion() {

	rootCmd.CompletionOptions.HiddenDefaultCmd = true

	var up = func(filename string, gen func(buf io.Writer) error) {
		ok, err := checkVersion(filename, Version)
		if err != nil {
			return
		}
		if ok {
			return
		}

		outFile, err := os.Create(filename)
		if err != nil {
			return
		}
		defer outFile.Close()

		outFile.Write([]byte("#" + Version + "\n"))
		gen(outFile)
	}

	// Version = "0.1.1" // for test with go run

	up(filepath.Join(os.Getenv("HOME"), ".config", "fish", "completions", "mbc.fish"), func(buf io.Writer) error {
		return rootCmd.GenFishCompletion(buf, true)
	})
	// ?? .local/share}/bash-completion}/completions
	up(filepath.Join(os.Getenv("HOME"), ".local", "share", "bash-completion", "completions", "mbc"), func(buf io.Writer) error {
		return rootCmd.GenBashCompletion(buf)
	})
	// ?? ~/.zsh/cache/mbc in manjaro-zsh-config
	up(filepath.Join(os.Getenv("HOME"), ".zsh", "cache", "mbc"), func(buf io.Writer) error {
		return rootCmd.GenZshCompletion(buf)
	})

}

func toHomeDir(abspath string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return abspath
	}
	return strings.Replace(abspath, homeDir, "~", 1)
}

func tree(config Config, cacheDir, confFilename string) {
	fmt.Println("# database:", toHomeDir(cacheDir))
	fmt.Println("# config:  ", toHomeDir(confFilename))
	fmt.Println()

	branches := append(config.Branches, "archlinux")
	for _, branch := range branches {
		fmt.Println(Theme(branch) + branch + Theme(""))
		for _, repo := range config.Repos {
			for range config.Arch {
				dirPath := filepath.Join(cacheDir, branch, "sync")
				if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
					fmt.Println("Error creating directory:", err)
					continue
				}
				filepath.Join(dirPath, repo+".db")

				fileInfo, _ := os.Stat(filepath.Join(dirPath, repo+".db"))
				tf := fileInfo.ModTime()
				d := time.Now().Sub(fileInfo.ModTime())
				days := ""
				if d.Hours() >= 48 {
					days = fmt.Sprintf("(%d days)", int(d.Hours()/24))
				}

				pkgs, _ := alpm.Load(dirPath, []string{repo}, branch, false)
				sep := Theme(branch) + "-" + Theme("")
				fmt.Printf("  %s %-10s %6d    (%s)  %s\n", sep, repo, len(pkgs), tf.Format("2006-01-02 15:04"), days)
			}
		}
		fmt.Println("")
	}
	urls := []string{}
	for _, url := range config.Urls {
		before, _, _ := strings.Cut(url, "$")
		urls = append(urls, before)
	}
	fmt.Println("# servers: ", urls)
	fmt.Printf("# %s Version: V%v %v %v %v\n", filepath.Base(os.Args[0]), Version, GitID, GitBranch, BuildDate)
}

// updateCmd represents the tree command
var treeCmd = &cobra.Command{
	Use:   "tree",
	Short: "list local repos",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		cacheDir := ctx.Value("cacheDir").(string)
		confFilename := ctx.Value("confFilename").(string)
		tree(ctx.Value("configVars").(Config), cacheDir, confFilename)
	},
}

func init() {
	rootCmd.AddCommand(treeCmd)
	setCompletion()
}
