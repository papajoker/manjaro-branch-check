package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

func shouldDownload(url, filePath string) (bool, error) {
	resp, err := http.Head(url)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("failed to access remote file: %s", resp.Status)
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
		return fmt.Errorf("download failed: %s", resp.Status)
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
		fmt.Println("Error creating directory:", err)
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
	/*config, err := loadConfig()
	if err != nil {
		fmt.Println("Error loading configuration:", err)
		return
	}*/

	cacheBase := filepath.Join(os.Getenv("HOME"), ".cache", "manjaro-branch-check")

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
						fmt.Fprintln(out, finalURL, Theme(branch)+"..."+Theme(""))

						dirPath := filepath.Join(cacheBase, branch, "sync")
						if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
							fmt.Fprintln(out, "Error creating directory:", err)
							continue
						}

						filePath := filepath.Join(dirPath, filepath.Base(finalURL))

						shouldDownload, err := shouldDownload(finalURL, filePath)
						if err != nil {
							fmt.Fprintln(out, "Error checking file:", err)
							continue
						}

						if shouldDownload {
							wg.Add(1)
							go func(url, path, branch string) {
								defer wg.Done()
								if err := downloadFile(url, path); err != nil {
									fmt.Fprintln(out, "Download error:", err)
								} else {
									fmt.Fprintln(out, Theme(branch)+"Downloaded:", Theme(""), path)
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
					fmt.Fprintln(out, finalURL, Theme(branch)+"..."+Theme(""))

					dirPath := filepath.Join(cacheBase, branch, "sync")
					if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
						fmt.Fprintln(out, "Error creating directory:", err)
						continue
					}

					filePath := filepath.Join(dirPath, filepath.Base(finalURL))

					shouldDownload, err := shouldDownload(finalURL, filePath)
					if err != nil {
						fmt.Fprintln(out, "Error checking file:", err)
						continue
					}

					if shouldDownload {
						wg.Add(1)
						go func(url, path, branch string) {
							defer wg.Done()
							if err := downloadFile(url, path); err != nil {
								fmt.Fprintln(out, "Download error:", err)
							} else {
								fmt.Fprintln(out, Theme(branch)+"Downloaded:", Theme(""), path)
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
}

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:     "update",
	Aliases: []string{"upgrade", "up"},
	Short:   "Update repos",
	Long:    `Update Manjaro and Archlinux pacman databases`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		silent := len(args) > 0 && args[0] == "silent"
		update(ctx.Value("configVars").(Config), silent)
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
