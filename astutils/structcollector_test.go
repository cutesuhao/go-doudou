package astutils

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
)

func ExampleStruct() {
	file := "/Users/wubin1989/workspace/cloud/usersvc/vo/vo.go"
	fset := token.NewFileSet()
	root, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	sc := NewStructCollector()
	ast.Walk(sc, root)
	fmt.Println(sc.Structs)

	//ast.Print(fset, root)

	var a = []string{"a", "b"}
	var b = []string{"b", "a"}
	fmt.Println(reflect.DeepEqual(a, b))

	//Output:
}

func ExampleInter() {
	file := "/Users/wubin1989/workspace/cloud/usersvc/svc.go"
	fset := token.NewFileSet()
	root, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	var sc InterfaceCollector
	ast.Walk(&sc, root)
	fmt.Println(sc.Interfaces)

	ast.Print(fset, root)

	var a = []string{"a", "b"}
	var b = []string{"b", "a"}
	fmt.Println(reflect.DeepEqual(a, b))

	//Output:
}
