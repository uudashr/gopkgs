package gopkgs

import (
	v2 "github.com/uudashr/gopkgs/v2"
)

// Pkg hold the information of the package.
type Pkg v2.Pkg

// Options for retrieve packages.
type Options v2.Options

// List packages on workDir.
// workDir is required for module mode. If the workDir is not under module, then it will fallback to GOPATH mode.
func List(opts Options) (map[string]Pkg, error) {
	result, err := v2.List(v2.Options(opts))
	if err != nil {
		return nil, err
	}

	pkgs := make(map[string]Pkg, len(result))
	for key, pkg := range result {
		pkgs[key] = Pkg(pkg)
	}
	return pkgs, nil
}
