package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"mbc/ai"
	"mbc/alpm"
	"mbc/theme"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/leonelquinteros/gotext"
	"github.com/spf13/cobra"
)

type branchNaneFlagType struct {
	value  string
	valids []string
}

var (
	FlagAI         bool
	FlagInstalled  bool
	FlagDetailInfo branchNaneFlagType
)

func (e *branchNaneFlagType) String() string {
	return e.value
}

func (e *branchNaneFlagType) Set(v string) error {
	if len(v) == 1 {
		return e.SetOne(v)
	}
	if slices.Contains(e.valids, v) {
		e.value = v
		return nil
	}
	return errors.New(gotext.Get("must be one of") + ` "` + strings.Join(e.valids, `", "`) + `"`)
}

func (e *branchNaneFlagType) SetOne(branch string) error {
	firsts := make([]string, len(e.valids))
	for v := range e.valids {
		firsts[v] = string(e.valids[v][0])
	}
	if i := slices.Index(firsts, string(branch[0])); i != -1 {
		e.value = e.valids[i]
		return nil
	}
	return errors.New(gotext.Get("must be one of") + ` "` + strings.Join(e.valids, `", "`) + `"`)
}

func (e *branchNaneFlagType) Type() string {
	return "branch"
}

func isInstalled(pkg string, test bool) string {
	if !test {
		return ""
	}
	matches, _ := filepath.Glob("/var/lib/pacman/local/" + pkg + "-*/desc")
	if len(matches) > 0 {
		return "*"
	}
	return ""
}

// cobra arg is regex or not ?
func isPlainString(s string) bool {
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			continue
		case r >= '0' && r <= '9':
			continue
		case r == '-' || r == '.' || r == '_':
			continue
		default:
			return false
		}
	}
	return len(s) > 0
}

// search packages in all branchies (if regex test all entries)
func getKeys(pkgs map[string]alpm.Packages, search string) (keys []string) {
	search = strings.ToLower(search)
	reg, err := regexp.Compile(strings.TrimSpace(search))
	if err != nil {
		return
	}
	for _, branch := range pkgs {
		if len(branch) < 1 {
			continue
		}
		if isPlainString(search) {
			if branch[search] != nil {
				if !slices.Contains(keys, search) {
					return append(keys, search)
				}
			}
			return keys
		}
		for k := range branch {
			if reg.MatchString(k) {
				if slices.Contains(keys, k) {
					continue
				}
				keys = append(keys, k)
			}
		}
	}
	return
}

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:   "info pakageName(s)",
	Short: "a brief description of your package",
	Long: `Compare versions for one or more packages.
Returns version differences across branches, if differences exist.

ex:
	info pacman grub
	info 'linux\d{2}$' 'linux\d..$' '^linux\d.*-rt$'
	info "#kernel"  	# search all kernels
	info pacman --detail t		# run at end pacman -Si in branch Testing
	echo -e "pacman grub" | mbc info -
	`,
	Args:       cobra.MinimumNArgs(1),
	ArgAliases: []string{"package"},
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		conf := ctx.Value(ctxConfigVars).(Config)
		cacheDir := ctx.Value(ctxCacheDir).(string)

		if updateDateFromFile() >= AutoUpdate {
			updateCmd.Run(cmd, []string{""})
			fmt.Println()
		}

		branches := append(conf.Branches, "archlinux")

		if len(args) > 0 && args[0] == "-" {
			args = []string{}
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Split(bufio.ScanWords)
			for scanner.Scan() {
				word := strings.TrimSpace(scanner.Text())
				if len(word) > 1 {
					args = append(args, word)
				}
			}
		}

		var warnings []string
		pkgs := make(map[string]alpm.Packages, len(branches))
		for _, branch := range branches {
			p, warns := alpm.Load(filepath.Join(cacheDir, branch, "sync"), conf.Repos, branch, false)
			pkgs[branch] = p
			if warns != nil {
				warnings = append(warnings, warns...)
			}
		}

		for i, arg := range args {
			pkgName := strings.TrimSpace(strings.ToLower(arg))
			if pkgName == "" {
				fmt.Fprintln(os.Stderr, "Empty package name")
				os.Exit(2)
			}
			if len(pkgName) > 6 && pkgName[0:7] == "#kernel" {
				pkgName = `^linux\d{2,3}(-rt)?$`
			}
			repo := ""

			for _, pkgName = range getKeys(pkgs, pkgName) {
				fmt.Printf("\n%s %s", pkgName, isInstalled(pkgName, FlagInstalled))
				oldVersion := ""
				for _, branch := range branches {
					//fmt.Println("  ", Theme(branch)+branch+Theme(""))
					pkg := pkgs[branch][pkgName]
					if pkg != nil {
						if pkg.VERSION == oldVersion {
							continue
						}

						d := time.Since(pkg.BUILDDATE)
						days := ""
						if d.Hours() >= 24 {
							days = fmt.Sprintf("(%d %s)", int(d.Hours()/24), gotext.Get("days"))
						}
						ver := pkg.VERSION
						if oldVersion != "" {
							ver = highlightDiff(oldVersion, pkg.VERSION, theme.Theme(branch))
						}
						oldVersion = pkg.VERSION
						fmt.Println()
						fmt.Printf("   Version:  %-11s %s\n", padRightANSI(theme.Theme(branch)+branch+theme.Theme(""), 11), ver)
						fmt.Printf("   Date:     %-11s %s\t%s\n", " ", pkg.BUILDDATE.Format("06-01-02 15:04"), days)

						repo = pkg.REPO
					} /*else {
						fmt.Printf("\n   -         %s\n", Theme(branch)+branch+Theme(""))
					}*/

				}

				if len(FlagDetailInfo.value) > 0 {
					fmt.Println()
					FlagBranches.Set(string(FlagDetailInfo.value))
					FlagInfo = true
					var args = []string{pkgName}
					pacmanCmd.Run(cmd, args)
				}
				if FlagAI {
					if a := ai.MakeAi(conf.API); a != nil {
						s := a.Ask(ai.GetAskPackage(pkgName, repo))
						if s != "" {
							fmt.Println()
							fmt.Println(s)
						}
					}
				}

				if i > 264 { //TODO remove ?
					fmt.Fprintf(os.Stderr, "WARNING!\n  %s\n", gotext.Get("Too many packages, stop here"))
					break
				}
			}
		}
		if len(warnings) > 0 {
			fmt.Fprintln(os.Stderr)
			fmt.Fprintf(os.Stderr, "WARNING!\n  %s\n", strings.Join(warnings, "  "))
		}
	},
}

func init() {
	setLocale()
	rootCmd.AddCommand(infoCmd)
	infoCmd.Short = gotext.Get("a brief description of your package")

	if len(os.Getenv("GEMINI_API_KEY")) > 1 {
		infoCmd.Flags().BoolVarP(&FlagAI, "ai", "", FlagAI, gotext.Get("add General Info by Gemini"))
	}
	if _, err := os.Stat("/var/lib/pacman/local"); err == nil {
		infoCmd.Flags().BoolVarP(&FlagInstalled, "installed", "i", FlagInstalled, gotext.Get("package installed ?"))
		if _, err := os.Stat("/usr/bin/pacman"); err == nil {
			infoCmd.Flags().Var(&FlagDetailInfo, "detail", gotext.Get("run pacman -Si in `branch`"))
		}
	}

	conf, _ := loadConfig(Config{}.configFile())
	FlagDetailInfo = branchNaneFlagType{
		value:  "",
		valids: append(conf.Branches, "archlinux"),
	}

}
