package cmd

import (
	"fmt"
	"mbc/cmd/alpm"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"

	"github.com/spf13/cobra"
)

var FlagPackager string = "manjaro"

type listResult struct {
	name  string
	count int
}

func listP(packagers *[]listResult, config Config, cacheDir string, branch string) int {

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

	max := 10
	for _, key := range keys {
		*packagers = append(*packagers, listResult{key, items[key]})
		if len(key) > max {
			max = len(key)
		}
	}
	return max
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

		var packagers []listResult
		max := listP(&packagers, conf, cacheDir, FlagBranches.toSlice()[0]) + 1

		for _, packager := range packagers {
			//TODO gray color for email
			fmt.Printf("%-"+strconv.Itoa(max)+"s %5d\n", packager.name, packager.count)
		}
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
