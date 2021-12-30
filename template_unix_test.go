//go:build !windows
// +build !windows

package gomplate

import (
	"testing"

	"github.com/spf13/afero"

	"github.com/stretchr/testify/assert"
)

func TestWalkDir(t *testing.T) {
	origfs := fs
	defer func() { fs = origfs }()
	fs = afero.NewMemMapFs()

	_, err := walkDir("/indir", simpleNamer("/outdir"), nil, 0, false)
	assert.Error(t, err)

	_ = fs.MkdirAll("/indir/one", 0777)
	_ = fs.MkdirAll("/indir/two", 0777)
	afero.WriteFile(fs, "/indir/one/foo", []byte("foo"), 0644)
	afero.WriteFile(fs, "/indir/one/bar", []byte("bar"), 0664)
	afero.WriteFile(fs, "/indir/two/baz", []byte("baz"), 0644)

	templates, err := walkDir("/indir", simpleNamer("/outdir"), []string{"*/two"}, 0, false)

	assert.NoError(t, err)
	expected := []*Tplate{
		{
			Name:       "/indir/one/bar",
			TargetPath: "/outdir/one/bar",
			Mode:       0664,
		},
		{
			Name:       "/indir/one/foo",
			TargetPath: "/outdir/one/foo",
			Mode:       0644,
		},
	}
	assert.EqualValues(t, expected, templates)
}
