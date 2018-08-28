package gopkgs

import (
	"bufio"
	"errors"
	"go/build"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/karrick/godirwalk"
	pkgerrors "github.com/pkg/errors"
)

// Pkg hold the information of the package.
type Pkg struct {
	Dir        string // directory containing package sources
	ImportPath string // import path of package in dir
	Name       string // package name
}

// Options for retrieve packages.
type Options struct {
	WorkDir  string // Will return importable package under WorkDir. Any vendor dependencies outside the WorkDir will be ignored.
	NoVendor bool   // Will not retrieve vendor dependencies, except inside WorkDir (if specified)
}

type goFile struct {
	path   string
	dir    string
	srcDir string
}

func mustClose(c io.Closer) {
	if err := c.Close(); err != nil {
		panic(err)
	}
}

func readPackageName(filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}

	s := bufio.NewScanner(f)
	var inComment bool
	for s.Scan() {
		line := strings.TrimSpace(s.Text())

		if line == "" {
			continue
		}

		if !inComment {
			if strings.HasPrefix(line, "/*") {
				inComment = true
				continue
			}

			if strings.HasPrefix(line, "//") {
				// skip inline comment
				continue
			}

			if strings.HasPrefix(line, "package") {
				ls := strings.Split(line, " ")
				if len(ls) < 2 {
					mustClose(f)
					return "", errors.New("expect pattern 'package <name>':" + line)
				}

				mustClose(f)
				return ls[1], nil
			}

			// package should be found first
			mustClose(f)
			return "", errors.New("invalid go file, expect package declaration")
		}

		// inComment = true
		if strings.HasSuffix(line, "*/") {
			inComment = false
		}
	}

	mustClose(f)
	return "", errors.New("cannot find package information")
}

// Packages available to import.
func Packages(opts Options) (map[string]Pkg, error) {
	pkgs := make(map[string]Pkg)

	filec, errc := listFiles(opts)
	for f := range filec {
		pkgDir := f.dir
		if _, found := pkgs[pkgDir]; found {
			// already have this package, skip
			continue
		}

		pkgName, err := readPackageName(f.path)
		if err != nil {
			// skip unparseable file
			continue
		}

		if pkgName == "main" {
			// skip main package
			continue
		}

		pkgs[pkgDir] = Pkg{
			Name:       pkgName,
			ImportPath: getImportPath(pkgDir, f.srcDir, opts.WorkDir),
			Dir:        pkgDir,
		}
	}

	if err := <-errc; err != nil {
		return nil, err
	}

	return pkgs, nil
}

func getImportPath(pkgDir, srcDir, workDir string) string {
	pathPrefix := pkgDir[len(srcDir)+len("/"):]
	if srcDir == workDir {
		goModPath := srcDir + "/go.mod"
		file, err := os.Open(goModPath)
		if err == nil {
			defer file.Close()

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "module") {
					pathPrefix = strings.TrimLeft(line, "module ") + "/" + pathPrefix
				}
			}
		}

	}
	return filepath.ToSlash(pathPrefix)
}

func listFiles(opts Options) (<-chan goFile, <-chan error) {
	filec := make(chan goFile, 10000)
	errc := make(chan error, 1)

	go func() {
		defer func() {
			close(filec)
			close(errc)
		}()

		srcDirs := build.Default.SrcDirs()

		workDir := opts.WorkDir
		if workDir != "" {
			if !filepath.IsAbs(workDir) {
				wd, err := filepath.Abs(workDir)
				if err != nil {
					errc <- err
					return
				}

				workDir = wd
			}

			goModFile := strings.TrimRight(workDir, "/") + "/" + "go.mod"
			if _, err := os.Stat(goModFile); err != nil {
				srcDirs = append(srcDirs, workDir)
			}
		}

		for _, srcDir := range srcDirs {
			err := godirwalk.Walk(srcDir, &godirwalk.Options{
				FollowSymbolicLinks: true,
				Callback: func(osPathname string, de *godirwalk.Dirent) error {
					name := de.Name()
					pathDir := filepath.Dir(osPathname)

					// Symlink not supported by go
					if de.IsSymlink() {
						return filepath.SkipDir
					}

					// Ignore files begin with "_", "." "_test.go" and directory named "testdata"
					// see: https://golang.org/cmd/go/#hdr-Description_of_package_lists

					if de.IsDir() {
						if name[0] == '.' || name[0] == '_' || name == "testdata" || name == "node_modules" {
							return filepath.SkipDir
						}

						if name == "vendor" {
							if workDir != "" {
								if !visibleVendor(workDir, pathDir) {
									return filepath.SkipDir
								}

								return nil
							}

							if opts.NoVendor {
								return filepath.SkipDir
							}
						}

						return nil
					}

					if name[0] == '.' || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
						return nil
					}

					if pathDir == srcDir {
						// Cannot put files on $GOPATH/src or $GOROOT/src.
						return nil
					}

					filec <- goFile{
						path:   osPathname,
						dir:    pathDir,
						srcDir: srcDir,
					}
					return nil
				},
				ErrorCallback: func(s string, err error) godirwalk.ErrorAction {
					err = pkgerrors.Cause(err)
					if v, ok := err.(*os.PathError); ok && os.IsNotExist(v.Err) {
						return godirwalk.SkipNode
					}

					return godirwalk.Halt
				},
			})

			if err != nil {
				errc <- err
				return
			}
		}
	}()
	return filec, errc
}
