//go:build windows
// +build windows

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

	_, err := walkDir(`C:\indir`, simpleNamer(`C:\outdir`), nil, 0, false)
	assert.Error(t, err)

	_ = fs.MkdirAll(`C:\indir\one`, 0777)
	_ = fs.MkdirAll(`C:\indir\two`, 0777)
	afero.WriteFile(fs, `C:\indir\one\foo`, []byte("foo"), 0644)
	afero.WriteFile(fs, `C:\indir\one\bar`, []byte("bar"), 0644)
	afero.WriteFile(fs, `C:\indir\two\baz`, []byte("baz"), 0644)

	templates, err := walkDir(`C:\indir`, simpleNamer(`C:\outdir`), []string{`*\two`}, 0, false)

	assert.NoError(t, err)
	expected := []*Tplate{
		{
			Name:       `C:\indir\one\bar`,
			TargetPath: `C:\outdir\one\bar`,
			Mode:       0644,
		},
		{
			Name:       `C:\indir\one\foo`,
			TargetPath: `C:\outdir\one\foo`,
			Mode:       0644,
		},
	}
	assert.EqualValues(t, expected, templates)
}
