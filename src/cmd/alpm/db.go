package alpm

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
func (p *Package) set(desc string, long bool) bool {
	adesc := make(tdesc)
	for _, block := range strings.Split(desc, "\n\n") {
		tmp := strings.Split(block, "\n")
		if len(tmp) == 0 || len(tmp[0]) < 5 {
			continue
		}

		idx := strings.TrimRight(tmp[0][1:], "%")
		if len(tmp) > 1 {
			adesc[idx] = tmp[1:]
		} else {
			adesc[idx] = []string{}
		}
	}

	p.VERSION = getFieldString(adesc, "VERSION")
	p.NAME = getFieldString(adesc, "NAME")
	p.BUILDDATE = getFieldDate(adesc, "BUILDDATE")
	p.PACKAGER = getFieldString(adesc, "PACKAGER")
	if long {
		p.DESC = getFieldString(adesc, "DESC")
		p.URL = getFieldString(adesc, "URL")
	}
	//p.REPLACES = getFieldArray(adesc, "REPLACES")
	//p.PROVIDES = getFieldArray(adesc, "PROVIDES")
	return true
}

func (p Package) String() string {
	d := p.BUILDDATE.Format("06-01-02 15:04")
	return fmt.Sprintf(` Name:     %s
 Version:  %s
 Date:     %s`, p.NAME, p.VERSION, d)
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

/*
func getFieldArray(adesc tdesc, key string) []string {
	if len(adesc[key]) < 1 {
		return make([]string, 0)
	}
	for k, v := range adesc[key] {
		adesc[key][k] = strings.TrimSpace(strings.SplitN(v, ":", 2)[0])
	}
	return adesc[key][0:]
}
*/

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

func ExtractTarGz(gzipStream io.Reader, pkgs Packages, repo string, long bool) Packages {

	errMsg := gotext.Get("Error")

	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		fmt.Println(errMsg, err.Error())
		return pkgs
	}
	defer uncompressedStream.Close()

	tarReader := tar.NewReader(uncompressedStream)

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

		data, err := io.ReadAll(tarReader)
		if err != nil {
			fmt.Fprintln(os.Stderr, errMsg, err.Error())
			break
		}

		pkg := Package{REPO: repo}
		if pkg.set(string(data), long) {
			if _, ok := pkgs[pkg.NAME]; ok {
				fmt.Fprintf(os.Stderr, "\t # %s : %s\n", gotext.Get("ignore duplicate"), pkg.NAME)
			} else {
				pkgs[pkg.NAME] = &pkg
			}
		}
	}
	return pkgs
}

// load one branch
func Load(dirPath string, repos []string, long bool) (pkgs Packages) {
	pkgs = make(Packages, 5000)
	for _, repo := range repos {
		nb := len(pkgs)
		f, err := os.Open(filepath.Join(dirPath, repo+".db"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: can't read file %s\n", filepath.Join(dirPath, repo+".db"))
			//os.Exit(1)
			continue
		}

		pkgs = ExtractTarGz(f, pkgs, repo, long)
		f.Close()

		if len(pkgs)-nb == 0 {
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
	}
	return pkgs
}
