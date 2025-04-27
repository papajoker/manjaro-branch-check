package cmd

import (
	"fmt"
	"mbc/tr"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// rmCmd represents the rm command
var rmCmd = &cobra.Command{
	Use:   "rm",
	Short: "remove database in" + " ~/.cache/",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		cacheDir := ctx.Value(ctxCacheDir).(string)
		confFilename := ctx.Value(ctxConfFilename).(string)
		err := os.RemoveAll(cacheDir)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(tr.T("Directory"), cacheDir, tr.T("removed successfully"))
		}

		execPath, err := os.Executable()
		if err != nil {
			return
		}

		var response string
		fmt.Print("\n" + tr.T("Delete this script and configuration") + " ? (y/N) : ")
		if _, err = fmt.Scanln(&response); err != nil {
			return
		}
		response = strings.ToLower(response)
		switch response {
		case "y", "o":
			os.Remove(confFilename)
			os.Remove(execPath)

			completionFile := filepath.Join(os.Getenv("HOME"), ".config", "fish", "completions", "mbc.fish")
			os.Remove(completionFile)
			completionFile = filepath.Join(os.Getenv("HOME"), ".local", "share", "bash-completion", "completions", "mbc")
			os.Remove(completionFile)
			completionFile = filepath.Join(os.Getenv("HOME"), ".zsh", "cache", "mbc")
			os.Remove(completionFile)
		}
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
	rmCmd.Short = tr.T("remove database in") + " ~/.cache/"
}
