package gopkgs

import (
	v1 "github.com/uudashr/gopkgs"
)

// Pkg hold the information of the package.
type Pkg v1.Pkg

// Options for retrieve packages.
type Options v1.Options

// List packages on workDir.
func List(opts Options) (map[string]Pkg, error) {
	result, err := v1.List(v1.Options(opts))
	if err != nil {
		return nil, err
	}

	pkgs := make(map[string]Pkg, len(result))
	for key, pkg := range result {
		pkgs[key] = Pkg(pkg)
	}
	return pkgs, nil
}
