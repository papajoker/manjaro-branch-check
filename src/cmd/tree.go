package cmd

import (
	"bufio"
	"fmt"
	"io"
	"mbc/cmd/alpm"
	"strings"

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
}

func tree(config Config, cacheDir, confFilename string) {
	fmt.Println("# database:", cacheDir)
	fmt.Println("# config:  ", confFilename)
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
				t := fileInfo.ModTime()
				pkgs, _ := alpm.Load(dirPath, []string{repo}, branch, false)
				sep := Theme(branch) + "-" + Theme("")
				fmt.Printf("  %s %-10s %6d    (%s)\n", sep, repo, len(pkgs), t.Format("2006-01-02 15:04"))
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
