package cmd

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"mbc/alpm"
	"mbc/theme"
	"mbc/tr"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	GitBranch string
	Version   string
	BuildDate string
	GitID     string
	Project   string
)

func getLTSFamilies() (families map[string]bool) {

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

	families = make(map[string]bool)

	resp, err := http.Get("https://www.kernel.org/feeds/kdist.xml")
	if err != nil {
		return families
	}
	defer resp.Body.Close()

	var rss RSS
	if err := xml.NewDecoder(resp.Body).Decode(&rss); err != nil {
		return families
	}

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
	return
}

func filterLTSKernels(input []string, ltsFamilies map[string]bool) ([]string, error) {
	if len(ltsFamilies) < 1 {
		return nil, fmt.Errorf("LTS not found")
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

	if runtime.GOOS == "windows" {
		return
	}

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

	h, _ := os.UserHomeDir()
	up(filepath.Join(h, ".config", "fish", "completions", "mbc.fish"), func(buf io.Writer) error {
		return rootCmd.GenFishCompletion(buf, true)
	})
	// ?? .local/share}/bash-completion}/completions
	up(filepath.Join(h, ".local", "share", "bash-completion", "completions", "mbc"), func(buf io.Writer) error {
		return rootCmd.GenBashCompletion(buf)
	})
	// ?? ~/.zsh/cache/mbc in manjaro-zsh-config
	up(filepath.Join(h, ".zsh", "cache", "mbc"), func(buf io.Writer) error {
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

	var ltsFamilies map[string]bool
	ltsChan := make(chan struct{})
	go func() {
		defer close(ltsChan)
		ltsFamilies = getLTSFamilies()
	}()

	kernels := []string{}
	branches := append(config.Branches, "archlinux")

	padw := 0
	for _, b := range config.Repos {
		long := len(b)
		if long > padw {
			padw = long
		}
	}

	for _, branch := range branches {
		fmt.Println(theme.Theme(branch) + branch + theme.Theme(""))
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
				if fileInfo.Size() < 1 {
					continue
				}
				tf := fileInfo.ModTime()
				d := time.Since(fileInfo.ModTime())
				days := ""
				if d.Hours() >= 48 {
					days = fmt.Sprintf("(%d %s)", int(d.Hours()/24), tr.T("days"))
				}

				pkgs, _ := alpm.Load(dirPath, []string{repo}, branch, false)
				sep := theme.Theme(branch) + "-" + theme.Theme("")
				fmt.Printf("  %s %-*s   %6d    (%s)  %s\n", sep, padw, repo, len(pkgs), tf.Format("2006-01-02 15:04"), days)
				if repo == "core" && len(pkgs) > 1 {
					// search kernels
					keys = getKeys(map[string]alpm.Packages{"core": pkgs}, `^linux\d{2,3}(-rt)?$`)
				}
			}
		}
		if len(keys) > 0 {
			sortKernels(keys)
			fmt.Printf("    %s%s%s\n", theme.ColorGray, strings.Join(keys, " "), theme.ColorNone)
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

	<-ltsChan

	if len(kernels) > 0 {
		lts, err := filterLTSKernels(kernels, ltsFamilies)
		if err == nil {
			fmt.Printf("# %-16s: %s\t%s\n",
				"LTS", strings.Join(lts, ", "),
				theme.ColorGray+"\t("+tr.T("by")+" kernel.org)"+theme.ColorNone)
		}
	}
	fmt.Printf("# %-16s: %s\n", tr.T("mirrors"), strings.Join(urls, ", "))
	fmt.Printf("# %-16s: %s\n", tr.T("database"), toHomeDir(cacheDir))
	fmt.Printf("# %-16s: %s\n", tr.T("config"), toHomeDir(confFilename))
	if alpm.LocalDBExists() {
		if pkgs, err := alpm.LoadLocal(); err == nil {
			fmt.Printf("# %-16s: %d %s\n", tr.T("installed"), len(pkgs), tr.T("packages"))
		}
	}
	fmt.Printf("# %s: V%v %v %v %v\n", filepath.Base(os.Args[0]), Version, GitID, GitBranch, BuildDate)
}

var treeCmd = &cobra.Command{
	Use:   "tree",
	Short: "infos on local repos",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		cacheDir := ctx.Value(ctxCacheDir).(string)
		confFilename := ctx.Value(ctxConfFilename).(string)
		tree(ctx.Value(ctxConfigVars).(Config), cacheDir, confFilename)
	},
}

func init() {
	treeCmd.Short = tr.T(treeCmd.Short)
	rootCmd.AddCommand(treeCmd)
	setCompletion()
}
