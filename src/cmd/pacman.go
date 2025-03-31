package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
	fmt.Println(Theme(branch) + branch + Theme(""))
	fmt.Println()

	if FlagQuiet {
		cmd = append(cmd, "-q")
	}
	run := exec.Command("/usr/bin/pacman", cmd...) //, "--debug")
	run.Stdout = os.Stdout
	err := run.Run()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println()
	fmt.Println(Theme(branch) + branch + Theme(""))
}

// pacmanCmd use the `pacman -S[i|s]`
var pacmanCmd = &cobra.Command{
	Use:   "pacman [packageName]",
	Short: "run pacman in branch",
	Long: `run a pacman command as:
pacman -S* --config '~/.cache/manjaro-branch-check/BRANCH/pacman.conf'
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
		cachePath := ctx.Value("cacheDir").(string)
		branch := FlagBranches.toSlice()[0]
		cachePath = filepath.Join(cachePath, branch, "pacman.conf")
		fmt.Println("run ...", cachePath)

		search := ""
		if len(args) > 0 {
			search = strings.TrimSpace(strings.ToLower(args[0]))
		}

		args = []string{}
		if FlagSearch {
			args = []string{"-Ss", search, "--config", cachePath}
		}
		if FlagInfo {
			if len(search) > 0 {
				args = []string{"-Sii", search, "--config", cachePath}
			} else {
				args = []string{"-Si", "--config", cachePath}
			}
		}
		if FlagList {
			args = []string{"-Sl", "--config", cachePath}
		}
		execPacman(args, branch)
	},
}

func init() {
	if _, err := os.Stat("/usr/bin/pacman"); err == nil {
		rootCmd.AddCommand(pacmanCmd)

		pacmanCmd.Flags().BoolVarP(&FlagSearch, "Search", "S", false, "search")
		pacmanCmd.Flags().BoolVarP(&FlagList, "List", "L", false, "list")
		pacmanCmd.Flags().BoolVarP(&FlagInfo, "Info", "I", false, "info")
		pacmanCmd.MarkFlagsOneRequired("Search", "List", "Info")
		pacmanCmd.MarkFlagsMutuallyExclusive("Search", "List", "Info")

		pacmanCmd.Flags().BoolVarP(&FlagBranches.FlagStable, "stable", "s", FlagBranches.FlagStable, "stable branch")
		pacmanCmd.Flags().BoolVarP(&FlagBranches.FlagTesting, "testing", "t", FlagBranches.FlagTesting, "testing branch")
		pacmanCmd.Flags().BoolVarP(&FlagBranches.FlagUnstable, "unstable", "u", FlagBranches.FlagUnstable, "unstable branch")
		pacmanCmd.Flags().BoolVarP(&FlagBranches.FlagArchlinux, "archlinux", "a", FlagBranches.FlagArchlinux, "archlinux branch")
		pacmanCmd.MarkFlagsOneRequired("stable", "testing", "unstable", "archlinux")
		pacmanCmd.MarkFlagsMutuallyExclusive("stable", "testing", "unstable", "archlinux")
		pacmanCmd.Flags().BoolVarP(&FlagQuiet, "quiet", "q", FlagQuiet, "show less information")
	}
}
