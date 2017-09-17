package gopkgs_test

import "testing"
import "github.com/uudashr/gopkgs"

func BenchmarkPackages(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := gopkgs.Packages(); err != nil {
			b.Fatal("err:", err)
		}
	}
}
