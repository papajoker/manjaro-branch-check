package cmd

import (
	"context"
	"fmt"
	"mbc/cmd/alpm"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/spf13/cobra"
	"google.golang.org/api/option"
)

var FlagIA bool

func geminiInformation(pkg, repo string) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR!", err)
		return
	}
	defer client.Close()

	req := `
		Use this lang ` + os.Getenv("LANG") + ` for the response.
		Response not in markdown but in simple text for console.

		Informations on a pacman package in Manjaro linux.
		This package is in repository : ` + repo + `
		Informations on utility of this manjaro package : ` + pkg + `
		if this package have an application, can you add a descriptif of this app ? 10 lines maximum ?
	`

	model := client.GenerativeModel("gemini-2.0-flash")
	resp, err := model.GenerateContent(ctx, genai.Text(req))
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR!", err)
		return
	}
	fmt.Println()
	fmt.Println()
	fmt.Println(resp.Candidates[0].Content.Parts[0])
	/*fmt.Println()
	response := fmt.Sprint(resp.Candidates[0].Content.Parts[0])
	response = strings.ReplaceAll(response, "```", "")
	response, _ = strings.CutPrefix(response, "text\n")
	fmt.Println(response)
	*/
}

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:        "info pakageName",
	Short:      "A brief description of your package",
	Long:       ``,
	Args:       cobra.MinimumNArgs(1),
	ArgAliases: []string{"package"},
	Run: func(cmd *cobra.Command, args []string) {
		flags := cmd.Flags()
		flags.Bool("test", false, "test flag") // ? arg optionel

		pkgName := strings.TrimSpace(strings.ToLower(args[0]))
		if pkgName == "" {
			fmt.Fprintln(os.Stderr, "Empty package name")
			os.Exit(2)
		}

		ctx := cmd.Context()
		conf := ctx.Value("configVars").(Config)
		cacheDir := ctx.Value("cacheDir").(string)

		branches := append(conf.Branches, "archlinux")
		repo := ""
		for _, branch := range branches {
			fmt.Println(Theme(branch) + branch + Theme(""))
			pkgs := alpm.Load(filepath.Join(cacheDir, branch, "sync"), conf.Repos)
			pkg := pkgs[pkgName]
			if pkg != nil {
				fmt.Println(pkg)
				repo = pkg.REPO
			} else {
				fmt.Println(" ?")
			}
		}
		if FlagIA {
			geminiInformation(pkgName, repo)
		}
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
	if len(os.Getenv("GEMINI_API_KEY")) > 1 {
		infoCmd.Flags().BoolVarP(&FlagIA, "ia", "", FlagIA, "add General Info by Gemini")
	}
}
