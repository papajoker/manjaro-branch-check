/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"mbc/cmd/alpm"
	"path/filepath"
	"reflect"
	"regexp"
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
func (p *Branch) Set(branch string) {
	if len(branch) < 1 {
		return
	}
	first := string(branch[0])
	switch first {
	case "s":
		p.FlagStable = true
	case "t":
		p.FlagTesting = true
	case "u":
		p.FlagUnstable = true
	case "a":
		p.FlagArchlinux = true
	}

}

var (
	FlagStable    bool = false
	FlagTesting   bool = false
	FlagUnstable  bool = false
	FlagArchlinux bool = false
	branches           = []string{}
	FlagBranches       = Branch{}
	FlagDiffNew   bool
	FlagDiffRm    bool
)

type diffResult struct {
	first  string
	second string
}

func diff(diffs *[]diffResult, config Config, cacheDir string, branches []string, long bool) (int, int, int, [2]alpm.Packages) {
	excludes := []string{
		"pacman-mirrorlist", "reflector$", "linux-", "linux$", "r8168-lts", "tp_smapi",
		"vhba-module", "virtualbox-host-modules", "acpi_call",
		"archlinux-xdg-menu", "archlinux-wallpaper", "archlinux-themes-slim",
		"arch-release", "devtools$", "archinstall", "mirro-rs", "pkgstats", "xdm-archlinux", "grub-customizer",
		"nvidia$", "nvidia-lts", "nvidia-open$", "nvidia-open-lts",
	}

	var tmp [2]alpm.Packages
	var pkgs [2][]string

	tmp[0], _ = alpm.Load(filepath.Join(cacheDir, branches[0], "sync"), config.Repos, branches[0], long)
	tmp[1], _ = alpm.Load(filepath.Join(cacheDir, branches[1], "sync"), config.Repos, branches[1], long)

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
			if branches[1] == "archlinux" && startsWith(key, &excludes) {
				continue
			}
			pkgs[0] = append(pkgs[0], key+" *")
			l1 += 1
		}
	}
	sort.Strings(pkgs[0])
	if !long {
		tmp[0] = make(map[string]*alpm.Package)
		tmp[1] = make(map[string]*alpm.Package)
	}

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
	return max, l0, l1, tmp
}

func startsWith(input string, excludes *[]string) bool {
	for _, pattern := range *excludes {
		matched, err := regexp.MatchString("^"+pattern, input)
		if err == nil && matched {
			return true
		}
	}
	return false
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

		long := FlagDiffNew || FlagDiffRm

		var diffs []diffResult
		max, l0, l1, pkgs := diff(&diffs, conf, cacheDir, branches, long)
		fmt.Printf("%-"+strconv.Itoa(max+11)+"s / %s\n", Theme(branches[0])+branches[0]+Theme(""), Theme(branches[1])+branches[1]+Theme(""))
		for _, d := range diffs {
			fmt.Printf("%-"+strconv.Itoa(max)+"s / %s\n", d.first, d.second)
		}
		fmt.Println()
		fmt.Printf("# %-"+strconv.Itoa(max-2)+"d / %d\n", l0, l1)

		if FlagDiffNew && l1 > 0 {
			fmt.Println()
			fmt.Println("New in " + Theme(branches[1]) + branches[1] + Theme(""))
			for _, d := range diffs {
				if d.first == "" {
					pkg := pkgs[1][d.second]
					fmt.Println(Theme(branches[1])+pkg.NAME+Theme(""), "", pkg.Desc(48), ColorGray+pkg.URL+Theme(""))
				}
			}
		}
		if FlagDiffRm && l0 > 0 {
			fmt.Println()
			fmt.Println("Not in " + Theme(branches[0]) + branches[1] + Theme(""))
			for _, d := range diffs {
				if d.second == "" {
					pkg := pkgs[0][d.first]
					fmt.Println(Theme(branches[0])+pkg.NAME+Theme(""), "", pkg.Desc(48), ColorGray+pkg.URL+Theme(""))
				}
			}
		}
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

	diffCmd.Flags().BoolVarP(&FlagDiffNew, "new", "", FlagDiffNew, "new packages detail")
	diffCmd.Flags().BoolVarP(&FlagDiffRm, "rm", "", FlagDiffRm, "removed manjaro packages detail")
	diffCmd.MarkFlagsMutuallyExclusive("new", "rm")
	diffCmd.MarkFlagsMutuallyExclusive("archlinux", "rm") // or display manjaro exclusive packages but not deleted
}
