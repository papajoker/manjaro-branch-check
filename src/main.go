/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"fmt"
	"mbc/cmd"
	"runtime/debug"
)

var (
	GitBranch string
	Version   string
	BuildDate string
	GitID     string
	Project   string
)

func getAppName() string{
	info, ok := debug.ReadBuildInfo()
	if ok {
		for _, dep := range info.Deps {
			fmt.Println(dep)
			if dep.Sum == "" {
				return dep.Path
			}
		}
	 }
	 return "" //os.Args[0]
}

func main() {
	/*
	app := os.Args[0]
	fmt.Println("app:", app, "# pas avec go run")
	fmt.Println("app:", getAppName(), "# PAS avec le build")

	fmt.Println("inclu a la compilation maison: ", Project)
	fmt.Printf("\n%s Version: %v %v %v %v\n", filepath.Base(os.Args[0]), Version, GitID, GitBranch, BuildDate)
	fmt.Println(strings.Repeat("#", 30))
	*/
	cmd.Execute()
}
