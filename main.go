package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path"
	"strings"
	"text/template"

	"golang.org/x/tools/imports"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	fmt.Printf("  cwd = %s\n", cwd)
	fmt.Printf("  os.Args = %#v\n", os.Args)

	for _, ev := range []string{"GOARCH", "GOOS", "GOFILE", "GOLINE", "GOPACKAGE", "DOLLAR"} {
		fmt.Println("  ", ev, "=", os.Getenv(ev))
	}

	filePath := path.Join(cwd, os.Getenv("GOFILE"))

	log.Printf("generating prometheus wrapper for '%s'\n", filePath)

	// read file
	file, err := os.ReadFile(filePath)
	if err != nil {
		panic(err)
	}

	fset := token.NewFileSet()
	root, err := parser.ParseFile(fset, filePath, file, parser.ParseComments)
	if err != nil {
		log.Fatalln(err)
	}

	visitor := promWrapGenVisitor{filename: os.Getenv("GOFILE"), cwd: cwd, fset: fset, text: file}

	ast.Walk(&visitor, root)

}

type promWrapGenVisitor struct {
	cwd      string
	filename string
	text     []byte
	fset     *token.FileSet

	packageName string
	imports     string
}

func (v *promWrapGenVisitor) Visit(nRaw ast.Node) ast.Visitor {
	if nRaw == nil {
		return nil
	}

	switch n := nRaw.(type) {
	case *ast.File:
		v.packageName = n.Name.String()
	case *ast.GenDecl:
		// we only care about import statements
		if n.Tok.String() != "import" {
			return v
		}

		v.imports = string(v.text[n.Pos()-1 : n.End()-1])
	case *ast.TypeSpec:
		switch x := n.Type.(type) {
		case *ast.InterfaceType:
			err := v.handleInterface(n.Name.String(), x)
			if err != nil {
				panic(err)
			}

			// we are done with this interface do not proceed further.
			return nil
		case *ast.StructType:
			fmt.Println("struct type not implemented")
			return v
		}
	}

	return v
}

type method struct {
	MethodSigFull    string
	MethodName       string
	MethodParamNames string
}

func (v *promWrapGenVisitor) handleInterface(intrName string, intr *ast.InterfaceType) error {

	// TODO make this a text/template

	wrapperDecl := `{{- $wn := printf "%sPrometheusWrapperImpl" .WrapperTypeName}}
// This code is generate by github.com/itzloop/promwrapgen. DO NOT EDIT!
package {{ .PackageName }}

{{ .Imports }}

// {{$wn}} wraps {{ .WrapperTypeName }} and adds metrics like:
// 1. success count
// 2. error count
// 3. total count
// 4. duration
type {{$wn}} struct {
    // TODO what are fields are required
    wrapped {{.WrapperTypeName}}
}

func New{{$wn}}(wrapped {{.WrapperTypeName}}) *{{$wn}} {
    return &{{$wn}}{ 
        wrapped: wrapped,
        // TODO other fields
    }
}

{{range .MethodList }}
func (w *{{$wn}}) {{ .MethodSigFull }} {
    // TODO handle unnamed parameters
    // TODO handle return type
    // TODO handle metrics
    //      1. total
    //      2. success
    //      3. error
    //      4. duration
    return w.wrapped.{{.MethodName}}({{ .MethodParamNames }})
}
{{ end }}
`
	t := template.Must(template.New("promwrapgen").Parse(wrapperDecl))
	methods := make([]method, 0, cap(intr.Methods.List))
	for _, m := range intr.Methods.List {
        method := method{
        	MethodSigFull:    string(v.text[m.Pos()-1 : m.End()-1]),
        	MethodName:       m.Names[0].String(),
        }
		ft, ok := m.Type.(*ast.FuncType)
		if !ok {
			panic("wtf")
		}

		var paramNames []string

		for _, param := range ft.Params.List {
			for _, name := range param.Names {
				paramNames = append(paramNames, name.String())
			}
		}

		method.MethodParamNames = strings.Join(paramNames, ", ")
		methods = append(methods, method)

	}
	vals := struct {
		PackageName     string
		WrapperTypeName string
		MethodList      []method
		Imports         string
	}{
		PackageName:     v.packageName,
		WrapperTypeName: intrName,
		MethodList:      methods,
		Imports:         v.imports,
	}

	p := path.Join(v.cwd, strings.Split(v.filename, ".")[0]+"_promwrapgen.go")
	tmp, err := os.Create(p)
	if err != nil {
		panic(err)
	}
	defer tmp.Close()

	b := &bytes.Buffer{}

	t.Execute(b, vals)

	// processed := b.Bytes()
	processed, err := imports.Process(p, b.Bytes(), nil)
	if err != nil {
		panic(err)
	}

	fmt.Printf("writing to %s\n", tmp.Name())
	_, err = tmp.Write(processed)
	if err != nil {
		panic(err)
	}

	return nil
}
