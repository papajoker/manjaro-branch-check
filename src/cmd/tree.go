package cmd

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"mbc/cmd/alpm"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"os"
	"path/filepath"
)

var (
	GitBranch string
	Version   string
	BuildDate string
	GitID     string
	Project   string
)

func getLTSFamilies() (map[string]bool, error) {

	type (
		Item struct {
			Title string `xml:"title"`
		}
		Channel struct {
			Items []Item `xml:"item"`
		}
		RSS struct {
			Channel Channel `xml:"channel"`
		}
	)

	var extractFamily = func(version string) string {
		// version "5.10.210" to "510"
		parts := strings.SplitN(version, ".", 3)
		if len(parts) >= 2 {
			return parts[0] + parts[1]
		}
		return ""
	}

	resp, err := http.Get("https://www.kernel.org/feeds/kdist.xml")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var rss RSS
	if err := xml.NewDecoder(resp.Body).Decode(&rss); err != nil {
		return nil, err
	}

	families := make(map[string]bool)

	for _, item := range rss.Channel.Items {
		if strings.Contains(item.Title, "longterm") {
			fields := strings.Fields(item.Title)
			if len(fields) > 1 {
				version := fields[0]
				family := extractFamily(version)
				if family != "" {
					families[family] = true
				}
			}
		}
	}
	return families, nil
}

func filterLTSKernels(input []string) ([]string, error) {
	ltsFamilies, err := getLTSFamilies()
	if err != nil {
		return nil, err
	}

	var result []string
	var family string
	for _, kernel := range input {
		if strings.Contains(kernel, "-rt") {
			continue
		}
		family = strings.TrimPrefix(kernel, "linux")
		if ltsFamilies[family] {
			result = append(result, family)
		}
	}
	return result, nil
}

func checkVersion(filename, version string) (bool, error) {
	inFile, err := os.Open(filename)
	if err != nil {
		return false, err
	}
	defer inFile.Close()

	scanner := bufio.NewScanner(inFile)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 5 {
			return false, nil
		}
		if line[1:] == version {
			return true, nil
		}
	}
	return false, nil
}

func sortKernels(kernels []string) {
	type kernelVersion struct {
		original string
		major    int
		minor    int
		isRT     bool
	}

	var parseKernelVersion = func(s string) kernelVersion {
		re := regexp.MustCompile(`linux(\d)(\d+)(-rt)?`)
		match := re.FindStringSubmatch(s)
		if len(match) >= 3 {
			major, _ := strconv.Atoi(match[1])
			minor, _ := strconv.Atoi(match[2])
			isRT := match[3] == "-rt"
			return kernelVersion{
				original: s,
				major:    major,
				minor:    minor,
				isRT:     isRT,
			}
		}
		return kernelVersion{original: s}
	}

	versions := make([]kernelVersion, 0, len(kernels))
	for _, k := range kernels {
		versions = append(versions, parseKernelVersion(k))
	}

	sort.Slice(versions, func(i, j int) bool {
		a, b := versions[i], versions[j]
		if a.major != b.major {
			return a.major < b.major
		}
		if a.minor != b.minor {
			return a.minor < b.minor
		}
		return !a.isRT && b.isRT
	})

	for i, v := range versions {
		kernels[i] = v.original
	}
}

func setCompletion() {

	rootCmd.CompletionOptions.HiddenDefaultCmd = true

	var up = func(filename string, gen func(buf io.Writer) error) {
		ok, err := checkVersion(filename, Version)
		if err != nil {
			return
		}
		if ok {
			return
		}

		outFile, err := os.Create(filename)
		if err != nil {
			return
		}
		defer outFile.Close()

		outFile.Write([]byte("#" + Version + "\n"))
		gen(outFile)
	}

	// Version = "0.1.1" // for test with go run

	up(filepath.Join(os.Getenv("HOME"), ".config", "fish", "completions", "mbc.fish"), func(buf io.Writer) error {
		return rootCmd.GenFishCompletion(buf, true)
	})
	// ?? .local/share}/bash-completion}/completions
	up(filepath.Join(os.Getenv("HOME"), ".local", "share", "bash-completion", "completions", "mbc"), func(buf io.Writer) error {
		return rootCmd.GenBashCompletion(buf)
	})
	// ?? ~/.zsh/cache/mbc in manjaro-zsh-config
	up(filepath.Join(os.Getenv("HOME"), ".zsh", "cache", "mbc"), func(buf io.Writer) error {
		return rootCmd.GenZshCompletion(buf)
	})

}

func toHomeDir(abspath string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return abspath
	}
	return strings.Replace(abspath, homeDir, "~", 1)
}

func tree(config Config, cacheDir, confFilename string) {
	fmt.Println("# database:", toHomeDir(cacheDir))
	fmt.Println("# config:  ", toHomeDir(confFilename))
	fmt.Println()

	kernels := []string{}
	branches := append(config.Branches, "archlinux")
	for _, branch := range branches {
		fmt.Println(Theme(branch) + branch + Theme(""))
		keys := []string{}
		for _, repo := range config.Repos {
			for range config.Arch {
				dirPath := filepath.Join(cacheDir, branch, "sync")
				if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
					fmt.Println("Error creating directory:", err)
					continue
				}
				filepath.Join(dirPath, repo+".db")

				fileInfo, _ := os.Stat(filepath.Join(dirPath, repo+".db"))
				tf := fileInfo.ModTime()
				d := time.Since(fileInfo.ModTime())
				days := ""
				if d.Hours() >= 48 {
					days = fmt.Sprintf("(%d days)", int(d.Hours()/24))
				}

				pkgs, _ := alpm.Load(dirPath, []string{repo}, branch, false)
				sep := Theme(branch) + "-" + Theme("")
				fmt.Printf("  %s %-10s %6d    (%s)  %s\n", sep, repo, len(pkgs), tf.Format("2006-01-02 15:04"), days)

				if repo == "core" && len(pkgs) > 1 {
					// search kernels
					keys = getKeys(map[string]alpm.Packages{"core": pkgs}, `^linux\d{2,3}(-rt)?$`)
				}
			}
		}
		if len(keys) > 0 {
			sortKernels(keys)
			fmt.Printf("    %s%s%s\n", ColorGray, strings.Join(keys, " "), ColorNone)
			if branch == "stable" {
				kernels = append(kernels, keys...)
			}
		}
		fmt.Println("")
	}
	urls := []string{}
	for _, url := range config.Urls {
		before, _, _ := strings.Cut(url, "$")
		urls = append(urls, before)
	}

	if len(kernels) > 0 {
		lts, err := filterLTSKernels(kernels)
		if err == nil {
			fmt.Println("# LTS: ", strings.Join(lts, ", "), ColorGray+"\t(by kernel.org)"+ColorNone)
		}
	}
	fmt.Println("# servers: ", urls)
	fmt.Printf("# %s Version: V%v %v %v %v\n", filepath.Base(os.Args[0]), Version, GitID, GitBranch, BuildDate)
}

// updateCmd represents the tree command
var treeCmd = &cobra.Command{
	Use:   "tree",
	Short: "list local repos",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		cacheDir := ctx.Value("cacheDir").(string)
		confFilename := ctx.Value("confFilename").(string)
		tree(ctx.Value("configVars").(Config), cacheDir, confFilename)
	},
}

func init() {
	rootCmd.AddCommand(treeCmd)
	setCompletion()
}
