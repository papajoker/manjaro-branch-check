package cmd

import (
	"bufio"
	"fmt"
	"mbc/theme"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/leonelquinteros/gotext"
	"github.com/spf13/cobra"
)

var (
	FlagQuiet  = false
	FlagInfo   bool
	FlagSearch bool
	FlagList   bool
)

func execPacman(cmd []string, branch string) {
	//fmt.Println()
	fmt.Println(theme.Theme(branch) + branch + theme.Theme(""))
	fmt.Println()

	run := exec.Command("/usr/bin/pacman", cmd...) //, "--debug")
	run.Stdout = os.Stdout
	err := run.Run()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println()
	fmt.Println(theme.Theme(branch) + branch + theme.Theme(""))
}

// pacmanCmd use the `pacman -S[i|s]`
var pacmanCmd = &cobra.Command{
	Use:   "pacman [packageName]",
	Short: "run pacman in branch",
	Long: `run a pacman command as:
pacman -S* --config '~/.cache/` + ApplicationID + `/BRANCH/pacman.conf'
Examples in stable branch.
pacman -Si: Info :
  -Is package_name
pacman -Ss: Search :
  -Ss text
pacman -Sl: List :
  -Ls
	`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		cachePath := ctx.Value(ctxCacheDir).(string)
		branch := FlagBranches.toSlice()[0]
		cachePath = filepath.Join(cachePath, branch, "pacman.conf")

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

		search := ""
		if len(args) > 0 {
			search = strings.TrimSpace(strings.ToLower(args[0]))
		}

		runargs := []string{}
		if FlagSearch {
			runargs = []string{"-Ss", search}
		}
		if FlagInfo {
			if len(search) > 0 {
				runargs = []string{"-Sii"}
				if FlagQuiet {
					runargs = []string{"-Si"}
				}
				for _, arg := range args {
					runargs = append(runargs, strings.TrimSpace(strings.ToLower(arg)))
				}
			} else {
				runargs = []string{"-Si"}
			}
		}
		if FlagList {
			runargs = []string{"-Sl"}
		}
		if FlagQuiet {
			runargs = append(runargs, "-q")
		}
		execPacman(append(runargs, "--config", cachePath), branch)
	},
}

func init() {
	if _, err := os.Stat("/usr/bin/pacman"); err == nil {
		rootCmd.AddCommand(pacmanCmd)
		pacmanCmd.Short = gotext.Get("run pacman in branch")

		pacmanCmd.Flags().BoolVarP(&FlagSearch, "Search", "S", false, gotext.Get("search"))
		pacmanCmd.Flags().BoolVarP(&FlagList, "List", "L", false, gotext.Get("list"))
		pacmanCmd.Flags().BoolVarP(&FlagInfo, "Info", "I", false, gotext.Get("info"))
		pacmanCmd.MarkFlagsOneRequired("Search", "List", "Info")
		pacmanCmd.MarkFlagsMutuallyExclusive("Search", "List", "Info")

		pacmanCmd.Flags().BoolVarP(&FlagBranches.FlagStable, "stable", "s", FlagBranches.FlagStable, "stable "+gotext.Get("branch"))
		pacmanCmd.Flags().BoolVarP(&FlagBranches.FlagTesting, "testing", "t", FlagBranches.FlagTesting, "testing "+gotext.Get("branch"))
		pacmanCmd.Flags().BoolVarP(&FlagBranches.FlagUnstable, "unstable", "u", FlagBranches.FlagUnstable, "unstable "+gotext.Get("branch"))
		pacmanCmd.Flags().BoolVarP(&FlagBranches.FlagArchlinux, "archlinux", "a", FlagBranches.FlagArchlinux, "archlinux "+gotext.Get("branch"))
		pacmanCmd.MarkFlagsOneRequired("stable", "testing", "unstable", "archlinux")
		pacmanCmd.MarkFlagsMutuallyExclusive("stable", "testing", "unstable", "archlinux")
		pacmanCmd.Flags().BoolVarP(&FlagQuiet, "quiet", "q", FlagQuiet, gotext.Get("show less information"))
	}
}
