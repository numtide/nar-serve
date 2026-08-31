package nar_test

import (
	"fmt"
	"testing"

	"github.com/numtide/nar-serve/pkg/nar"
	"github.com/stretchr/testify/assert"
)

// nolint:gochecknoglobals
var cases = []struct {
	path1    string
	path2    string
	expected bool
}{
	{
		path1:    "/foo",
		path2:    "/foo",
		expected: true,
	},
	{
		path1:    "/fooa",
		path2:    "/foob",
		expected: true,
	},
	{
		path1:    "/foob",
		path2:    "/fooa",
		expected: false,
	},
	{
		path1:    "/cmd/structlayout/main.go",
		path2:    "/cmd/structlayout-optimize",
		expected: true,
	},
	{
		path1:    "/cmd/structlayout-optimize",
		path2:    "/cmd/structlayout-ao/main.go",
		expected: false,
	},
	// The entry `structlayout` sorts before `structlayout-optimize`, so
	// everything under it comes first. `-` is below `/` as a byte, so a plain
	// string comparison gets this pair backwards.
	{
		path1:    "/cmd/structlayout-optimize",
		path2:    "/cmd/structlayout/main.go",
		expected: false,
	},
	{
		path1:    "/share/applications-extra/a",
		path2:    "/share/applications/a.desktop",
		expected: false,
	},
	{
		path1:    "/share/applications/a.desktop",
		path2:    "/share/applications-extra/a",
		expected: true,
	},
}

func TestLexicographicallyOrdered(t *testing.T) {
	for i, testCase := range cases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			result := nar.PathIsLexicographicallyOrdered(testCase.path1, testCase.path2)
			assert.Equal(t, result, testCase.expected)
		})
	}
}

// The ordering has to be a total order, or the reader will reject NARs that
// are in fact well formed. For every pair of distinct paths exactly one
// direction holds, and every path is ordered with respect to itself.
func TestLexicographicallyOrderedIsTotal(t *testing.T) {
	paths := []string{
		"/",
		"/bin",
		"/bin/hello",
		"/share",
		"/share/applications",
		"/share/applications/a.desktop",
		"/share/applications-extra",
		"/share/applications-extra/a",
		"/share/applications.d",
		"/share/doc",
	}

	for _, p1 := range paths {
		for _, p2 := range paths {
			forwards := nar.PathIsLexicographicallyOrdered(p1, p2)
			backwards := nar.PathIsLexicographicallyOrdered(p2, p1)

			if p1 == p2 {
				assert.True(t, forwards, "%q should be ordered with itself", p1)
				continue
			}

			assert.NotEqual(t, forwards, backwards,
				"exactly one of (%q, %q) and (%q, %q) should hold", p1, p2, p2, p1)
		}
	}
}

func BenchmarkLexicographicallyOrdered(b *testing.B) {
	for i, testCase := range cases {
		b.Run(fmt.Sprint(i), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				nar.PathIsLexicographicallyOrdered(testCase.path1, testCase.path2)
			}
		})
	}
}
