package wrapper

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/itzloop/promwrapgen/wrapper/types"
	"golang.org/x/tools/imports"
)

// TODO how can we go about using multiple template files
type GeneratorOpts struct {
	// FormatImports, if set to true, will be used
	// call imports.Process
	FormatImports bool

	// Template will be used to generate the wrapper
	// If this is null TemplateStr or TemplatePath
	// will be used.
	Template *template.Template

	// TemplateStr will be used to generate the wrapper
	// If this is null Template or TemplatePath
	// will be used.
	TemplateStr string

	// TemplatePath will be used to generate the wrapper
	// The code will first read the contents. If this is
	// null Template or TemplateStr will be used.
	// TODO can this be a directory?
	TemplatePath string

	// Suffix will be used to name the generated wrapper.
	// GOFILE.Suffix.go. By default promwrapgen will be used.j
	Suffix string
}

type TemplateVals struct {
	PackageName     string
	WrapperTypeName string
	MethodList      []types.Method
	Imports         string
	StartTimeName   string
	DurationName    string
	RandomHex       string
}

type WrapperGenerator struct {
	tmpl *template.Template
	opts GeneratorOpts
}

func MustNewWrapperGenerator(w *WrapperGenerator, err error) *WrapperGenerator {
	if err != nil {
		log.Fatalln("failed to create WrapperGenerator:", err)
	}

	return w
}

func NewWrapperGenerator(opts GeneratorOpts) (*WrapperGenerator, error) {
	var (
		w   = WrapperGenerator{opts: opts}
		err error
	)

	if w.opts.Template != nil {
		w.tmpl = w.opts.Template
	} else if strings.TrimSpace(w.opts.TemplateStr) != "" {
		w.tmpl, err = template.New("wrapper.gotmpl").Parse(string(w.opts.TemplateStr))
		if err != nil {
			return nil, err
		}

	} else if strings.TrimSpace(w.opts.TemplatePath) != "" {
		w.tmpl, err = template.New("wrapper.gotmpl").ParseFiles(w.opts.TemplatePath)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("no template provided")
	}

	if strings.TrimSpace(w.opts.Suffix) == "" {
		w.opts.Suffix = "promwrapgen"
	}

	return &w, nil
}

func (w *WrapperGenerator) Generate(outPath, filename string, tmplVals TemplateVals) error {
	var (
		b                = &bytes.Buffer{}
		processed        []byte
		filenameSuffixed = fmt.Sprintf("%s.%s.go", strings.Replace(filename, ".go", "", 1), w.opts.Suffix)
		p                = path.Join(outPath, filenameSuffixed)
	)

	tmp, err := os.Create(p)
	if err != nil {
		panic(err)
	}
	defer tmp.Close()

    if err = w.tmpl.Execute(b, tmplVals); err != nil {
        return err
    }

	if w.opts.FormatImports {
		processed, err = imports.Process(p, b.Bytes(), nil)
		if err != nil {
			return err
		}
	} else {
		processed = b.Bytes()
	}

	fmt.Printf("writing to %s\n", tmp.Name())
	_, err = tmp.Write(processed)
	if err != nil {
		return err
	}

	return nil
}
