package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

// Dependency represents a package and its dependencies
type Dependency struct {
	Name         string
	Dependencies []string
}

// getModulePath finds the module path by reading the go.mod file
func getModulePath(dir string) (string, error) {
	modFile := filepath.Join(dir, "go.mod")
	data, err := os.ReadFile(modFile)
	if err != nil {
		return "", fmt.Errorf("could not read go.mod: %v", err)
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}

	return "", fmt.Errorf("module path not found in go.mod")
}

// stripModulePrefix removes the module prefix from a package path
func stripModulePrefix(modulePath, pkgPath string) string {
	return strings.TrimPrefix(pkgPath, modulePath+"/")
}

// buildDAG builds the dependency graph for the Go project, considering only project-specific packages
func buildDAG(dir, modulePath string) (map[string]*Dependency, error) {
	// Load the Go packages from the given directory
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedImports,
		Dir:  dir,
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, err
	}

	dag := make(map[string]*Dependency)

	// Iterate through each package
	for _, pkg := range pkgs {
		if pkg.Name == "" || !strings.HasPrefix(pkg.PkgPath, modulePath) {
			continue
		}

		// Strip the module prefix from the package path
		pkgPath := stripModulePrefix(modulePath, pkg.PkgPath)

		// Initialize the package in the DAG if it doesn't exist
		if _, exists := dag[pkgPath]; !exists {
			dag[pkgPath] = &Dependency{
				Name:         pkgPath,
				Dependencies: []string{},
			}
		}

		// Add the package's imports as dependencies, but only for project-specific packages
		for importPath := range pkg.Imports {
			if strings.HasPrefix(importPath, modulePath) {
				strippedImportPath := stripModulePrefix(modulePath, importPath)
				dag[pkgPath].Dependencies = append(dag[pkgPath].Dependencies, strippedImportPath)
			}
		}

		// Sort dependencies alphabetically
		sort.Strings(dag[pkgPath].Dependencies)
	}

	return dag, nil
}

// prettyPrintDAG prints the DAG in alphabetical order
func prettyPrintDAG(dag map[string]*Dependency) {
	// Collect and sort package names alphabetically
	pkgs := make([]string, 0, len(dag))
	for pkg := range dag {
		pkgs = append(pkgs, pkg)
	}
	sort.Strings(pkgs)

	// Print the packages and their dependencies
	for _, pkg := range pkgs {
		dep := dag[pkg]
		fmt.Printf("Package: %s\n", pkg)

		// Only print dependencies if there are any
		if len(dep.Dependencies) > 0 {
			fmt.Printf("  Depends on:\n")
			for _, d := range dep.Dependencies {
				fmt.Printf("    %s\n", d)
			}
		}
		fmt.Println(strings.Repeat("-", 40))
	}
}

func main() {
	// Accept the directory as a command-line argument
	dir := flag.String("dir", ".", "Directory of the Go project")
	flag.Parse()

	// Check if the provided directory exists
	if _, err := os.Stat(*dir); os.IsNotExist(err) {
		log.Fatalf("Directory does not exist: %s", *dir)
	}

	// Get the module path from go.mod
	modulePath, err := getModulePath(*dir)
	if err != nil {
		log.Fatalf("Error retrieving module path: %v", err)
	}

	// Build the DAG of dependencies
	dag, err := buildDAG(*dir, modulePath)
	if err != nil {
		log.Fatalf("Error building DAG: %v", err)
	}

	// Pretty-print the DAG
	prettyPrintDAG(dag)
}
