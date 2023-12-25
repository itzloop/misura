package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"flag"
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

const wrapperDecl = `{{- $wn := printf "%sPrometheusWrapperImpl" .WrapperTypeName}}
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
        // Failure will be called when err != nil passing the duration and err to it
        Failure(pkg, intr, method string, duration time.Duration, err error)

        // Success will be called if err == nil passing the duration to it
        Success(pkg, intr, method string,duration time.Duration)

        // Total will be called as soon as the function is called.
        Total(pkg, intr, method string)
    }
}

func New{{$wn}}(
    wrapped {{.WrapperTypeName}},
    metrics interface{
        Failure(pkg, intr, method string, duration time.Duration, err error)
        Success(pkg, intr, method string, duration time.Duration)
        Total(pkg, intr, method string)
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
    w.metrics.Total("{{ $.PackageName }}", "{{ $.WrapperTypeName }}", "{{ .MethodName }}")
{{- if .NamedResults }}
    {{.ResultNames }} = w.wrapped.{{.MethodName}}({{ .MethodParamNames }})
{{- else }}
    {{.ResultNames }} := w.wrapped.{{.MethodName}}({{ .MethodParamNames }})
{{- end}}
    {{- if .HasError }}
    {{ $.DurationName }} := time.Since({{$.StartTimeName}})
    if err != nil {
        w.metrics.Failure("{{ $.PackageName }}", "{{ $.WrapperTypeName }}", "{{ .MethodName }}", {{ $.DurationName }}, err)
        // TODO find a way to add default values here and return the error. for now return the same thing :)
        return {{.ResultNames }}
    }

    // TODO if method has no error does success matter or not?
    w.metrics.Success("{{ $.PackageName }}", "{{ $.WrapperTypeName }}", "{{ .MethodName }}", {{ $.DurationName }})
    {{- end }}

    return {{.ResultNames }}
}
{{ end }}
`

func main() {
	// should accept multipe targets
	target := flag.String("t", "", "target interface(s)")
	flag.String("m", "all", "specify with metrics to add")
	flag.Parse()

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

	visitor := promWrapGenVisitor{
		filename: os.Getenv("GOFILE"),
		cwd:      cwd,
		fset:     fset,
		text:     file,
		target:   *target,
	}

	ast.Walk(&visitor, root)
}

type promWrapGenVisitor struct {
	cwd      string
	filename string
	text     []byte
	fset     *token.FileSet

	packageName string
	imports     string
	target      string
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
			if n.Name.String() != v.target {
				fmt.Printf("target interface is'%s' but found %s, ignoring\n", n.Name.String(), v.target)
				return nil
			}

			err := v.handleInterface(n.Name.String(), x)
			if err != nil {
				panic(err)
			}

			// we are done with this interface do not proceed further.
			return nil
		case *ast.StructType:
			fmt.Println("struct type not implemented")
			return nil
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
		n := p.Name
		if strings.Contains(p.Type, "...") {
			n += "..."
		}
		if i == len(f)-1 {
			str += n
			continue
		}
		str += n + ", "
	}

	return str
}

func (f funcParams) JoinTypes() string {
	str := ""
	for i, p := range f {
		if i == len(f)-1 {
			str += fmt.Sprintf("%s", p.Type)
			continue
		}
		str += fmt.Sprintf("%s, ", p.Type)
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
	MethodSigFull    string
	MethodName       string
	MethodParamNames string
	ResultNames      string
	NamedResults     bool
	HasError         bool
}

func (v *promWrapGenVisitor) handleInterface(intrName string, intr *ast.InterfaceType) error {
	t := template.Must(template.New("promwrapgen").Parse(wrapperDecl))

	methods := make([]method, 0, cap(intr.Methods.List))
	for _, m := range intr.Methods.List {
		method := method{
			MethodSigFull: string(v.text[m.Pos()-1 : m.End()-1]),
			MethodName:    m.Names[0].String(),
		}
		ft, ok := m.Type.(*ast.FuncType)
		if !ok {
			return errors.New("TODO: don't want to think about this now :)")
		}

		// handle parameters
		var (
			paramNameAndTypes = funcParams(make([]funcParam, 0, cap(ft.Params.List)))
			resultNames       = funcParams(make([]funcParam, 0, cap(ft.Results.List)))
			f                 = genNameHelper(1)
		)
		for _, param := range ft.Params.List {
			// get the param type
			t := string(v.text[param.Type.Pos()-1 : param.Type.End()-1])

			// if params are unnamed(i.e. func t(int, string, bool)),
			// generate name by calling f
			if param.Names == nil {
				paramNameAndTypes = append(paramNameAndTypes, funcParam{
					Name: f(),
					Type: t,
				})
				continue
			}

			// if params are named, iterate over names and handle them
			// if we encouter an underscore(_), generate a name by calling f.
			for _, name := range param.Names {
				if name.String() == "_" {
					paramNameAndTypes = append(paramNameAndTypes, funcParam{
						Name: f(),
						Type: t,
					})
					continue
				}

				paramNameAndTypes = append(paramNameAndTypes, funcParam{
					Name: name.String(),
					Type: t,
				})
			}
		}

		// TODO This is the fist and simplest solution and probably need refactoring.
		// This is for handling a case where we have underscore(_) in params. since
		// we are calling another function we need to pass all parameters so we can't
		// have a parameter with underscore. we will replace everything we had as parameters
		// with the parameters we created either by generating new name or using old ones.
		method.MethodSigFull = strings.Replace(
			method.MethodSigFull,
			string(v.text[ft.Params.Pos():ft.Params.End()-2]),
			paramNameAndTypes.Join(),
			1,
		)

		// This is used when calling the fucntion to make template simple.
		// wrapped.F({{ .MethodParamNames }}) => i.e. wrapped.F(a, b, c, d)
		method.MethodParamNames = paramNameAndTypes.JoinNames()

		// handle results
		for _, result := range ft.Results.List {
			t := string(v.text[result.Type.Pos()-1 : result.Type.End()-1])
			// TODO multipe error?
			// Assume we only return one error for now and name it err. Also
			// set HasError to true for template to add error handling.
			if t == "error" {
				method.HasError = true
				resultNames = append(resultNames, funcParam{
					Name: "err",
					Type: t,
				})
				continue
			}

			// if we have unnamedd results (i.e. f(...) (int, string, error)),
			// generate a name by calling f(). This is then used in getting the
			// return value from calling wrapped function. a, b, err = wrapped.F(...)
			if result.Names == nil {
				resultNames = append(resultNames, funcParam{
					Name: f(),
					Type: t,
				})
				continue
			}

			// If we reach here it means we have named results so set NamedResult.
			// Doing that will let us use = instead of := since we have no new variable
			// in the right part of the expression.
			method.NamedResults = true

			// if results are named, iterate over names and handle them
			// if we encouter an underscore(_), generate a name by calling f.
			for _, name := range result.Names {
				if name.String() == "_" {
					resultNames = append(resultNames, funcParam{
						Name: f(),
						Type: t,
					})
					continue
				}
				resultNames = append(resultNames, funcParam{
					Name: name.String(),
					Type: t,
				})
			}
		}

		// this will replace old return values with normalized ones.
		// if we have named results, replace with (name type, ...)
		// if we have unnamed results (i.e. (string, error, ...))
        // for other cases such as signle result without parantheses 
        // replace sth :). TODO else might be redundant but i'm to 
        // tierd to think about it now.
		var (
			nStr string
			oStr string
		)
		if method.NamedResults {
			nStr = resultNames.Join()
			oStr = string(v.text[ft.Results.Pos() : ft.Results.End()-2])
		} else if ft.Results.Closing.IsValid() {
			nStr = resultNames.JoinTypes()
			oStr = string(v.text[ft.Results.Pos() : ft.Results.End()-2])
		} else {
			nStr = resultNames.JoinTypes()
			oStr = string(v.text[ft.Results.Pos()-1 : ft.Results.End()-1])
		}

		method.MethodSigFull = strings.Replace(
			method.MethodSigFull,
			oStr,
			nStr,
			1,
		)

		// This is used when getting results from the wrapped fucntion 
        // to make template simple.
        // {{ .ResultNames }} = wrapped.F(...) => i.e. a, b, c, d := wrapped.F(...)
        // or
        // {{ .ResultNames }} := wrapped.F(...) => i.e. a, b, c, d = wrapped.F(...)
		method.ResultNames = resultNames.JoinNames()

        // finally add current method to the methods slice, to use them when 
        // populating templatess.
		methods = append(methods, method)

	}

	// populate template
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

	processed, err := imports.Process(p, b.Bytes(), nil)
	if err != nil {
		return err
	}

	fmt.Printf("writing to %s\n", tmp.Name())
	_, err = tmp.Write(processed)
	if err != nil {
		return err
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
