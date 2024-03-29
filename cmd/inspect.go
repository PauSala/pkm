/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"time"

	"bufio"
	"encoding/json"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

const (
	Red   = "\033[31m"
	Green = "\033[32m"
	Reset = "\033[0m"
)

var file string

// inspectCmd represents the inspect command
var inspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "Analyzes package.json dependencies across projects",
	Long: `
		The 'inspect' command scans the provided directories or directories listed in a file for package.json files. 
		It extracts the dependencies and their versions from each package.json file and compares them with the current version in your local declaration. 
		The command outputs a table for each project, listing the dependencies, the version in the package.json, and the current version in your local declaration. 
		`,
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		if len(args) == 0 && file == "" {
			fmt.Println("Please, provide at least one folder to inspect")
			return
		}
		paths := make([]string, 0)
		if file != "" {
			readed := readInputFile(file)
			paths = append(paths, readed...)
		}
		if len(args) > 0 {
			paths = append(paths, args...)
		}
		files := make([]string, 0)
		filesChan := make(chan []string, len(paths))

		for _, folder := range paths {
			go func(folder string) {
				jsonFiles, err := GetPackageJsonFiles(folder, "")
				if err != nil {
					panic(err)
				}
				filesChan <- jsonFiles
			}(folder)
		}

		for range paths {
			jsonFiles := <-filesChan
			files = append(files, jsonFiles...)
		}
		processJsonFiles(files)
		duration := time.Since(start)
		fmt.Println(duration)
	},
}

func init() {
	rootCmd.AddCommand(inspectCmd)
	inspectCmd.Flags().StringVarP(&file, "file", "f", "", "Path to the file containing a list of paths to projects")
}

type Package struct {
	path               string
	name               string
	version            string
	dependencies       map[string]interface{}
	devDependencies    map[string]interface{}
	dependencyInspects []DependencyInspect
}

func (p Package) PrettyPrint() {
	w := tabwriter.NewWriter(os.Stdout, 10, 1, 1, ' ', tabwriter.Debug)
	if len(p.dependencyInspects) > 0 {
		fmt.Fprintf(w, "\n")
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t\n", p.name+":"+p.path, "versionInPackageJson", "localVersion", "Check")
	}
	for _, dep := range p.dependencyInspects {
		if dep.name != "" {
			w = dep.PrintAsTable(w)
		}
	}
	w.Flush()
}

type DependencyInspect struct {
	name                 string
	versionInPackageJson string
	currentVersion       string
}

func (p DependencyInspect) PrintAsTable(w *tabwriter.Writer) *tabwriter.Writer {
	shouldUpdate := p.versionInPackageJson != p.currentVersion
	var shouldUpdateValue string
	if shouldUpdate {
		shouldUpdateValue = "✕"
	} else {
		shouldUpdateValue = "✓"
	}
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t\n", p.name, p.versionInPackageJson, p.currentVersion, shouldUpdateValue)
	return w
}

func readInputFile(path string) []string {
	file, err := os.Open(path)
	check(err)
	defer file.Close()

	scanner := bufio.NewScanner(file)
	paths := []string{}
	for scanner.Scan() {
		paths = append(paths, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}
	return paths
}

func readJsonFile(path string) (map[string]interface{}, error) {
	jsonFile, err := os.Open(path)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	defer jsonFile.Close()
	byteValue, _ := io.ReadAll(jsonFile)
	var result map[string]interface{}
	json.Unmarshal([]byte(byteValue), &result)
	return result, nil
}

func processJsonFiles(files []string) {
	packageChan := make(chan Package, len(files))
	errChan := make(chan error, len(files))

	for _, file := range files {
		go func(file string) {
			json, err := readJsonFile(file)
			if err != nil {
				errChan <- err
				return
			}
			pkg := processJsonFile(json, file)
			packageChan <- pkg
		}(file)
	}

	var packages []Package
	for i := 0; i < len(files); i++ {
		select {
		case pkg := <-packageChan:
			packages = append(packages, pkg)
		case err := <-errChan:
			panic(err)
		}
	}
	processPackages(packages)
}

func processJsonFile(json map[string]interface{}, path string) Package {

	pkg := Package{}

	pkg.path = path

	if name, ok := json["name"].(string); ok {
		pkg.name = name
	}
	if version, ok := json["version"].(string); ok {
		pkg.version = version
	}
	if dependencies, ok := json["dependencies"].(map[string]interface{}); ok {
		pkg.dependencies = dependencies
	}
	if devDependencies, ok := json["devDependencies"].(map[string]interface{}); ok {
		pkg.devDependencies = devDependencies
	}
	pkg.dependencyInspects = make([]DependencyInspect, 0)
	return pkg
}

func processPackages(pkgs []Package) {
	packageNames := make(map[string]string, 0)
	for _, pkg := range pkgs {
		if pkg.name != "" {
			packageNames[pkg.name] = pkg.version
		}
	}
	for _, pkg := range pkgs {
		for key, dep := range pkg.dependencies {
			if value, ok := packageNames[key]; ok {
				versionName := key
				versionInPackageJson := dep.(string)
				versionInPackageJson = strings.ReplaceAll(versionInPackageJson, "^", "")
				versionInPackageJson = strings.ReplaceAll(versionInPackageJson, "~", "")
				currentVersion := value
				dependencyInspect := DependencyInspect{}
				dependencyInspect.name = versionName
				dependencyInspect.currentVersion = currentVersion
				dependencyInspect.versionInPackageJson = versionInPackageJson
				pkg.dependencyInspects = append(pkg.dependencyInspects, dependencyInspect)
			}
		}
		pkg.PrettyPrint()
	}
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
