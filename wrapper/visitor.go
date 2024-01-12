package wrapper

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"strings"

	"github.com/itzloop/promwrapgen/wrapper/types"
)

type TypeVisitorOpts struct {
	CWD      string
	FileName string
	Targets  types.Strings
}

type TypeVisitor struct {
	err error

	opts TypeVisitorOpts

	fset        *token.FileSet
	packageName string
	imports     string

	text []byte

	// TODO make this interface
	g *WrapperGenerator
}

func NewTypeVisitor(g *WrapperGenerator, opts TypeVisitorOpts) (*TypeVisitor, error) {
	p := path.Join(opts.CWD, opts.FileName)
	f, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	return &TypeVisitor{
		g:    g,
		text: f,
		opts: opts,
	}, nil
}

func (t *TypeVisitor) Walk() error {
	fset := token.NewFileSet()
	root, err := parser.ParseFile(fset, t.opts.FileName, t.text, parser.ParseComments)
	if err != nil {
		return err
	}

	ast.Walk(t, root)
	return t.err
}

func (t *TypeVisitor) Visit(nRaw ast.Node) ast.Visitor {
	if nRaw == nil {
		return nil
	}

	switch n := nRaw.(type) {
	case *ast.File:
		t.packageName = n.Name.String()
	case *ast.GenDecl:
		// we only care about import statements
		if n.Tok.String() != "import" {
			return t
		}

		// copy all the imports from the file to the wrapper
		t.imports = string(t.text[n.Pos()-1 : n.End()-1])
	case *ast.TypeSpec:
		switch x := n.Type.(type) {
		case *ast.InterfaceType:
			// TODO support unnamed interfaces?
			// Ignore unnamed interfaces for now
			if n.Name.String() == "" {
				fmt.Printf("ignoring unnamed interface\n")
				return nil
			}
			if !t.opts.Targets.Exists(n.Name.String()) {
				fmt.Printf("ignoring %s since it is not a target\n", n.Name.String())
				return nil
			}

			err := t.handleInterface(n.Name.String(), x)
			if err != nil {
				panic(err)
			}

			// we are done with this interface do not proceed further.
			return nil
		case *ast.StructType:
			// TODO support unnamed structs?
			// Ignore unnamed structs for now
			if n.Name.String() == "" {
				fmt.Printf("ignoring unnamed struct\n")
				return nil
			}

			if !t.opts.Targets.Exists(n.Name.String()) {
				fmt.Printf("ignoring %s since it is not a target\n", n.Name.String())
				return nil
			}

			fmt.Println("struct type not implemented")
			return nil
		}
	}

	return t
}

func (t *TypeVisitor) handleInterface(intrName string, intr *ast.InterfaceType) error {
	if t.packageName == "" {
		return errors.New("TypeVisitor: package name can't be empty")
	}

	methods := make([]types.Method, 0, cap(intr.Methods.List))
	for _, m := range intr.Methods.List {
		method := types.Method{
			MethodSigFull: string(t.text[m.Pos()-1 : m.End()-1]),
			MethodName:    m.Names[0].String(),
		}
		ft, ok := m.Type.(*ast.FuncType)
		if !ok {
			return errors.New("TODO: don't want to think about this now :)")
		}

		f := genNameHelper(1)
		t.handleParams(&method, ft.Params, f)
		t.handleResults(&method, ft.Results, f)

		// finally add current method to the methods slice, to use them when
		// populating templatess.
		methods = append(methods, method)

	}

	// populate template
	randBytes := make([]byte, 4)
	_, err := rand.Read(randBytes)
	if err != nil {
		return err
	}

	return t.g.Generate(t.opts.CWD, t.opts.FileName, TemplateVals{
		PackageName:     t.packageName,
		WrapperTypeName: intrName,
		MethodList:      methods,
		Imports:         t.imports,
		RandomHex:       strings.ToUpper(hex.EncodeToString(randBytes)),
	})
}

func (t *TypeVisitor) handleParams(m *types.Method, params *ast.FieldList, f func() string) types.FuncParams {
	var paramNames types.FuncParams

	if params == nil {
		return paramNames
	}

	// handle parameters
	for _, param := range params.List {
		// get the param type
		t := string(t.text[param.Type.Pos()-1 : param.Type.End()-1])
		// TODO This works for now but make sure it's context.Context not any random Context
		// if params are unnamed(i.e. func t(int, string, bool)),
		// generate name by calling f
		if param.Names == nil {
			n := f()
			if !m.HasCtx && strings.Contains(t, "Context") {
				n = "ctx"
				m.HasCtx = true
				m.Ctx = n
			}
			paramNames = append(paramNames, types.FuncParam{
				Name: n,
				Type: t,
			})

			continue
		}

		// if params are named, iterate over names and handle them
		// if we encouter an underscore(_), generate a name by calling f.
		for _, name := range param.Names {
			if name.String() == "_" {
				n := f()
				if !m.HasCtx && strings.Contains(t, "Context") {
					n = "ctx"
					m.HasCtx = true
					m.Ctx = n
				}
				paramNames = append(paramNames, types.FuncParam{
					Name: n,
					Type: t,
				})

				continue
			}

			paramNames = append(paramNames, types.FuncParam{
				Name: name.String(),
				Type: t,
			})

			if !m.HasCtx && strings.Contains(t, "Context") {
				m.HasCtx = true
				m.Ctx = name.String()
			}
		}
	}

	// TODO This is the fist and simplest solution and probably need refactoring.
	// This is for handling a case where we have underscore(_) in params. since
	// we are calling another function we need to pass all parameters so we can't
	// have a parameter with underscore. we will replace everything we had as parameters
	// with the parameters we created either by generating new name or using old ones.
	m.MethodSigFull = strings.Replace(
		m.MethodSigFull,
		string(t.text[params.Pos():params.End()-2]),
		paramNames.Join(),
		1,
	)

	// This is used when calling the fucntion to make template simple.
	// wrapped.F({{ .MethodParamNames }}) => i.e. wrapped.F(a, b, c, d)
	m.MethodParamNames = paramNames.JoinNames()

	return paramNames
}

func (t *TypeVisitor) handleResults(m *types.Method, results *ast.FieldList, f func() string) types.FuncParams {
	var resultNames types.FuncParams

	if results == nil {
		return resultNames
	}

	for _, result := range results.List {
		t := string(t.text[result.Type.Pos()-1 : result.Type.End()-1])
		// TODO multipe error?
		// Assume we only return one error for now and name it err. Also
		// set HasError to true for template to add error handling.
		if t == "error" {
			m.HasError = true
			resultNames = append(resultNames, types.FuncParam{
				Name: "err",
				Type: t,
			})
			continue
		}

		// if we have unnamedd results (i.e. f(...) (int, string, error)),
		// generate a name by calling f(). This is then used in getting the
		// return value from calling wrapped function. a, b, err = wrapped.F(...)
		if result.Names == nil {
			resultNames = append(resultNames, types.FuncParam{
				Name: f(),
				Type: t,
			})
			continue
		}

		// If we reach here it means we have named results so set NamedResult.
		// Doing that will let us use = instead of := since we have no new variable
		// in the right part of the expression.
		m.NamedResults = true

		// if results are named, iterate over names and handle them
		// if we encouter an underscore(_), generate a name by calling f.
		for _, name := range result.Names {
			if name.String() == "_" {
				resultNames = append(resultNames, types.FuncParam{
					Name: f(),
					Type: t,
				})
				continue
			}
			resultNames = append(resultNames, types.FuncParam{
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
	if m.NamedResults {
		nStr = resultNames.Join()
		oStr = string(t.text[results.Pos() : results.End()-2])
	} else if results.Closing.IsValid() {
		nStr = resultNames.JoinTypes()
		oStr = string(t.text[results.Pos() : results.End()-2])
	} else {
		nStr = resultNames.JoinTypes()
		oStr = string(t.text[results.Pos()-1 : results.End()-1])
	}

	m.MethodSigFull = strings.Replace(
		m.MethodSigFull,
		oStr,
		nStr,
		1,
	)

	// This is used when getting results from the wrapped fucntion
	// to make template simple.
	// {{ .ResultNames }} = wrapped.F(...) => i.e. a, b, c, d := wrapped.F(...)
	// or
	// {{ .ResultNames }} := wrapped.F(...) => i.e. a, b, c, d = wrapped.F(...)
	m.ResultNames = resultNames.JoinNames()

	return resultNames
}
