package gopkgs

import (
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// Pkg hold the information of the package.
type Pkg struct {
	Dir        string // directory containing package sources
	ImportPath string // import path of package in dir
	Name       string // package name
}

// Packages return the go packages.
func Packages() (map[string]*Pkg, error) {
	fset := token.NewFileSet()
	pkgs := make(map[string]*Pkg)
	badPkgs := make(map[string]bool)
	for _, dir := range build.Default.SrcDirs() {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Ignore files begin with "_", "." "_test.go" and directory named "testdata"
			// see: https://golang.org/cmd/go/#hdr-Description_of_package_lists

			if strings.HasPrefix(info.Name(), ".") {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			if strings.HasPrefix(info.Name(), "_") {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			if info.IsDir() && info.Name() == "testdata" {
				return filepath.SkipDir
			}

			if strings.HasSuffix(info.Name(), "_test.go") {
				return nil
			}

			if !strings.HasSuffix(info.Name(), ".go") {
				return nil
			}

			filename := path
			src, err := parser.ParseFile(fset, filename, nil, parser.PackageClauseOnly)
			if err != nil {
				// skip unparseable go file
				return nil
			}

			pkgDir := filepath.Dir(filename)
			pkgName := src.Name.Name
			if pkgName == "main" {
				// skip main package
				return nil
			}

			pkgPath := filepath.ToSlash(pkgDir[len(dir)+len("/"):])

			pkg, found := pkgs[pkgDir]
			if found {
				if pkg.Name != pkgName {
					badPkgs[pkgDir] = true
				}
				// we've done with this package
				return nil
			}

			pkg = &Pkg{
				Name:       pkgName,
				ImportPath: pkgPath,
				Dir:        pkgDir,
			}
			pkgs[pkgDir] = pkg
			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	return pkgs, nil
}
