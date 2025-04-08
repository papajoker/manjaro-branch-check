package alpm

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/leonelquinteros/gotext"
)

type (
	tdesc map[string][]string

	Package struct {
		NAME      string
		VERSION   string
		REPO      string
		BUILDDATE time.Time
		PACKAGER  string
		URL       string
		DESC      string
	}

	Packages map[string]*Package
)

// parse desc file content
func (p *Package) set(descReader io.Reader, long bool) bool {
	scanner := bufio.NewScanner(descReader)
	adesc := make(tdesc)
	var key string
	var values []string

	flush := func() {
		if key != "" {
			descValues := make([]string, len(values))
			copy(descValues, values)
			descValues = append([]string(nil), values...)
			descValues = descValues[:len(values)]
			adesc[key] = descValues
			values = values[:0]
			key = ""
		}
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			flush()
			continue
		}
		if strings.HasPrefix(line, "%") {
			flush()
			key = strings.TrimSuffix(line[1:], "%")
		} else {
			values = append(values, line)
		}
	}
	flush()

	p.VERSION = getFieldString(adesc, "VERSION")
	p.NAME = getFieldString(adesc, "NAME")
	p.BUILDDATE = getFieldDate(adesc, "BUILDDATE")
	p.PACKAGER = getFieldString(adesc, "PACKAGER")
	if long {
		p.DESC = getFieldString(adesc, "DESC")
		p.URL = getFieldString(adesc, "URL")
	}
	return true
}

func (p Package) String() string {
	d := p.BUILDDATE.Format("06-01-02 15:04")
	return fmt.Sprintf(` Name:     %s\n Version:  %s\n Date:     %s`, p.NAME, p.VERSION, d)
}

func (p Package) Desc(maxi int) string {
	if maxi > len(p.DESC) {
		return p.DESC
	}
	return p.DESC[:maxi-1] + "…"
}

func getFieldString(adesc tdesc, key string) string {
	values, ok := adesc[key]
	if !ok || len(values) < 1 {
		return ""
	}
	return strings.TrimSpace(values[0])
}

func getFieldInt(adesc tdesc, key string) int {
	if items, ok := adesc[key]; ok && len(items) > 0 {
		if i, err := strconv.Atoi(items[0]); err == nil {
			return i
		}
	}
	return -1
}

func getFieldDate(adesc tdesc, key string) time.Time {
	if timestamp := int64(getFieldInt(adesc, key)); timestamp > -1 {
		return time.Unix(timestamp, 0)
	}
	return time.Time{}
}

func ExtractTarGz(gzipStream io.Reader, pkgs Packages, repo string, branch string, long bool) (Packages, []string) {
	errMsg := gotext.Get("Error")
	warnings := []string{}

	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		fmt.Println(errMsg, err.Error())
		return pkgs, warnings
	}
	defer uncompressedStream.Close()

	tarReader := tar.NewReader(uncompressedStream)
	var mu sync.Mutex
	var wg sync.WaitGroup

	localPkgs := make(Packages)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			fmt.Println(errMsg, err.Error())
			log.Fatalf("ExtractTarGz: Next() failed: %s", err.Error())
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		var buf strings.Builder
		if _, err := io.Copy(&buf, tarReader); err != nil {
			continue
		}

		entry := buf.String()
		wg.Add(1)
		go func(content string) {
			defer wg.Done()
			p := Package{REPO: repo}
			if p.set(strings.NewReader(content), long) {
				mu.Lock()
				if _, ok := localPkgs[p.NAME]; ok {
					warnings = append(warnings,
						fmt.Sprintf("# %s : %s (%s.%s)\n", gotext.Get("ignore duplicate"), p.NAME, branch, p.REPO),
					)
				} else {
					localPkgs[p.NAME] = &p
				}
				mu.Unlock()
			}
		}(entry)
	}
	wg.Wait()
	return localPkgs, warnings
}

// load one branch in parallel
func Load(dirPath string, repos []string, branch string, long bool) (pkgs Packages, warnings []string) {
	pkgs = make(Packages, 5000)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, repo := range repos {
		repo := repo
		wg.Add(1)
		go func() {
			defer wg.Done()
			f, err := os.Open(filepath.Join(dirPath, repo+".db"))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: can't read file %s\n", filepath.Join(dirPath, repo+".db"))
				return
			}
			defer f.Close()

			localPkgs, warn := ExtractTarGz(f, make(Packages), repo, branch, long)

			mu.Lock()
			for k, v := range localPkgs {
				pkgs[k] = v
			}
			if warn != nil {
				warnings = append(warnings, warn...)
			}

			if len(localPkgs) == 0 {
				sync := "sync"
				if strings.Contains(dirPath, "/var/lib/") {
					sync = "local"
				}
				fmt.Fprintf(
					os.Stderr,
					"%s: '%s' %s, %s\n",
					gotext.Get("warning"),
					repo, sync,
					gotext.Get("repo empty ? or all packages are ignored"),
				)
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
	return pkgs, warnings
}
