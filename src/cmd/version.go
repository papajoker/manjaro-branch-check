/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"mbc/cmd/alpm"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/spf13/cobra"
)

var (
	FlagDowngrade bool
	FlagGrep      string
)

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
	// Gérer les préfixes comme "2:" dans "2:1.0.0"
	v1 = strings.TrimPrefix(v1, "2:")
	v2 = strings.TrimPrefix(v2, "2:")

	ver1, err1 := semver.NewVersion(v1)
	ver2, err2 := semver.NewVersion(v2)

	if err1 == nil && err2 == nil {
		return ver1.Compare(ver2)
	}

	// Si les versions ne sont pas strictement sémantiques, faire une comparaison lexicographique
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

func version(config Config, cacheDir string, branches []string) {
	fmt.Printf("# %-53s %-49s / %s\n", "compare versions", Theme(branches[0])+branches[0]+Theme(""), Theme(branches[1])+branches[1]+Theme(""))
	var tmp [2]alpm.Packages
	tmpkeys := make(map[string]bool)
	tmp[0] = alpm.Load(cacheDir+"/"+branches[0]+"/sync", config.Repos)
	tmp[1] = alpm.Load(cacheDir+"/"+branches[1]+"/sync", config.Repos)

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

	fmt.Println("FlagGrep:", FlagGrep)
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

	for _, pkg := range keys {
		va := tmp[0][pkg].VERSION
		vb := tmp[1][pkg].VERSION

		//marker := ""
		highlightVb := vb
		highlightVa := va
		switch compareVersions(va, vb) {
		case -1:
			//marker = "" // vb >
			highlightVb = highlightDiff(va, vb, Theme(branches[1]))
			if FlagDowngrade {
				highlightVb = ""
			}
		case 1:
			//marker = "(downgrade)" // vb <
			highlightVa = highlightDiff(vb, va, Theme(branches[0]))
			highlightVa = padRightANSI(highlightVa, 40)
		}
		if highlightVb != "" {
			fmt.Printf("%-55s %-40s %s\n", pkg, highlightVa, highlightVb)
		}
	}
	fmt.Printf("# %d packages\n", len(keys))
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
		conf := ctx.Value("configVars").(Config)
		cacheDir := ctx.Value("cacheDir").(string)
		branches = FlagBranches.toSlice()
		version(conf, cacheDir, branches)
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
	versionCmd.Flags().BoolVarP(&FlagDowngrade, "downgrade", "", FlagDowngrade, "display only downgrade up")
	versionCmd.Flags().StringVarP(&FlagGrep, "grep", "", "", "name filter (regex)")
}
