package cmd

import (
	"bufio"
	"fmt"
	"mbc/alpm"
	"mbc/theme"
	"mbc/tr"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

const INSTALL_FILE = "/desktopfs-pkgs.txt"

var (
	FlagAdd bool
	FlagRm  bool
)

func reafPkgFile(filename string) (pkgs alpm.Packages) {
	f, err := os.Open(filename)
	if err != nil {
		return
	}
	defer f.Close()

	fileScanner := bufio.NewScanner(f)
	//fileScanner.Split(bufio.ScanLines)
	pkgs = make(alpm.Packages)

	for fileScanner.Scan() {
		parts := strings.Fields(fileScanner.Text())
		//fmt.Println(parts)
		if len(parts) < 2 {
			continue
		}
		//fmt.Println(strings.TrimSpace(parts[0]))
		pkg := alpm.Package{NAME: parts[0], VERSION: parts[1]}
		pkgs[pkg.NAME] = &pkg
		//fmt.Println(pkg)
	}
	return pkgs
}

var installCmd = &cobra.Command{
	Use:    "install",
	Hidden: true,
	Short:  "compare installed packages",
	Long:   `compare ` + INSTALL_FILE + ` with pacman -Qq`,
	Run: func(cmd *cobra.Command, args []string) {
		pkgs := reafPkgFile(INSTALL_FILE)
		locals, err := alpm.LoadLocal()
		if err != nil {
			return
		}

		cRemoved, cNew, cExists := -1, -1, -1
		col1, col2 := 0, 0
		// uninstalled
		for _, pkg := range pkgs {
			if _, ok := locals[pkg.NAME]; !ok {
				cRemoved++
				if !FlagAdd || FlagRm {
					fmt.Println("-", pkg.NAME)
				}
			} else {
				l := len(pkg.NAME)
				if l > col1 {
					col1 = l
				}
				l = len(pkg.VERSION)
				if l > col2 {
					col2 = l
				}
			}
		}
		col1w := strconv.Itoa(col1 + 1)
		col2w := strconv.Itoa(col2 + 1)

		// exists
		if !FlagAdd && !FlagRm {
			fmt.Println()
			for _, pkg := range pkgs {
				if _, ok := locals[pkg.NAME]; !ok {
					continue
				}
				cExists++
				va := pkg.VERSION
				vb := locals[pkg.NAME].VERSION

				highlightVb := vb
				highlightVa := va
				switch alpm.AlpmPkgVerCmp(va, vb) {
				case -1:
					highlightVb = highlightDiff(va, vb, theme.Theme("s"))
					if FlagDowngrade {
						highlightVb = ""
					}
				case 1:
					highlightVa = highlightDiff(vb, va, theme.Theme("s"))
				}
				fmt.Printf("%-"+col1w+"s : %"+col2w+"s -> %s\n", pkg.NAME, highlightVa, highlightVb)
			}
		}

		// new packages
		if FlagAdd || (!FlagRm && !FlagAdd) {
			fmt.Println()
			for _, pkg := range locals {
				if _, ok := pkgs[pkg.NAME]; !ok {
					cNew++
					fmt.Println("+", pkg.NAME)
				}
			}
		}

		fmt.Println()
		if cRemoved > 0 {
			fmt.Printf("%4d %s\n", cRemoved, tr.T("uninstalled"))
		}
		if cExists > 0 {
			fmt.Printf("%4d %s\n", cExists, tr.T("updated"))
		}
		if cNew > 0 {
			fmt.Printf("%4d %s\n", cNew, tr.T("installed"))
		}
	},
}

func init() {
	if _, err := os.Stat(INSTALL_FILE); err != nil {
		return
	}
	rootCmd.AddCommand(installCmd)
	installCmd.Short = tr.T(installCmd.Short)
	installCmd.Flags().BoolVarP(&FlagAdd, "new", "", false, tr.T("new packages"))
	installCmd.Flags().BoolVarP(&FlagRm, "rm", "", false, tr.T("uninstalled"))
}
