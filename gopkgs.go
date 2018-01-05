package gopkgs

import (
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/MichaelTJones/walk"
)

// Pkg hold the information of the package.
type Pkg struct {
	Dir        string // directory containing package sources
	ImportPath string // import path of package in dir
	Name       string // package name
}

// Packages available to import.
func Packages() (map[string]*Pkg, error) {
	fset := token.NewFileSet()

	var pkgsMu sync.Mutex
	pkgs := make(map[string]*Pkg)

	for _, srcDir := range build.Default.SrcDirs() {
		err := walk.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			pathDir := filepath.Dir(path)
			if pathDir == srcDir {
				// Cannot put files on $GOPATH/src or $GOROOT/src.
				return nil
			}

			// Ignore files begin with "_", "." "_test.go" and directory named "testdata"
			// see: https://golang.org/cmd/go/#hdr-Description_of_package_lists

			name := info.Name()
			if info.IsDir() {
				if name[0] == '.' || name[0] == '_' || name == "testdata" || name == "node_modules" {
					return walk.SkipDir
				}
				return nil
			}

			if name[0] == '.' || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				return nil
			}

			filename := path
			src, err := parser.ParseFile(fset, filename, nil, parser.PackageClauseOnly)
			if err != nil {
				// skip unparseable go file
				return nil
			}

			pkgDir := pathDir
			pkgName := src.Name.Name
			if pkgName == "main" {
				// skip main package
				return nil
			}

			pkgsMu.Lock()
			if _, ok := pkgs[pkgDir]; !ok {
				pkgs[pkgDir] = &Pkg{
					Name:       pkgName,
					ImportPath: filepath.ToSlash(pkgDir[len(srcDir)+len("/"):]),
					Dir:        pkgDir,
				}
			}
			pkgsMu.Unlock()
			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	return pkgs, nil
}
