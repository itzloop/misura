package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/itzloop/misura/config"
	"github.com/itzloop/misura/wrapper"

	"embed"
)

var (
	version  = "v0.0.9-34c7ef6"
	commit   = "34c7ef6f97e3c5a9a09b5f7a72b0e36d35f8ac56"
	builtBy  = "golang"
	date     = "2024-06-16 23:17:21+00:00"
	progDesc = "misura (Italian for measure) gives insight about a golang type by generating a wrapper"
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
	cfg := config.NewConfig(os.Args[0])

	// error will be handled by flag.ExitOnError
	cfg.Parse(os.Args[1:])

	if *cfg.ShowVersion {
		fmt.Print(versionString())
		os.Exit(0)
	}

	if len(*cfg.Measures) == 0 {
		*cfg.Measures = append(*cfg.Measures, "all")
	}

	if os.Getenv("GOFILE") != "" {
		*cfg.FilePath = os.Getenv("GOFILE")
		fmt.Println("using GOFILE=", *cfg.FilePath)
	}

	fmt.Printf("running command: %s\n", strings.Join(os.Args, " "))

	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	*cfg.FilePath = path.Join(cwd, *cfg.FilePath)
	fmt.Printf("generating wrapper for '%s'\n", *cfg.FilePath)

	generator, err := wrapper.NewWrapperGenerator(wrapper.GeneratorOpts{
		Metrics:       []string(*cfg.Measures),
		FormatImports: *cfg.FormatImports,
		Template:      templates(),
	})
	if err != nil {
		log.Fatalf("failed to create WrapperGenerator: %v\n", err)
	}

	// parse comments for //misura:<Type>
	cv, err := wrapper.NewCommentVisitor(*cfg.FilePath)
	if err != nil {
		log.Fatalf("failed to create CommentVisitor: %v\n", err)
	}

	err = cv.Walk()
	if err != nil {
		log.Fatalf("failed to walk over ast: %v\n", err)
	}

	visitor, err := wrapper.NewTypeVisitor(generator, wrapper.TypeVisitorOpts{
		FilePath: *cfg.FilePath,
		Targets:  append([]string(*cfg.Types), cv.Targets()...),
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
