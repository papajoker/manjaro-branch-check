package cmd

import (
	"fmt"
	"mbc/cmd/alpm"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/spf13/cobra"
)

var FlagPackager string = "manjaro"

func listP(config Config, cacheDir string, branch string) {

	reg, err := regexp.Compile(FlagPackager)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR! bad regex: %s\n", FlagGrep)
		os.Exit(2)
	}

	fmt.Println(Theme(branch) + branch + Theme(""))
	fmt.Println()

	items := make(map[string]int)
	pkgs := alpm.Load(filepath.Join(cacheDir, branch, "sync"), config.Repos)
	for _, pkg := range pkgs {
		if reg.MatchString(pkg.PACKAGER) {
			items[pkg.PACKAGER] += 1
		}
	}

	keys := make([]string, 0, len(items))
	for k := range items {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		//TODO mettre email en gris
		fmt.Printf("%-56s %5d\n", key, items[key])
	}
}

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list packagers",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		conf := ctx.Value("configVars").(Config)
		cacheDir := ctx.Value("cacheDir").(string)

		listP(conf, cacheDir, FlagBranches.toSlice()[0])
	},
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return fmt.Errorf("use only flags! %v too mutch", args)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVarP(&FlagBranches.FlagStable, "stable", "s", FlagBranches.FlagStable, "stable branch")
	listCmd.Flags().BoolVarP(&FlagBranches.FlagTesting, "testing", "t", FlagBranches.FlagTesting, "testing branch")
	listCmd.Flags().BoolVarP(&FlagBranches.FlagUnstable, "unstable", "u", FlagBranches.FlagUnstable, "unstable branch")
	listCmd.Flags().BoolVarP(&FlagBranches.FlagArchlinux, "archlinux", "a", FlagBranches.FlagArchlinux, "archlinux branch")
	listCmd.MarkFlagsOneRequired("stable", "testing", "unstable", "archlinux")
	listCmd.MarkFlagsMutuallyExclusive("stable", "testing", "unstable", "archlinux")
	listCmd.Flags().StringVarP(&FlagPackager, "grep", "", FlagPackager, "packager filter (regex)")
}
