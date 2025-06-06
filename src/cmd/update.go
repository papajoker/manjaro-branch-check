package cmd

import (
	"fmt"
	"io"
	"mbc/theme"
	"mbc/tr"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

func _getDateFile() string {
	return filepath.Join(Config{}.cache(), "date")
}

func updateDateFromFile() int {
	data, err := os.ReadFile(_getDateFile())
	if err != nil {
		return 0
	}
	if date, err := time.Parse(time.RFC3339, string(data)); err == nil {
		d := time.Since(date)
		return int(d.Hours() / 24)
	}
	return 0
}

func updateDateToFile() error {
	file, err := os.Create(_getDateFile())
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(time.Now().Format(time.RFC3339))
	return err
}

func shouldDownload(url, filePath string) (bool, error) {
	resp, err := http.Head(url)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf(tr.T("failed to access remote file: %s"), resp.Status)
	}

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}

	remoteSize := resp.Header.Get("Content-Length")
	if remoteSize != "" {
		var size int64
		fmt.Sscanf(remoteSize, "%d", &size)
		if fileInfo.Size() != size {
			return true, nil
		}
	}

	remoteMod := resp.Header.Get("Last-Modified")
	if remoteMod != "" {
		remoteTime, err := time.Parse(http.TimeFormat, remoteMod)
		if err == nil && fileInfo.ModTime().Before(remoteTime) {
			return true, nil
		}
	}

	return false, nil
}

func downloadFile(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s", tr.T("download failed"), resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func createConfigPacman(directory string, repos []string) error {
	if err := os.MkdirAll(directory, os.ModePerm); err != nil {
		fmt.Printf("%s: %v", tr.T("Error creating directory"), err)
		return err
	}
	f, err := os.Create(directory + "/pacman.conf")
	if err != nil {
		return err
	}
	defer f.Close()
	content := "[options]\nDBPath = " + directory + "\nColor\n\n"
	for _, repo := range repos {
		content = content + "[" + repo + "]\n"
	}
	f.WriteString(content)
	return nil
}

func update(config Config, silent bool) {

	cacheBase := config.cache()

	var wg sync.WaitGroup

	var out io.Writer = os.Stdout
	if silent {
		out = io.Discard
	}

	var branch string
	for _, url := range config.Urls {
		if strings.Contains(url, "$branch") {
			for _, branch = range config.Branches {
				err := createConfigPacman(filepath.Join(cacheBase, branch), config.Repos)
				if err != nil {
					panic(err)
				}
				for _, repo := range config.Repos {
					for _, arch := range config.Arch {
						finalURL := strings.ReplaceAll(url, "$branch", branch)
						finalURL = strings.ReplaceAll(finalURL, "$repo", repo)
						finalURL = strings.ReplaceAll(finalURL, "$arch", arch)
						finald, _ := strings.CutPrefix(finalURL, "https://")
						fmt.Fprintln(out, finald, theme.Theme(branch)+"..."+theme.Theme(""))

						dirPath := filepath.Join(cacheBase, branch, "sync")
						if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
							fmt.Fprintf(out, "%s: %v\n", tr.T("Error creating directory"), err)
							continue
						}

						filePath := filepath.Join(dirPath, filepath.Base(finalURL))

						shouldDownload, err := shouldDownload(finalURL, filePath)
						if err != nil {
							fmt.Fprintf(out, "%s: %v\n", tr.T("Error checking file"), err)
							continue
						}

						if shouldDownload {
							wg.Add(1)
							go func(url, path, branch string) {
								defer wg.Done()
								if err := downloadFile(url, path); err != nil {
									fmt.Fprintf(out, "%s: %v\n", tr.T("Download error"), err)
								} else {
									path := strings.ReplaceAll(path, "/"+branch+"/", "/"+theme.Theme(branch)+branch+theme.Theme("")+"/")
									fmt.Fprintf(out, "%s%s:%s %s\n", theme.Theme(branch), tr.T("Downloaded"), theme.Theme(""), path)
								}
							}(finalURL, filePath, branch)
						}
					}
				}
			}
		} else {
			branch = "archlinux"
			err := createConfigPacman(filepath.Join(cacheBase, branch), config.Repos)
			if err != nil {
				panic(err)
			}
			for _, repo := range config.Repos {
				for _, arch := range config.Arch {
					//finalURL := strings.ReplaceAll(url, "$branch", branch)
					finalURL := strings.ReplaceAll(url, "$repo", repo)
					finalURL = strings.ReplaceAll(finalURL, "$arch", arch)
					finald, _ := strings.CutPrefix(finalURL, "https://")
					finald, _ = strings.CutPrefix(finald, "http://")
					fmt.Fprintln(out, finald, theme.Theme(branch)+"..."+theme.Theme(""))

					dirPath := filepath.Join(cacheBase, branch, "sync")
					if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
						fmt.Fprintf(out, "%s! %v\n", tr.T("Error creating directory"), err)
						continue
					}

					filePath := filepath.Join(dirPath, filepath.Base(finalURL))

					shouldDownload, err := shouldDownload(finalURL, filePath)
					if err != nil {
						fmt.Fprintf(out, "%s: %v\n", tr.T("Error checking file"), err)
						continue
					}

					if shouldDownload {
						wg.Add(1)
						go func(url, path, branch string) {
							defer wg.Done()
							if err := downloadFile(url, path); err != nil {
								fmt.Fprintf(out, "%s: %s, %v\n", tr.T("Download error"), url, err)
							} else {
								path := strings.ReplaceAll(path, "/"+branch+"/", "/"+theme.Theme(branch)+branch+theme.Theme("")+"/")
								fmt.Fprintf(out, "%s%s:%s %s\n", theme.Theme(branch), tr.T("Downloaded"), theme.Theme(""), path)
							}
						}(finalURL, filePath, branch)
					}
				}
			}
		}
	}
	wg.Wait()
	if silent {
		fmt.Println("\n## End auto update")
	}

	updateDateToFile()
}

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:     "update",
	Aliases: []string{"upgrade", "up"},
	Short:   "Update branches",
	Long:    `Update Manjaro and Archlinux pacman databases`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		silent := len(args) > 0 && args[0] == "silent"
		update(ctx.Value(ctxConfigVars).(Config), silent)
	},
}

func init() {
	updateCmd.Short = tr.T(updateCmd.Short)
	updateCmd.Long = tr.T(updateCmd.Long)
	rootCmd.AddCommand(updateCmd)
}
