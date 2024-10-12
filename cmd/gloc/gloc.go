package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var (
	ignoreTests = false
	ignoreDirs  []string
)

func main() {
	flag.Parse()
	log.SetFlags(0)

	var (
		sumLines  int
		sumTypes  int
		sumVars   int
		sumConsts int
		sumFuncs  int
	)

	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("0 arguments given")
		return
	}
	fstat, abspath, err := fileInfo(args[0])
	if err != nil {
		panic(err)
	}
	if fset := token.NewFileSet(); fstat.IsDir() {
		fmt.Printf("%-30s%-10s%-10s%-10s%-10s%-10s\n", "Path", "Lines", "Types", "Funcs", "Consts", "Vars")
		err = fs.WalkDir(os.DirFS(abspath), ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			if ignoreTests && strings.HasSuffix(path, "_test.go") {
				return nil
			}
			for _, ignored := range ignoreDirs {
				if strings.Contains(path, ignored+"/") {
					return nil
				}
			}

			fullpath := filepath.Join(abspath, path)
			src, err := os.ReadFile(fullpath)
			if err != nil {
				return err
			}
			fsetFile := fset.AddFile(fullpath, fset.Base(), len(src))
			toks, err := scanTokens(fsetFile, src)
			if err != nil {
				return err
			}

			var (
				lines  int
				types  int
				vars   int
				consts int
				funcs  int
			)
			lines, err = countLOC(fset, toks)
			if err != nil {
				return err
			}
			pfile, err := parser.ParseFile(fset, fullpath, src, parser.SkipObjectResolution)
			if err != nil {
				return err
			}
			for _, decl := range pfile.Decls {
				switch dt := decl.(type) {
				case *ast.GenDecl:
					switch dt.Tok {
					case token.CONST:
						consts += len(dt.Specs)
					case token.VAR:
						vars += len(dt.Specs)
					case token.TYPE:
						types += len(dt.Specs)
					}
				case *ast.FuncDecl:
					// only counting functions not methods
					if dt.Recv == nil {
						funcs++
					}
				}
			}

			fmt.Printf("%-30s%-10d%-10d%-10d%-10d%-10d\n", path, lines, types, funcs, consts, vars)

			sumLines += lines
			sumTypes += types
			sumVars += vars
			sumConsts += consts
			sumFuncs += funcs
			return nil
		})
		if err != nil {
			panic(err)
		}
	} else if strings.HasSuffix(abspath, ".go") {
		src, err := os.ReadFile(abspath)
		if err != nil {
			panic(err)
		}
		fsetFile := fset.AddFile(abspath, fset.Base(), len(src))
		toks, err := scanTokens(fsetFile, src)
		if err != nil {
			panic(err)
		}
		lines, err := countLOC(fset, toks)
		if err != nil {
			panic(err)
		}
		sumLines += lines
	}

	fmt.Printf("%-30s%-10d%-10d%-10d%-10d%-10d\n", "Total", sumLines, sumTypes, sumFuncs, sumConsts, sumVars)
}

// fileInfo returns the os.FileInfo, the absolute path and error.
func fileInfo(path string) (os.FileInfo, string, error) {
	abspath, err := filepath.Abs(path)
	if err != nil {
		return nil, "", err
	}
	f, err := os.Open(abspath)
	defer f.Close()
	if err != nil {
		return nil, abspath, err
	}
	fstat, err := f.Stat()
	if err != nil {
		return nil, abspath, err
	}
	return fstat, abspath, nil
}

func countLOC(fset *token.FileSet, toks []tokenInfo) (int, error) {
	lines := make(map[int]struct{})
	for _, t := range toks {
		lines[fset.Position(t.pos).Line] = struct{}{}
	}
	return len(lines), nil
}

type tokenInfo struct {
	tok token.Token
	pos token.Pos
}

func scanTokens(file *token.File, src []byte) ([]tokenInfo, error) {
	var ret []tokenInfo
	var s scanner.Scanner
	s.Init(file, src, nil, 0)
	for {
		pos, tok, _ := s.Scan()
		if tok == token.EOF {
			break
		}
		ret = append(ret, tokenInfo{
			tok: tok,
			pos: pos,
		})
	}
	return ret, nil
}
