package cmd

import (
	"fmt"
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
}
