/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"mbc/cmd/alpm"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

type Branch struct {
	FlagStable    bool
	FlagTesting   bool
	FlagUnstable  bool
	FlagArchlinux bool
}

func (p Branch) count() int {
	countTrue := func(values []bool) int {
		count := 0
		for _, v := range values {
			if v {
				count++
			}
		}
		return count
	}
	return countTrue([]bool{p.FlagStable, p.FlagTesting, p.FlagUnstable, p.FlagArchlinux})
}

func (p Branch) toSlice() []string {
	ts := reflect.TypeOf(p)
	vs := reflect.ValueOf(p)
	result := []string{}
	for i := 0; i < vs.NumField(); i++ {
		v := vs.Field(i).Bool()
		if v {
			n := strings.ToLower(ts.Field(i).Name[4:])
			result = append(result, n)
		}
	}
	return result
}

var (
	FlagStable    bool = false
	FlagTesting   bool = false
	FlagUnstable  bool = false
	FlagArchlinux bool = false
	branches           = []string{}
	FlagBranches       = Branch{}
)

type diffResult struct {
	first  string
	second string
}

func diff(diffs *[]diffResult, config Config, cacheDir string, branches []string) (int, int, int) {
	var tmp [2]alpm.Packages
	var pkgs [2][]string

	tmp[0] = alpm.Load(filepath.Join(cacheDir, branches[0], "sync"), config.Repos)
	tmp[1] = alpm.Load(filepath.Join(cacheDir, branches[1], "sync"), config.Repos)

	for key := range tmp[0] {
		if _, exists := tmp[1][key]; !exists {
			pkgs[0] = append(pkgs[0], key)
		}
	}
	sort.Strings(pkgs[0])
	l0 := len(pkgs[0])
	l1 := 0

	for key := range tmp[1] {
		if _, exists := tmp[0][key]; !exists {
			pkgs[0] = append(pkgs[0], key+" *")
			l1 += 1
		}
	}
	sort.Strings(pkgs[0])
	tmp[0] = make(map[string]*alpm.Package)
	tmp[1] = make(map[string]*alpm.Package)

	max := 12
	for _, name := range pkgs[0] {
		if strings.HasSuffix(name, "*") {
			*diffs = append(*diffs, diffResult{"", strings.TrimSuffix(name, " *")})
		} else {
			*diffs = append(*diffs, diffResult{name, ""})
			if len(name) > max {
				max = len(name)
			}
		}
	}
	return max, l0, l1
}

// diffCmd represents the diff command
var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "branch packages differences",
	Long: `Differentiate the branches, display the packages unique to the branch.
Example, compare "stable" to "archlinux":
diff  -sa
stable                                             / archlinux
# 48                                               /   10
crossover                                          / 
crossover-extras                                   / 
                                                   / lib32-directx-headers
lib32-gamescope-plus                               /
...
`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		conf := ctx.Value("configVars").(Config)
		cacheDir := ctx.Value("cacheDir").(string)
		branches = FlagBranches.toSlice()

		var diffs []diffResult
		max, l0, l1 := diff(&diffs, conf, cacheDir, branches)
		fmt.Printf("%-"+strconv.Itoa(max+11)+"s / %s\n", Theme(branches[0])+branches[0]+Theme(""), Theme(branches[1])+branches[1]+Theme(""))
		for _, d := range diffs {
			fmt.Printf("%-"+strconv.Itoa(max)+"s / %s\n", d.first, d.second)
		}
		fmt.Println()
		fmt.Printf("# %-"+strconv.Itoa(max-2)+"d / %d\n", l0, l1)
	},
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return fmt.Errorf("use only flags! %v too mutch", args)
		}
		result := FlagBranches.count()
		if result != 2 {
			return fmt.Errorf("invalid branches specified: %s", "not 2")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(diffCmd)
	diffCmd.Flags().BoolVarP(&FlagBranches.FlagStable, "stable", "s", FlagBranches.FlagStable, "stable branch")
	diffCmd.Flags().BoolVarP(&FlagBranches.FlagTesting, "testing", "t", FlagBranches.FlagTesting, "testing branch")
	diffCmd.Flags().BoolVarP(&FlagBranches.FlagUnstable, "unstable", "u", FlagBranches.FlagUnstable, "unstable branch")
	diffCmd.Flags().BoolVarP(&FlagBranches.FlagArchlinux, "archlinux", "a", FlagBranches.FlagArchlinux, "archlinux branch")
}
