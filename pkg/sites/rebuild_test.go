package sites

import (
	"testing"

	"github.com/jschaf/jsc/pkg/dirs"
)

func BenchmarkRebuild(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if err := Rebuild(dirs.Dist); err != nil {
			b.Fatal(err)
		}
	}
}
