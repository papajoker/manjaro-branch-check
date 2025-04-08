/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"mbc/cmd/alpm"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/spf13/cobra"
)

var (
	FlagDowngrade bool
	FlagGrep      string
)

type versionResult struct {
	name    string
	vfirst  string
	vsecond string
}

// Regex pour supprimer les codes ANSI
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// Supprime les codes ANSI pour calculer la vraie largeur de la chaîne
func realLength(s string) int {
	return len(ansiRegex.ReplaceAllString(s, ""))
}

// Formate la chaîne en gardant la largeur correcte même avec ANSI
func padRightANSI(s string, width int) string {
	diff := width - realLength(s)
	if diff > 0 {
		return s + strings.Repeat(" ", diff)
	}
	return s
}

func compareVersions(v1, v2 string) int {
	if v1 == v2 {
		return 0
	}

	var epochRm = func(v string) string {
		if b, a, found := strings.Cut(v, ":"); found {
			v = a
		} else {
			v = b
		}
		return v
	}
	ve1 := epochRm(v1)
	ve2 := epochRm(v2)

	ver1, err1 := semver.NewVersion(ve1)
	ver2, err2 := semver.NewVersion(ve2)

	if err1 == nil && err2 == nil {
		return ver1.Compare(ver2)
	}

	// not good version for semver ?

	var convert = func(str string) string {
		// FIX this: -8 < -10 ?NO but -0008 < -0010 ok
		if b, a, found := strings.Cut(str, "-"); found {
			if rel, err := strconv.Atoi(a); err == nil {
				return b + "-" + fmt.Sprintf("%04d", rel)
			}
		}
		return str
	}
	v1 = convert(v1)
	v2 = convert(v2)
	//FIX this: 43.1-1             43.1+r37+gea6718427-1
	v1 = strings.ReplaceAll(v1, "+", "?")
	v2 = strings.ReplaceAll(v2, "+", "?")

	if v1 > v2 {
		return 1
	} else if v1 < v2 {
		return -1
	}
	return 0
}

func highlightDiff(va, vb string, color string) string {
	oldParts := strings.Split(va, "")
	newParts := strings.Split(vb, "")

	var highlighted strings.Builder

	for i := range newParts {
		if i >= len(oldParts) || oldParts[i] != newParts[i] {
			highlighted.WriteString(color + newParts[i]) // Partie différente en rouge
		} else {
			highlighted.WriteString(newParts[i])
		}
	}

	return highlighted.String() + Theme("")
}

func version(versions *[]versionResult, config Config, cacheDir string, branches []string) (int, int, string) {
	var tmp [2]alpm.Packages
	tmpkeys := make(map[string]bool)
	tmp[0], _ = alpm.Load(filepath.Join(cacheDir, branches[0], "sync"), config.Repos, branches[0], false)
	tmp[1], _ = alpm.Load(filepath.Join(cacheDir, branches[1], "sync"), config.Repos, branches[1], false)

	for key := range tmp[0] {
		if _, exists := tmp[1][key]; exists {
			if tmp[0][key].VERSION != tmp[1][key].VERSION {
				tmpkeys[key] = true
			}
		}
	}
	for key := range tmp[1] {
		if _, exists := tmp[0][key]; exists {
			if tmp[0][key].VERSION != tmp[1][key].VERSION {
				tmpkeys[key] = true
			}
		}
	}

	keys := make([]string, 0, len(tmpkeys))
	FlagGrep = strings.ToLower(FlagGrep)
	reg, err := regexp.Compile(FlagGrep)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR! bad regex: %s\n", FlagGrep)
		os.Exit(2)
	}

	for k := range tmpkeys {
		if FlagGrep != "" {
			if reg.MatchString(k) {
				keys = append(keys, k)
			}
		} else {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	clear(tmpkeys)

	col1, col2 := 12, 12

	for _, pkg := range keys {
		va := tmp[0][pkg].VERSION
		vb := tmp[1][pkg].VERSION

		highlightVb := vb
		highlightVa := va
		switch compareVersions(va, vb) {
		case -1:
			// vb >
			highlightVb = highlightDiff(va, vb, Theme(branches[1]))
			if FlagDowngrade {
				highlightVb = ""
			}
		case 1:
			//vb <
			highlightVa = highlightDiff(vb, va, Theme(branches[0]))
		}
		if highlightVb != "" {
			*versions = append(*versions, versionResult{pkg, highlightVa, highlightVb})
			if len(pkg) > col1 {
				col1 = len(pkg)
			}
			if len(va) > col2 {
				col2 = len(va)
			}
		}
	}
	return col1 + 1, col2 + 1, FlagGrep
}

// diffCmd represents the diff command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Compare versions over branches",
	Long: `Example, compare "stable" vs "unstable":
version  -su --grep '^linux(..|...)$'
# package                          / stable                     / unstable
linux612                             6.12.19-1                    6.12.20-2
linux613                             6.13.7-1                     6.13.8-2
linux614                             6.14.0rc7-1                  6.14.0-1
linux66                              6.6.83-1                     6.6.84-1
...
`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		conf := ctx.Value(ctxConfigVars).(Config)
		cacheDir := ctx.Value(ctxCacheDir).(string)
		branches = FlagBranches.toSlice()

		if updateDateFromFile() >= AutoUpdate {
			updateCmd.Run(cmd, []string{""})
			fmt.Println()
		}

		var versions []versionResult
		col1, col2, grepflag := version(&versions, conf, cacheDir, branches)

		fmt.Printf("# %-"+strconv.Itoa(col1-2)+"s %-"+strconv.Itoa(col2+9)+"s / %s\n", "compare versions", Theme(branches[0])+branches[0]+Theme(""), Theme(branches[1])+branches[1]+Theme(""))
		for _, v := range versions {
			v.vfirst = padRightANSI(v.vfirst, col2)
			fmt.Printf("%-"+strconv.Itoa(col1)+"s %-"+strconv.Itoa(col2)+"s %s\n", v.name, v.vfirst, v.vsecond)
		}
		fmt.Println()
		fmt.Printf("# %d packages\n", len(versions))
		if grepflag != "" {
			fmt.Println("# filter:", grepflag)
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
	rootCmd.AddCommand(versionCmd)
	versionCmd.Flags().BoolVarP(&FlagBranches.FlagStable, "stable", "s", FlagBranches.FlagStable, "stable branch")
	versionCmd.Flags().BoolVarP(&FlagBranches.FlagTesting, "testing", "t", FlagBranches.FlagTesting, "testing branch")
	versionCmd.Flags().BoolVarP(&FlagBranches.FlagUnstable, "unstable", "u", FlagBranches.FlagUnstable, "unstable branch")
	versionCmd.Flags().BoolVarP(&FlagBranches.FlagArchlinux, "archlinux", "a", FlagBranches.FlagArchlinux, "archlinux branch")
	versionCmd.Flags().BoolVarP(&FlagDowngrade, "overgrade", "", FlagDowngrade, "display only downgrade up")
	versionCmd.Flags().StringVarP(&FlagGrep, "grep", "", "", "name filter (regex)")
}
