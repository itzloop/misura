package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/google/uuid"
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

type funcParam struct {
	Name string
	Type string
}

type funcParams []funcParam

func (f funcParams) JoinNames() string {
	str := ""
	for i, p := range f {
		if i == len(f)-1 {
			str += p.Name
			continue
		}
		str += p.Name + ", "
	}

	return str
}

func (f funcParams) Join() string {
	str := ""
	for i, p := range f {
		if i == len(f)-1 {
			str += fmt.Sprintf("%s %s", p.Name, p.Type)
			continue
		}
		str += fmt.Sprintf("%s %s, ", p.Name, p.Type)
	}

	return str
}

type method struct {
	MethodSigFull             string
	MethodName                string
	MethodParamNames          string
	ResultNames               string
	NamedResults              bool
	ResultsContainsUnderscore bool
	HasError                  bool
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
    metrics interface{
        // Error will be called when err != nil passing the duration and err to it
        Error(duration time.Duration, err error)

        // Success will be called if err == nil passing the duration to it
        Success(duration time.Duration)

        // Total will be called as soon as the function is called.
        Total()
    }
}

func New{{$wn}}(
    wrapped {{.WrapperTypeName}},
    metrics interface{
        Error(duration time.Duration, err error)
        Success(duration time.Duration)
        Total()
    },
) *{{$wn}} {
    return &{{$wn}}{ 
        wrapped: wrapped,
        metrics: metrics,
    }
}

{{range .MethodList }}
// {{ .MethodName }} wraps another instance of {{ $.WrapperTypeName }} and 
// adds prometheus metrics. See {{ .MethodName }} on {{$wn}}.wrapped for 
// more information.
func (w *{{$wn}}) {{ .MethodSigFull }} {
    {{- if .HasError }}
    // TODO time package conflicts
    {{ $.StartTimeName }} := time.Now()
    {{- end }}
    w.metrics.Total()
{{- if and .NamedResults (not .ResultsContainsUnderscore) }}
    {{.ResultNames }} = w.wrapped.{{.MethodName}}({{ .MethodParamNames }})
{{- else }}
    {{.ResultNames }} := w.wrapped.{{.MethodName}}({{ .MethodParamNames }})
{{- end}}
    {{- if .HasError }}
    {{ $.DurationName }} := time.Since({{$.StartTimeName}})
    if err != nil {
        w.metrics.Error({{ $.DurationName }}, err)
        // TODO find a way to add default values here and return the error. for now return the same thing :)
        return {{.ResultNames }}
    }

    // TODO if method has no error does success matter or not?
    w.metrics.Success({{ $.DurationName }})
    {{- end }}

    return {{.ResultNames }}
}
{{ end }}
`
	t := template.Must(template.New("promwrapgen").Parse(wrapperDecl))
	methods := make([]method, 0, cap(intr.Methods.List))
	for _, m := range intr.Methods.List {
		method := method{
			MethodSigFull: string(v.text[m.Pos()-1 : m.End()-1]),
			MethodName:    m.Names[0].String(),
		}
		ft, ok := m.Type.(*ast.FuncType)
		if !ok {
			panic("TODO: don't want to think about this now :)")
		}

		var (
			paramNameAndTypes = funcParams(make([]funcParam, 0, cap(ft.Params.List)))
			resultNames       []string
			f                 = genNameHelper(1)
		)

		for _, param := range ft.Params.List {
			t := string(v.text[param.Type.Pos()-1 : param.Type.End()-1])
			if param.Names == nil {
				paramNameAndTypes = append(paramNameAndTypes, funcParam{
					Name: f(),
					Type: t,
				})
				continue
			}

			for _, name := range param.Names {
				if name.String() == "_" {
					paramNameAndTypes = append(paramNameAndTypes, funcParam{
						Name: f(),
						Type: t,
					})
					// TODO find a more efficient way
					// method.MethodSigFull = strings.Replace(method.MethodSigFull, "_", newName, 1)
					continue
				}

				paramNameAndTypes = append(paramNameAndTypes, funcParam{
					Name: name.String(),
					Type: t,
				})
			}
		}

		method.MethodSigFull = strings.Replace(method.MethodSigFull, string(v.text[ft.Params.Pos():ft.Params.End()-2]), paramNameAndTypes.Join(), 1)
		method.MethodParamNames = paramNameAndTypes.JoinNames()

		returnNameHelper := func(returnType string, resultIdents []*ast.Ident, f func() string) {
			if resultIdents != nil {
				method.NamedResults = true
			}

			if returnType == "error" {
				// found error
				for _, ident := range resultIdents {
					if ident.String() == "_" {
						method.ResultsContainsUnderscore = true
					}
				}
				method.HasError = true
				resultNames = append(resultNames, "err")
				return
			}

			if method.NamedResults { // if named just pick the names
				for _, n := range resultIdents {
					if n.String() == "_" {
						method.ResultsContainsUnderscore = true
						resultNames = append(resultNames, f())
						continue
					}
					resultNames = append(resultNames, n.String())
				}

				return
			}

			resultNames = append(resultNames, f())
		}

		for _, result := range ft.Results.List {
			// TODO having multiple errors make no sense in my opinion but find out about this
			// if type is err just use err
			switch r := result.Type.(type) {
			case *ast.Ident:
				returnNameHelper(r.String(), result.Names, f)
			case *ast.StarExpr:
				returnNameHelper("", result.Names, f)
			case *ast.SelectorExpr:
				returnNameHelper("", result.Names, f)
			}
		}

		method.ResultNames = strings.Join(resultNames, ", ")
		methods = append(methods, method)

	}

	id := uuid.New()
	vals := struct {
		PackageName     string
		WrapperTypeName string
		MethodList      []method
		Imports         string
		StartTimeName   string
		DurationName    string
	}{
		PackageName:     v.packageName,
		WrapperTypeName: intrName,
		MethodList:      methods,
		Imports:         v.imports,
		StartTimeName:   fmt.Sprintf("start_%s", hex.EncodeToString(id[:4])),
		DurationName:    fmt.Sprintf("duration_%s", hex.EncodeToString(id[4:8])),
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

func genNameHelper(count int) func() string {
	start := -1
	if count < 1 {
		count = 1
	}

	alphabet := "abcdefghijklmnopqrstuvwxyz"

	return func() string {
		start++
		if start == len(alphabet) {
			start = 0
			count++
		}

		if start+count <= len(alphabet) {
			return string([]byte(alphabet)[start : start+count])
		}

		return string([]byte(alphabet)[start:start+count]) + string([]byte(alphabet)[0:start+count-len(alphabet)])
	}
}
