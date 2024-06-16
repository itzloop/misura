package wrapper

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"strings"
)

type CommentVisitor struct {
	targets []string
	p       string
	text    []byte
	err     error
}

func NewCommentVisitor(p string) (*CommentVisitor, error) {
	f, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}

	return &CommentVisitor{
		p:    p,
		text: f,
	}, nil
}

func (cv *CommentVisitor) Walk() error {
	fset := token.NewFileSet()
	root, err := parser.ParseFile(fset, path.Base(cv.p), cv.text, parser.ParseComments)
	if err != nil {
		return err
	}

	ast.Walk(cv, root)
	return cv.err
}

func (cv *CommentVisitor) Visit(nRaw ast.Node) ast.Visitor {
	if nRaw == nil {
		return nil
	}

	switch n := nRaw.(type) {
	case *ast.Comment:
		if !strings.Contains(n.Text, "//misura:") {
			return cv
		}

		cv.targets = append(cv.targets, strings.Replace(n.Text, "//misura:", "", 1))
	}

	return cv
}

func (cv *CommentVisitor) Targets() []string {
	return cv.targets
}
