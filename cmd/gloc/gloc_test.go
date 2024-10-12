package main

import (
	"go/token"
	"testing"
)

const (
	srcWithBlanks = `package main

import (
	"fmt"
	"time"
)

const b = 1
const (
	Stuff    = 123
	Registry = "source"
)

var a = 1

var (
	v1 = "v1"
	v2 = 2
)

type Foo struct {
	Name string
}

type Bar interface {
	Do() (string, error)
}

func main() {
	res := a + v2 + Stuff + b
	str := Registry + v1
	f1 := &Foo{}
	b1 := new(Bar)
	fmt.Println(str)
	fmt.Println(f1)
	fmt.Println(b1)
	fmt.Println(res)
	fmt.Println("Hello!")
	fmt.Println(time.Now())
}`

	srcNoBlanks = `package main
import (
	"fmt"
	"time"
)
const b = 1
const (
	Stuff    = 123
	Registry = "source"
)
var a = 1
var (
	v1 = "v1"
	v2 = 2
)
type Foo struct {
	Name string
}
type Bar interface {
	Do() (string, error)
}
func main() {
	res := a + v2 + Stuff + b
	str := Registry + v1
	f1 := &Foo{}
	b1 := new(Bar)
	fmt.Println(str)
	fmt.Println(f1)
	fmt.Println(b1)
	fmt.Println(res)
	fmt.Println("Hello!")
	fmt.Println(time.Now())
}`
)

func TestCountLOC(t *testing.T) {
	want := 33

	fset := token.NewFileSet()
	f := fset.AddFile("main.go", fset.Base(), len(srcWithBlanks))
	got, err := countLOC(fset, f, []byte(srcWithBlanks))
	if err != nil {
		t.Errorf("err not nil: %s\n", err.Error())
	}
	if want != got {
		t.Errorf("countLOC(..., src1): want %d, got %d\n", want, got)
	}

	fset = token.NewFileSet()
	f = fset.AddFile("main.go", fset.Base(), len(srcNoBlanks))
	got, err = countLOC(fset, f, []byte(srcNoBlanks))
	if err != nil {
		t.Errorf("err not nil: %s\n", err.Error())
	}
	if want != got {
		t.Errorf("countLOC(..., src2): want %d, got %d\n", want, got)
	}
}
