package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"

	"github.com/itzloop/promwrapgen/wrapper"

	_ "embed"
)

//go:embed templates/wrapper.gotmpl
var tmpl string

func main() {
	fmt.Println(os.Args)
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

	for _, ev := range []string{
		"GOARCH",
		"GOOS",
		"GOFILE",
		"GOLINE",
		"GOPACKAGE",
		"DOLLAR",
		"GOPATH",
		"GOBIN",
		"GOROOT",
		"GOMOD",
	} {
		fmt.Println("  ", ev, "=", os.Getenv(ev))
	}

	filePath := path.Join(cwd, os.Getenv("GOFILE"))
	log.Printf("generating prometheus wrapper for '%s'\n", filePath)

	generator, err := wrapper.NewWrapperGenerator(wrapper.GeneratorOpts{
		FormatImports: true,
		TemplateStr:   tmpl,
	})
	if err != nil {
		log.Fatalf("failed to create WrapperGenerator: %v\n", err)
	}

	visitor, err := wrapper.NewTypeVisitor(generator, wrapper.TypeVisitorOpts{
		CWD:      cwd,
		FileName: os.Getenv("GOFILE"),
		Targets:  []string{*target},
	})
	if err != nil {
		log.Fatalf("failed to create TypeVisitor: %v\n", err)
	}

	err = visitor.Walk()
	if err != nil {
		log.Fatalf("failed to walk over ast: %v\n", err)
	}
}
