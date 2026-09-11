package ls_test

import (
	"strings"
	"testing"

	"github.com/numtide/nar-serve/pkg/nar"
	"github.com/numtide/nar-serve/pkg/nar/ls"
	"github.com/stretchr/testify/assert"
)

const fixture = `
{
  "version": 1,
  "root": {
    "type": "directory",
    "entries": {
      "bin": {
        "type": "directory",
        "entries": {
          "curl": {
            "type": "regular",
            "size": 182520,
            "executable": true,
            "narOffset": 400
          }
        }
      }
    }
  }
}
`

func TestLS(t *testing.T) {
	r := strings.NewReader(fixture)
	root, err := ls.ParseLS(r)
	assert.NoError(t, err)

	expectedRoot := &ls.Root{
		Version: 1,
		Root: ls.Node{
			Type: nar.TypeDirectory,
			Entries: map[string]*ls.Node{
				"bin": {
					Type: nar.TypeDirectory,
					Entries: map[string]*ls.Node{
						"curl": {
							Type:       nar.TypeRegular,
							Size:       182520,
							Executable: true,
							NAROffset:  400,
						},
					},
				},
			},
		},
	}
	assert.Equal(t, expectedRoot, root)
}

func TestLookup(t *testing.T) {
	root, err := ls.ParseLS(strings.NewReader(fixture))
	assert.NoError(t, err)

	assert.Equal(t, &root.Root, root.Lookup("/"))
	assert.Equal(t, nar.TypeDirectory, root.Lookup("/bin").Type)
	assert.Equal(t, nar.TypeDirectory, root.Lookup("/bin/").Type)
	assert.Equal(t, int64(182520), root.Lookup("/bin/curl").Size)
	assert.Equal(t, int64(182520), root.Lookup("bin/curl").Size)
	assert.Nil(t, root.Lookup("/bin/absent"))
	assert.Nil(t, root.Lookup("/bin/curl/deeper"))
	assert.Nil(t, root.Lookup("/absent/curl"))
}
