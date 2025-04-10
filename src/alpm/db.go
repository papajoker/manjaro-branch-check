package alpm

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
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
		line := scanner.Bytes()
		if len(line) == 0 {
			flush()
			continue
		}
		if len(line) > 0 && line[0] == '%' {
			flush()
			key = string(line[1 : len(line)-1])
		} else {
			values = append(values, string(line))
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

func ExtractTarGz(gzipStream io.Reader, repo string, branch string, long bool, results chan<- Package, warningsChan chan<- []string) {

	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		warningsChan <- []string{fmt.Sprintf("gzip.NewReader error: %s %s:%s\n", err.Error(), branch, repo)}
		return
	}
	defer uncompressedStream.Close()

	tarReader := tar.NewReader(uncompressedStream)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			warningsChan <- []string{fmt.Sprintf("ExtractTarGz: Next() failed: %s\n", err.Error())}
			return
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		var buf bytes.Buffer //strings.Builder
		if _, err := io.Copy(&buf, tarReader); err != nil {
			warningsChan <- []string{fmt.Sprintf("io.Copy error: %s\n", err.Error())}
			continue
		}

		p := Package{REPO: repo}
		if p.set(bytes.NewReader(buf.Bytes()), long) {
			results <- p
		}
	}
}

// load one branch in parallel
func Load(dirPath string, repos []string, branch string, long bool) (pkgs Packages, warnings []string) {
	numRepos := len(repos)
	jobs := make(chan string, numRepos)
	results := make(chan Package)
	warningsChan := make(chan []string, numRepos)
	var wg sync.WaitGroup
	var wgResults sync.WaitGroup

	// Lancement des workers
	for w := 1; w <= numRepos; w++ {
		wg.Add(1)
		go worker(dirPath, branch, long, jobs, results, warningsChan, &wg)
	}

	/*
		repos := []string{core extra kde-unstable multilib ...}
		je désires faire une comparaison de ce type:
			if pkgs[pkg.NAME].repo "<" pkg.repo
			c'est l'ordre dans "repos" qui indique un ordre entre les repo de paquets
	*/
	compareRepo := func(repos []string, a Package, b Package) int {
		if a.REPO == b.REPO {
			fmt.Println(a.REPO, "==", b.REPO)
			return 0
		}

		indexOf := func(value string, repos []string) int {
			for i, v := range repos {
				if v == value {
					return i
				}
			}
			return len(repos) + 1
		}
		if indexOf(a.REPO, repos) < indexOf(b.REPO, repos) {
			return 1
		}
		return -1
	}

	wgResults.Add(1)
	go func() {
		defer wgResults.Done()
		pkgs = make(Packages)
		for pkg := range results {
			if _, ok := pkgs[pkg.NAME]; ok {
				if compareRepo(repos, *pkgs[pkg.NAME], pkg) > 0 {
					warnings = append(warnings, fmt.Sprintf("# %s : %s (%s.%s)\n", gotext.Get("ignore duplicate"), pkg.NAME, branch, pkg.REPO))
				} else {
					warnings = append(warnings, fmt.Sprintf("# %s : %s (%s.%s)\n", gotext.Get("ignore duplicate"), pkgs[pkg.NAME].NAME, branch, pkgs[pkg.NAME].REPO))
					pkgs[pkg.NAME] = &pkg
				}
				continue
			}
			pkgs[pkg.NAME] = &pkg
		}
		//warnings = append(warnings, fmt.Sprintf("# %s : %s (%s.%s)\n", gotext.Get("ignore duplicate"), "truc", "stable", "core"))
		//warnings = append(warnings, fmt.Sprintf("# %s : %s (%s.%s)\n", gotext.Get("ignore duplicate"), "machin", "stable", "core"))
	}()

	for _, repo := range repos {
		jobs <- repo
	}
	close(jobs)

	wg.Wait()
	close(results)
	wgResults.Wait()

	close(warningsChan)
	for warn := range warningsChan {
		if warn != nil {
			warnings = append(warnings, warn...)
		}
	}

	return pkgs, warnings
}

func worker(dirPath string, branch string, long bool, jobs <-chan string, results chan<- Package, warningsChan chan<- []string, wg *sync.WaitGroup) {
	defer wg.Done()
	for repo := range jobs {
		f, err := os.Open(filepath.Join(dirPath, repo+".db"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: can't read file %s\n", filepath.Join(dirPath, repo+".db"))
			warningsChan <- []string{fmt.Sprintf("Error: can't read file %s\n", filepath.Join(dirPath, repo+".db"))}
			continue
		}
		if fileInfo, _ := os.Stat(filepath.Join(dirPath, repo+".db")); fileInfo.Size() < 1 {
			continue
		}
		ExtractTarGz(f, repo, branch, long, results, warningsChan)
		f.Close()
	}
}
