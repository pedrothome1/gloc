package main

import (
	"flag"
	"fmt"
	"go/ast"
	"log"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

// Package represents a package and its dependencies
type Package struct {
	Name      string
	Imports   []string
	Functions []string
	Types     []string
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
func buildDAG(dir, modulePath string) (map[string]*Package, error) {
	// Load the Go packages from the given directory
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedImports | packages.NeedFiles | packages.NeedSyntax,
		Dir:  dir,
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, err
	}

	dag := make(map[string]*Package)

	// Iterate through each package
	for _, pkg := range pkgs {
		if pkg.Name == "" || !strings.HasPrefix(pkg.PkgPath, modulePath) {
			continue
		}

		// Strip the module prefix from the package path
		pkgPath := stripModulePrefix(modulePath, pkg.PkgPath)

		// Initialize the package in the DAG if it doesn't exist
		if _, exists := dag[pkgPath]; !exists {
			dag[pkgPath] = &Package{
				Name:    pkgPath,
				Imports: []string{},
			}
		}

		// Add the package's imports as dependencies, but only for project-specific packages
		for importPath := range pkg.Imports {
			if strings.HasPrefix(importPath, modulePath) {
				strippedImportPath := stripModulePrefix(modulePath, importPath)
				dag[pkgPath].Imports = append(dag[pkgPath].Imports, strippedImportPath)
			}
		}

		// Types
		var types []*ast.TypeSpec
		var funcs []*ast.FuncDecl
		for _, file := range pkg.Syntax {
			for _, decl := range file.Decls {
				switch d := decl.(type) {
				case *ast.GenDecl:
					for _, spec := range d.Specs {
						switch s := spec.(type) {
						case *ast.TypeSpec:
							types = append(types, s)
						}
					}
				case *ast.FuncDecl:
					if d.Recv != nil {
						break
					}
					funcs = append(funcs, d)
				}
			}
		}
		var (
			structs    []string
			interfaces []string
			otherTypes []string
		)
		for _, t := range types {
			switch t.Type.(type) {
			case *ast.StructType:
				structs = append(structs, fmt.Sprintf("type %s struct", t.Name.Name))
			case *ast.InterfaceType:
				interfaces = append(interfaces, fmt.Sprintf("type %s interface", t.Name.Name))
			default:
				otherTypes = append(otherTypes, fmt.Sprintf("type %s %s", t.Name.Name, printExpr(t.Type)))
			}
		}

		sort.Strings(dag[pkgPath].Imports)
		sort.Strings(interfaces)
		sort.Strings(structs)
		sort.Strings(otherTypes)
		sort.Strings(dag[pkgPath].Functions)

		dag[pkgPath].Types = slices.Concat(interfaces, structs, otherTypes)

		for _, f := range funcs {
			signature := fmt.Sprintf("func %s", printFuncType(f.Type, f.Name.Name))
			dag[pkgPath].Functions = append(dag[pkgPath].Functions, signature)
		}
	}

	return dag, nil
}

// prettyPrintDAG prints the DAG in alphabetical order
func prettyPrintDAG(dag map[string]*Package) {
	// Collect and sort package names alphabetically
	pkgs := make([]string, 0, len(dag))
	for pkg := range dag {
		pkgs = append(pkgs, pkg)
	}
	sort.Strings(pkgs)

	// Print the packages and their dependencies
	for _, pkg := range pkgs {
		p := dag[pkg]
		fmt.Printf("Package: %s\n", pkg)

		// Only print dependencies if there are any
		if len(p.Imports) > 0 {
			fmt.Printf("  Imports:\n")
			for _, d := range p.Imports {
				fmt.Printf("    %s\n", d)
			}
		}
		if len(p.Types) > 0 {
			fmt.Printf("  Types:\n")
			for _, t := range p.Types {
				fmt.Printf("    %s\n", t)
			}
		}
		if len(p.Functions) > 0 {
			fmt.Printf("  Functions:\n")
			for _, t := range p.Functions {
				fmt.Printf("    %s\n", t)
			}
		}
		fmt.Println(strings.Repeat("-", 40))
	}
}

func printExpr(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.Ellipsis:
		return "..."
	case *ast.BasicLit:
		return e.Value
	case *ast.SelectorExpr:
		return fmt.Sprintf("%s.%s", printExpr(e.X), e.Sel.Name)
	case *ast.StarExpr:
		return fmt.Sprintf("*%s", printExpr(e.X))
	case *ast.ArrayType:
		if e.Len == nil {
			return fmt.Sprintf("[]%s", printExpr(e.Elt))
		} else {
			return fmt.Sprintf("[%s]%s", printExpr(e.Len), printExpr(e.Elt))
		}
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", printExpr(e.Key), printExpr(e.Value))
	case *ast.ChanType:
		if e.Dir == ast.RECV {
			return fmt.Sprintf("<-chan %s", printExpr(e.Value))
		} else if e.Dir == ast.SEND {
			return fmt.Sprintf("chan<- %s", printExpr(e.Value))
		} else {
			return fmt.Sprintf("chan %s", printExpr(e.Value))
		}
	case *ast.StructType:
		return fmt.Sprintf("struct{%s}", strings.Join(printFieldList(e.Fields), "; "))
	case *ast.InterfaceType:
		return fmt.Sprintf("interface{%s}", strings.Join(printFieldList(e.Methods), "; "))
	case *ast.FuncType:
		return printFuncType(e, "func")
	default:
		return "UNMAPPED"
	}
}

func printFuncType(fn *ast.FuncType, name string) string {
	if fn.Results == nil {
		return fmt.Sprintf("%s(%s)", name, strings.Join(printFieldList(fn.Params), ", "))
	}
	results := strings.Join(printFieldList(fn.Results), ", ")
	if strings.Contains(results, " ") {
		results = "(" + results + ")"
	}
	return fmt.Sprintf("%s(%s) %s", name, strings.Join(printFieldList(fn.Params), ", "), results)
}

func printFieldList(list *ast.FieldList) []string {
	if list.List == nil {
		return []string{}
	}

	ret := make([]string, 0, len(list.List))
	for _, field := range list.List {
		if len(field.Names) == 0 {
			ret = append(ret, printExpr(field.Type))
			continue
		}
		if fn, ok := field.Type.(*ast.FuncType); ok {
			ret = append(ret, printFuncType(fn, field.Names[0].Name))
			continue
		}
		names := make([]string, 0, len(field.Names))
		for _, n := range field.Names {
			names = append(names, n.Name)
		}
		ret = append(ret, fmt.Sprintf("%s %s", strings.Join(names, ", "), printExpr(field.Type)))
	}
	return ret
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
