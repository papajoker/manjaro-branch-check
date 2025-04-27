package alpm

import (
	"fmt"
	"os"
	"path/filepath"
)

var LOCALPATH = "/var/lib/pacman/local"

func LocalDBExists() bool {
	if _, err := os.Stat(LOCALPATH); err != nil {
		return false
	}
	return true
}

func LoadLocal() (pkgs Packages, err error) {
	//var wg sync.WaitGroup
	pkgs = make(Packages)

	if !LocalDBExists() {
		return pkgs, fmt.Errorf("pacman DB not exists")
	}

	matches, _ := filepath.Glob("/var/lib/pacman/local/*/desc")
	count := len(matches)
	results := make(chan Package, count)
	for _, match := range matches {

		go func(descFile string) {
			f, err := os.Open(descFile)
			if err != nil {
				return
			}
			defer f.Close()
			pkg := Package{REPO: "local"}
			if pkg.set(f, false) {
				results <- pkg
			}
		}(match)
	}

	for range count {
		pkg := <-results
		pkgs[pkg.NAME] = &pkg
	}

	if len(pkgs) < 1 {
		return pkgs, fmt.Errorf("pacman DB empty")
	}
	return pkgs, nil
}

func FilterOnly(alls, wants Packages) (pkgs Packages) {

	pkgs = make(Packages)
	for k, _ := range wants {
		if r, ok := alls[k]; ok {
			pkgs[k] = r
		}
	}
	return pkgs
}
