package internal // import "github.com/uudashr/gopkgs/v2/internal"

import "strings"

func visibleVendor(workDir, vendorDir string) bool {
	return strings.Index(workDir, vendorDir) == 0
}
