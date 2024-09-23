package main

import (
	"flag"
	"fmt"
	"go/scanner"
	"go/token"
	"io/fs"
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

	loc := 0

	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("0 arguments given")
		return
	}
	abspath, err := filepath.Abs(args[0])
	if err != nil {
		panic(err)
	}
	f, err := os.Open(abspath)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	fstat, err := f.Stat()
	if err != nil {
		panic(err)
	}
	fset := token.NewFileSet()
	if fstat.IsDir() {
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

			lines, err := countLOC(fset, filepath.Join(abspath, path))
			if err != nil {
				return err
			}
			loc += lines
			return nil
		})
		if err != nil {
			panic(err)
		}
	} else if strings.HasSuffix(abspath, ".go") {
		lines, err := countLOC(fset, abspath)
		if err != nil {
			panic(err)
		}
		loc += lines
	}

	fmt.Printf("%d lines of code\n", loc)
}

func countLOC(fset *token.FileSet, path string) (int, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	fsetFile := fset.AddFile(path, fset.Base(), len(src))
	lines := make(map[int]struct{})
	var s scanner.Scanner
	s.Init(fsetFile, src, nil, 0)
	for {
		pos, tok, _ := s.Scan()
		if tok == token.EOF {
			break
		}
		lines[fset.Position(pos).Line] = struct{}{}
	}
	return len(lines), nil
}
