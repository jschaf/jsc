package compiler

import (
	"testing"

	"github.com/jschaf/jsc/pkg/dirs"
)

func BenchmarkNewDetailCompiler_Compile(b *testing.B) {
	b.StopTimer()
	c := NewDetailCompiler(dirs.Dist)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if err := c.Compile("procella"); err != nil {
			b.Fatal(err)
		}
	}
}
