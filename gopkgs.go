package gopkgs

import (
	"bufio"
	"bytes"
	"go/build"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/karrick/godirwalk"
	"github.com/pkg/errors"
)

const (
	goflagsEnv    = "GOFLAGS"
	modEmptyFlag  = "-mod="
	modVendorFlag = "-mod=vendor"
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
	path string
	dir  string
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
					return "", errors.Errorf("expect pattern 'package <name>':%s", line)
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

func listFiles(srcDir, workDir string, noVendor bool) (<-chan goFile, <-chan error) {
	filec := make(chan goFile, 10000)
	errc := make(chan error, 1)

	go func() {
		defer func() {
			close(filec)
			close(errc)
		}()

		if workDir != "" && !filepath.IsAbs(workDir) {
			wd, err := filepath.Abs(workDir)
			if err != nil {
				errc <- err
				return
			}

			workDir = wd
		}

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

						if noVendor {
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
					path: osPathname,
					dir:  pathDir,
				}
				return nil
			},
			ErrorCallback: func(s string, err error) godirwalk.ErrorAction {
				err = errors.Cause(err)
				if v, ok := err.(*os.PathError); ok && (os.IsNotExist(v.Err) || os.IsPermission(v.Err)) {
					return godirwalk.SkipNode
				}

				return godirwalk.Halt
			},
		})

		if err != nil {
			errc <- err
			return
		}
	}()
	return filec, errc
}

func listModFiles(modDir string) (<-chan goFile, <-chan error) {
	filec := make(chan goFile, 10000)
	errc := make(chan error, 1)

	go func() {
		defer func() {
			close(filec)
			close(errc)
		}()

		err := godirwalk.Walk(modDir, &godirwalk.Options{
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

					return nil
				}

				if name[0] == '.' || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
					return nil
				}

				filec <- goFile{
					path: osPathname,
					dir:  pathDir,
				}
				return nil
			},
			ErrorCallback: func(s string, err error) godirwalk.ErrorAction {
				err = errors.Cause(err)
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
	}()
	return filec, errc
}

func collectPkgs(srcDir, workDir string, noVendor bool, out map[string]Pkg) error {
	filec, errc := listFiles(srcDir, workDir, noVendor)
	for f := range filec {
		pkgDir := f.dir
		if _, found := out[pkgDir]; found {
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

		out[pkgDir] = Pkg{
			Name:       pkgName,
			ImportPath: filepath.ToSlash(pkgDir[len(srcDir)+len("/"):]),
			Dir:        pkgDir,
		}
	}

	if err := <-errc; err != nil {
		return err
	}

	return nil
}

func collectModPkgs(m mod, vendorMode bool, out map[string]Pkg) error {
	// choose proper directory for search
	dir := m.pkgDir
	if vendorMode {
		var err error
		if dir, err = pickDir(m.pkgDir, m.vendorDir); err != nil {
			return errors.Wrap(err, "unable to list mod files")
		}
	}

	filec, errc := listModFiles(dir)
	for f := range filec {
		pkgDir := f.dir
		if _, found := out[pkgDir]; found {
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

		// debug := true
		importPath := m.path
		if pkgDir != dir {
			// remove prefix if pkg is vendored
			if vendorMode && strings.HasPrefix(pkgDir, dir+"/vendor") {
				importPath = strings.TrimPrefix(pkgDir, dir+"/vendor/")
			} else {
				importPath += filepath.ToSlash(pkgDir[len(dir):])
			}
		}

		out[pkgDir] = Pkg{
			Name:       pkgName,
			ImportPath: importPath,
			Dir:        pkgDir,
		}
	}

	if err := <-errc; err != nil {
		return err
	}

	return nil
}

// List packages on workDir.
// workDir is required for module mode. If the workDir is not under module, then it will fallback to GOPATH mode.
func List(opts Options) (map[string]Pkg, error) {
	pkgs := make(map[string]Pkg)

	if opts.WorkDir == "" {
		// force on GOPATH mode
		for _, srcDir := range build.Default.SrcDirs() {
			err := collectPkgs(srcDir, opts.WorkDir, opts.NoVendor, pkgs)
			if err != nil {
				return nil, err
			}
		}
		return pkgs, nil
	}

	vendorMode := checkVendorMode()
	mods, err := listMods(opts.WorkDir, vendorMode)
	if err != nil {
		// GOPATH mode
		for _, srcDir := range build.Default.SrcDirs() {
			err = collectPkgs(srcDir, opts.WorkDir, opts.NoVendor, pkgs)
			if err != nil {
				return nil, err
			}
		}
		return pkgs, nil
	}

	// Module mode
	if err = collectPkgs(filepath.Join(build.Default.GOROOT, "src"), opts.WorkDir, false, pkgs); err != nil {
		return nil, err
	}

	for _, m := range mods {
		err = collectModPkgs(m, vendorMode, pkgs)
		if err != nil {
			return nil, err
		}
	}

	return pkgs, nil
}

type mod struct {
	path      string
	pkgDir    string
	vendorDir string
}

func listMods(workDir string, vendorMode bool) ([]mod, error) {
	// exec `go list -m ...` to get module path list
	// if GOFLAGS contains '-mod=vendor', it gets vendor path list instead
	s, err := execGoList(workDir, "list", "-m", "-f={{.Path}};{{.Dir}}", "all")
	if err != nil {
		return nil, errors.Wrap(err, "unable to execute `go list -m ...`")
	}

	var mods []mod
	for s.Scan() {
		line := s.Text()
		ls := strings.Split(line, ";")
		if vendorMode {
			mods = append(mods, mod{path: ls[0], vendorDir: ls[1]})
		} else {
			mods = append(mods, mod{path: ls[0], pkgDir: ls[1]})
		}
	}

	// if GOFLAGS contains '-mod=vendor', exec `go list -mod= -m ...` to fill the module paths as well
	if vendorMode {
		s, err = execGoList(workDir, "list", modEmptyFlag, "-m", "-f={{.Path}};{{.Dir}}", "all")
		if err != nil {
			return nil, errors.Wrap(err, "unable to execute `go list -mod= -m ...`")
		}

		keyIndexMap := make(map[string]int, len(mods))
		for i, m := range mods {
			keyIndexMap[m.path] = i
		}
		for s.Scan() {
			line := s.Text()
			ls := strings.Split(line, ";")
			if _, ok := keyIndexMap[ls[0]]; ok {
				mods[keyIndexMap[ls[0]]].pkgDir = ls[1]
			}
		}
	}

	return mods, nil
}

func execGoList(workDir string, cmdArgs ...string) (*bufio.Scanner, error) {
	cmd := exec.Command("go", cmdArgs...)
	cmd.Dir = workDir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return bufio.NewScanner(bytes.NewReader(out)), nil
}

func checkVendorMode() bool {
	return strings.Contains(os.Getenv(goflagsEnv), modVendorFlag)
}

// we prefer vendorDir to pkgDir if it exists
func pickDir(pkgDir, vendorDir string) (string, error) {
	if _, err := os.Stat(vendorDir); !os.IsNotExist(err) {
		return vendorDir, nil
	}
	if _, err := os.Stat(pkgDir); !os.IsNotExist(err) {
		return pkgDir, nil
	}
	return "", os.ErrNotExist
}
