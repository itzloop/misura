package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/itzloop/promwrapgen/wrapper"

	"embed"
)

//go:embed templates/*.gotmpl
var f embed.FS

func main() {
	fmt.Printf("running command: %s\n", strings.Join(os.Args, " "))

	// should accept multipe targets
	targetsFlag := flag.String("t", "", "comma seperated target interface(s)")
	metricsflag := flag.String("m", "all", `comma seperated list of metrics to include. 
possible values [all, duration, total, success, error].
If all is specified others will be ignored`)
	formatImports := flag.Bool("fmt", true, "if set to true, will run imports.Process on the generated wrapper")
	flag.Parse()

	targetsCS := strings.Split(*targetsFlag, ",")
	if len(targetsCS) == 0 {
		log.Fatalln("at least one target is needed")
	}

	metricsCS := strings.Split(*metricsflag, ",")
	if len(metricsCS) == 0 {
		metricsCS = append(metricsCS, "all")
	}

	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	filePath := path.Join(cwd, os.Getenv("GOFILE"))
	fmt.Printf("generating prometheus wrapper for '%s'\n", filePath)

	generator, err := wrapper.NewWrapperGenerator(wrapper.GeneratorOpts{
		Metrics:       metricsCS,
		FormatImports: *formatImports,
		Template:      templates(),
	})
	if err != nil {
		log.Fatalf("failed to create WrapperGenerator: %v\n", err)
	}

	visitor, err := wrapper.NewTypeVisitor(generator, wrapper.TypeVisitorOpts{
		CWD:      cwd,
		FileName: os.Getenv("GOFILE"),
		Targets:  targetsCS,
	})
	if err != nil {
		log.Fatalf("failed to create TypeVisitor: %v\n", err)
	}

	err = visitor.Walk()
	if err != nil {
		log.Fatalf("failed to walk over ast: %v\n", err)
	}
}

func templates() *template.Template {
	tmpl := template.New("wrapper")
	return template.Must(tmpl.ParseFS(f, "templates/*.gotmpl"))
}
