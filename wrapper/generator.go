package wrapper

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/itzloop/misura/wrapper/types"
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
	// GOFILE.Suffix.go. By default misura will be used.j
	Suffix string

	// Metrics will be used to decided what metrics to include.
	// Possible values are:
	// 1. duration
	// 2. total
	// 3. error
	// 4. success
	// 5. all
	// If all is specified then others will be ignored.
	Metrics types.Strings
}

type TemplateVals struct {
	PackageName     string
	WrapperTypeName string
	MethodList      []types.Method
	Imports         string
	StartTimeName   string
	DurationName    string
	RandomHex       string

	// metrics
	HasDuration bool
	HasTotal    bool
	HasError    bool
	HasSuccess  bool
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
		w.opts.Suffix = "misura"
	}

	return &w, nil
}

func (w *WrapperGenerator) Generate(outPath, filename string, tmplVals TemplateVals) error {
	var (
		b                = &bytes.Buffer{}
		processed        []byte
		filenameSuffixed = fmt.Sprintf("%s.%s.go", strings.Replace(filename, ".go", "", 1), w.opts.Suffix)
		p                = path.Join(outPath, filenameSuffixed)
		err              error
	)

	fmt.Println(w.opts.Metrics)
	if len(w.opts.Metrics) == 0 || w.opts.Metrics.Exists("all") {
		tmplVals.HasDuration = true
		tmplVals.HasTotal = true
		tmplVals.HasError = true
		tmplVals.HasSuccess = true
	} else {
		if w.opts.Metrics.Exists("duration") {
			tmplVals.HasDuration = true
		}

		if w.opts.Metrics.Exists("total") {
			tmplVals.HasTotal = true
		}

		if w.opts.Metrics.Exists("error") {
			tmplVals.HasError = true
		}

		if w.opts.Metrics.Exists("success") {
			tmplVals.HasSuccess = true
		}
	}

	tmplVals.RandomHex, err = getRandomHex(p, tmplVals.RandomHex)
	if err != nil {
		return err
	}

	if err = w.tmpl.ExecuteTemplate(b, "wrapper.gotmpl", tmplVals); err != nil {
		return err
	}

	processed, err = formatImports(p, b, w.opts.FormatImports)
	if err != nil {
		return err
	}

	f, err := os.Create(p)
	if err != nil {
		return err
	}

	fmt.Printf("writing to %s\n", f.Name())
	_, err = f.Write(processed)
	if err != nil {
		return err
	}

	return nil
}

func formatImports(filename string, b *bytes.Buffer, format bool) ([]byte, error) {
	if format {
		processed, err := imports.Process(filename, b.Bytes(), nil)
		if err != nil {
			return nil, err
		}

		return processed, nil
	}

	return b.Bytes(), nil

}

func getRandomHex(p, randomHex string) (string, error) {
	f, err := os.OpenFile(p, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return "", err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)

	if sc.Scan() && sc.Scan() {
		newRandomHex := strings.Replace(sc.Text(), "// RANDOM_HEX=", "", 1)
		if strings.TrimSpace(newRandomHex) != "" {
			fmt.Printf("reusing random hex: %s\n", newRandomHex)
			return newRandomHex, nil
		}
	}

	return randomHex, nil
}
