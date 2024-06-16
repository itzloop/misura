
package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/itzloop/misura/wrapper"

	"embed"
)

var (
	version  = "v0.0.7-754505d"
	commit   = "754505de44b1af62718c498ade0df829dc51304a"
	builtBy  = "golang"
	date     = "2024-06-16 17:55:38+00:00"
	progDesc = "misura (Italian for measure) gives isight about a golang type by generating a wrapper for that type"
	website  = "https://sinashabani.dev"
)

//go:embed templates/*.gotmpl
var f embed.FS

//go:embed ascii.txt
var asciiArt string

func versionString() string {
	t := template.Must(template.ParseFS(f, "templates/version.gotmpl"))
	buf := bytes.Buffer{}

	t.Execute(&buf, map[string]string{
		"ASCII":     asciiArt,
		"ProgDesc":  progDesc,
		"Website":   website,
		"ProgVer":   version,
		"GitCommit": commit,
		"BuildDate": date,
		"BuiltBy":   builtBy,
	})

	return buf.String()
}


func main() {
    var (
        showVersion bool
    )

	// should accept multipe targets
	targetsFlag := flag.String("t", "", "comma seperated target interface(s)")
	metricsflag := flag.String("m", "all", `comma seperated list of metrics to include. 
possible values [all, duration, total, success, error].
If all is specified others will be ignored`)
	formatImports := flag.Bool("fmt", true, "if set to true, will run imports.Process on the generated wrapper")
    flag.BoolVar(&showVersion, "version", false, "show program version")
    flag.BoolVar(&showVersion, "v", false, "show program version")
	flag.Parse()

    if showVersion {
        fmt.Print(versionString())
        os.Exit(0)
    }

	fmt.Printf("running command: %s\n", strings.Join(os.Args, " "))
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
