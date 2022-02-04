//go:build windows
// +build windows

package gomplate

import (
	"bytes"
	"context"
	"fmt"
	"github.com/hairyhenderson/gomplate/v3/data"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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
	expected := []*tplate{
		{
			name:       `C:\indir\one\bar`,
			targetPath: `C:\outdir\one\bar`,
			mode:       0644,
		},
		{
			name:       `C:\indir\one\foo`,
			targetPath: `C:\outdir\one\foo`,
			mode:       0644,
		},
	}
	assert.EqualValues(t, expected, templates)
}

func TestRunningTemplate(t *testing.T) {

	tDir, err := os.MkdirTemp("", "")
	t.Cleanup(func() { _ = os.RemoveAll(tDir) })
	assert.NoError(t, err)

	_ = ioutil.WriteFile(filepath.Join(tDir, `out.json`), []byte(`["banana","pear","apple"]`), 0644)

	template := `
{{ range (datasource "fruit") -}}
{{ . }}
{{ end -}}
`

	fruitURL, err := url.Parse(fmt.Sprintf(`file:///%s\out.json`, tDir))
	ctx := context.Background()
	assert.NoError(t, err)
	d := &data.Data{
		Ctx: ctx,
		Sources: map[string]*data.Source{"fruit": {
			URL:   fruitURL,
			Alias: "fruit",
		}},
	}
	funcMap := CreateFuncs(ctx, d)
	g := &gomplate{
		tmplctx:         nil,
		funcMap:         funcMap,
		nestedTemplates: nil,
		rootTemplate:    nil,
		leftDelim:       "{{",
		rightDelim:      "}}",
	}
	var out bytes.Buffer
	err = g.runTemplate(context.Background(), &tplate{
		name:     "testtemplate",
		contents: template,
		target:   &out,
	})
	str := out.String()
	assert.True(t, strings.Contains(str, "pear"))
	assert.True(t, strings.Contains(str, "apple"))
	assert.True(t, strings.Contains(str, "banana"))
	assert.NoError(t, err)

}
