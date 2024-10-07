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

			fullpath := filepath.Join(abspath, path)
			src, err := os.ReadFile(fullpath)
			if err != nil {
				return err
			}
			fsetFile := fset.AddFile(fullpath, fset.Base(), len(src))

			lines, err := countLOC(fset, fsetFile, src)
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
		src, err := os.ReadFile(abspath)
		if err != nil {
			panic(err)
		}
		fsetFile := fset.AddFile(abspath, fset.Base(), len(src))
		lines, err := countLOC(fset, fsetFile, src)
		if err != nil {
			panic(err)
		}
		loc += lines
	}

	fmt.Printf("%d lines of code\n", loc)
}

func countLOC(fset *token.FileSet, file *token.File, src []byte) (int, error) {
	lines := make(map[int]struct{})
	var s scanner.Scanner
	s.Init(file, src, nil, 0)
	for {
		pos, tok, _ := s.Scan()
		if tok == token.EOF {
			break
		}
		lines[fset.Position(pos).Line] = struct{}{}
	}
	return len(lines), nil
}
