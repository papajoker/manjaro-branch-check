package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// rmCmd represents the rm command
var rmCmd = &cobra.Command{
	Use:   "rm",
	Short: "remove database in ~/.cache/",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		cacheDir := ctx.Value("cacheDir").(string)
		confFilename := ctx.Value("confFilename").(string)
		err := os.RemoveAll(cacheDir)
		//var err error = nil
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Directory", cacheDir, "removed successfully")
		}

		execPath, err := os.Executable()
		if err != nil {
			return
		}

		var response string
		fmt.Print("\nDelete this script and configuration ? (y/N) : ")
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

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// rmCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// rmCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
