package wrapper

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/require"
)

func TestWrapper(t *testing.T) {
	targets := []struct {
		filename string
		target   string
	}{
		{filename: "test.go", target: "NamedParamsAndResults"},
		{filename: "test.go", target: "UnnamedAndNamedParamsAndResults"},
		{filename: "test.go", target: "UnderscoreNames"},
		{filename: "test.go", target: "NoParams"},
		{filename: "test.go", target: "NoResult"},
		// {filename: "test.go", target: "ConflictDuration"},
		// {filename: "test.go", target: "ConflictTimePackage"},
	}

	for _, target := range targets {
		t.Run(
			fmt.Sprintf("test_generated_target_%s_compliation", target.target),
			func(t *testing.T) {
				wd := copyFilesHelper(t)
				tv := createTypeVisitor(t, wd, target.filename, []string{target.target})
				err := tv.Walk()
				require.NoError(t, err)
				require.FileExists(t, path.Join(wd, strings.ReplaceAll(target.filename, path.Ext(target.filename), ".misura.go")))
			},
		)
	}

	t.Run("all_targets_compliation", func(t *testing.T) {
        files := map[string][]string{}
		for _, t := range targets {
			_, ok := files[t.filename]
			if !ok {
				files[t.filename] = []string{}
			}

			files[t.filename] = append(files[t.filename], t.target)
		}

		for f, target := range files {
			wd := copyFilesHelper(t)
			tv := createTypeVisitor(t, wd, f, target)
			err := tv.Walk()
			require.NoError(t, err)
            for _, target := range target {
                require.FileExists(t, path.Join(wd, strings.ReplaceAll(f, path.Ext(f), "."+target+".misura.go")))
            }
		}

	})

}

func createTypeVisitor(t *testing.T, cwd, filename string, targets []string) *TypeVisitor {
	t.Helper()

	tv, err := NewTypeVisitor(createGenerator(t), TypeVisitorOpts{
		CWD:      cwd,
		FileName: filename,
		Targets:  targets,
	})

	require.NoError(t, err)
	require.NotNil(t, tv)

	return tv
}

func createGenerator(t *testing.T) *WrapperGenerator {
	t.Helper()
	tmpl, err := template.ParseGlob(path.Join("..", "templates", "*.gotmpl"))
	require.NoError(t, err)
	require.NotNil(t, tmpl)

	g, err := NewWrapperGenerator(GeneratorOpts{
		FormatImports: true,
		Template:      tmpl,
		Metrics:       []string{"all"},
	})

	require.NoError(t, err)
	require.NotNil(t, g)

	return g

}

func copyFilesHelper(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	require.NoError(t, err)

	src := path.Join(wd, "test_samples")
	dst := t.TempDir()
	err = copyDir(src, dst)
	require.NoError(t, err)

	return dst

}

func copyDir(src, dst string) (err error) {
	if err = isDir(src); err != nil {
		return err
	}

	if err = isDir(dst); err != nil {
		return err
	}

	fmt.Printf("copyDir: %s -> %s\n", src, dst)

	return filepath.Walk(src, func(p string, info fs.FileInfo, err error) error {
		fmt.Println(p, info.Name(), err, path.Base(src))
		if err != nil {
			return err
		}

		if info.IsDir() && info.Name() != path.Base(src) {
			dstDir := path.Join(dst, strings.TrimPrefix(p, src))
			return os.MkdirAll(dstDir, info.Mode())
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		to := path.Join(dst, strings.TrimPrefix(p, src))

		f, err := os.Open(p)
		if err != nil {
			return nil
		}
		defer f.Close()

		tof, err := os.Create(to)
		if err != nil {
			return err
		}
		defer tof.Close()

		if err = tof.Chmod(info.Mode()); err != nil {
			return err
		}

		_, err = io.Copy(tof, f)
		return err
	})
}

func isDir(p string) error {
	info, err := os.Stat(p)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return fmt.Errorf("'%s' is not a directory", p)
	}

	return nil
}
