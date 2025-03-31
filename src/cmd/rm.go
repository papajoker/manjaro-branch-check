package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rmCmd represents the rm command
var rmCmd = &cobra.Command{
	Use:   "rm",
	Short: "remove database in ~/.cache/",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		cacheDir := ctx.Value("cacheDir").(string)
		err := os.RemoveAll(cacheDir)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Directory", cacheDir, "removed successfully")
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
