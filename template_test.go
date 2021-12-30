package gomplate

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/hairyhenderson/gomplate/v3/internal/config"
	"github.com/hairyhenderson/gomplate/v3/internal/iohelpers"
	"github.com/spf13/afero"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenOutFile(t *testing.T) {
	origfs := fs
	defer func() { fs = origfs }()
	fs = afero.NewMemMapFs()
	_ = fs.Mkdir("/tmp", 0777)

	cfg := &config.Config{}
	f, err := openOutFile(cfg, "/tmp/foo", 0644, false)
	assert.NoError(t, err)

	wc, ok := f.(io.WriteCloser)
	assert.True(t, ok)
	err = wc.Close()
	assert.NoError(t, err)

	i, err := fs.Stat("/tmp/foo")
	assert.NoError(t, err)
	assert.Equal(t, iohelpers.NormalizeFileMode(0644), i.Mode())

	cfg.Stdout = &bytes.Buffer{}

	f, err = openOutFile(cfg, "-", 0644, false)
	assert.NoError(t, err)
	assert.Equal(t, cfg.Stdout, f)
}

func TestLoadContents(t *testing.T) {
	origfs := fs
	defer func() { fs = origfs }()
	fs = afero.NewMemMapFs()

	afero.WriteFile(fs, "foo", []byte("Contents"), 0644)

	tmpl := &Tplate{Name: "foo"}
	b, err := tmpl.loadContents(nil)
	assert.NoError(t, err)
	assert.Equal(t, "Contents", string(b))
}

func TestGatherTemplates(t *testing.T) {
	origfs := fs
	defer func() { fs = origfs }()
	fs = afero.NewMemMapFs()
	afero.WriteFile(fs, "foo", []byte("bar"), 0600)

	afero.WriteFile(fs, "in/1", []byte("foo"), 0644)
	afero.WriteFile(fs, "in/2", []byte("bar"), 0644)
	afero.WriteFile(fs, "in/3", []byte("baz"), 0644)

	cfg := &config.Config{
		Stdin:  &bytes.Buffer{},
		Stdout: &bytes.Buffer{},
	}
	cfg.ApplyDefaults()
	templates, err := gatherTemplates(cfg, nil)
	assert.NoError(t, err)
	assert.Len(t, templates, 1)

	cfg = &config.Config{
		Input:  "foo",
		Stdout: &bytes.Buffer{},
	}
	cfg.ApplyDefaults()
	templates, err = gatherTemplates(cfg, nil)
	assert.NoError(t, err)
	assert.Len(t, templates, 1)
	assert.Equal(t, "foo", templates[0].Contents)
	assert.Equal(t, cfg.Stdout, templates[0].Target)

	templates, err = gatherTemplates(&config.Config{
		Input:       "foo",
		OutputFiles: []string{"out"},
	}, nil)
	assert.NoError(t, err)
	assert.Len(t, templates, 1)
	assert.Equal(t, "out", templates[0].TargetPath)
	assert.Equal(t, iohelpers.NormalizeFileMode(0644), templates[0].Mode)

	// out file is created only on demand
	_, err = fs.Stat("out")
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))

	_, err = templates[0].Target.Write([]byte("hello world"))
	assert.NoError(t, err)

	info, err := fs.Stat("out")
	require.NoError(t, err)
	assert.Equal(t, iohelpers.NormalizeFileMode(0644), info.Mode())
	fs.Remove("out")

	cfg = &config.Config{
		InputFiles:  []string{"foo"},
		OutputFiles: []string{"out"},
		Stdout:      &bytes.Buffer{},
	}
	templates, err = gatherTemplates(cfg, nil)
	assert.NoError(t, err)
	assert.Len(t, templates, 1)
	assert.Equal(t, "bar", templates[0].Contents)
	assert.NotEqual(t, cfg.Stdout, templates[0].Target)
	assert.Equal(t, os.FileMode(0600), templates[0].Mode)

	_, err = templates[0].Target.Write([]byte("hello world"))
	assert.NoError(t, err)

	info, err = fs.Stat("out")
	assert.NoError(t, err)
	assert.Equal(t, iohelpers.NormalizeFileMode(0600), info.Mode())
	fs.Remove("out")

	cfg = &config.Config{
		InputFiles:  []string{"foo"},
		OutputFiles: []string{"out"},
		OutMode:     "755",
		Stdout:      &bytes.Buffer{},
	}
	templates, err = gatherTemplates(cfg, nil)
	assert.NoError(t, err)
	assert.Len(t, templates, 1)
	assert.Equal(t, "bar", templates[0].Contents)
	assert.NotEqual(t, cfg.Stdout, templates[0].Target)
	assert.Equal(t, iohelpers.NormalizeFileMode(0755), templates[0].Mode)

	_, err = templates[0].Target.Write([]byte("hello world"))
	assert.NoError(t, err)

	info, err = fs.Stat("out")
	assert.NoError(t, err)
	assert.Equal(t, iohelpers.NormalizeFileMode(0755), info.Mode())
	fs.Remove("out")

	templates, err = gatherTemplates(&config.Config{
		InputDir:  "in",
		OutputDir: "out",
	}, simpleNamer("out"))
	assert.NoError(t, err)
	assert.Len(t, templates, 3)
	assert.Equal(t, "foo", templates[0].Contents)
	fs.Remove("out")
}

func TestProcessTemplates(t *testing.T) {
	origfs := fs
	defer func() { fs = origfs }()
	fs = afero.NewMemMapFs()
	afero.WriteFile(fs, "foo", []byte("bar"), iohelpers.NormalizeFileMode(0600))

	afero.WriteFile(fs, "in/1", []byte("foo"), iohelpers.NormalizeFileMode(0644))
	afero.WriteFile(fs, "in/2", []byte("bar"), iohelpers.NormalizeFileMode(0640))
	afero.WriteFile(fs, "in/3", []byte("baz"), iohelpers.NormalizeFileMode(0644))

	afero.WriteFile(fs, "existing", []byte(""), iohelpers.NormalizeFileMode(0644))

	cfg := &config.Config{
		Stdout: &bytes.Buffer{},
	}
	testdata := []struct {
		templates []*Tplate
		contents  []string
		modes     []os.FileMode
		targets   []io.Writer
	}{
		{},
		{
			templates: []*Tplate{{Name: "<arg>", Contents: "foo", TargetPath: "-", Mode: 0644}},
			contents:  []string{"foo"},
			modes:     []os.FileMode{0644},
			targets:   []io.Writer{cfg.Stdout},
		},
		{
			templates: []*Tplate{{Name: "<arg>", Contents: "foo", TargetPath: "out", Mode: 0644}},
			contents:  []string{"foo"},
			modes:     []os.FileMode{0644},
		},
		{
			templates: []*Tplate{{Name: "foo", TargetPath: "out", Mode: 0600}},
			contents:  []string{"bar"},
			modes:     []os.FileMode{0600},
		},
		{
			templates: []*Tplate{{Name: "foo", TargetPath: "out", Mode: 0755}},
			contents:  []string{"bar"},
			modes:     []os.FileMode{0755},
		},
		{
			templates: []*Tplate{
				{Name: "in/1", TargetPath: "out/1", Mode: 0644},
				{Name: "in/2", TargetPath: "out/2", Mode: 0640},
				{Name: "in/3", TargetPath: "out/3", Mode: 0644},
			},
			contents: []string{"foo", "bar", "baz"},
			modes:    []os.FileMode{0644, 0640, 0644},
		},
		{
			templates: []*Tplate{
				{Name: "foo", TargetPath: "existing", Mode: 0755},
			},
			contents: []string{"bar"},
			modes:    []os.FileMode{0644},
		},
		{
			templates: []*Tplate{
				{Name: "foo", TargetPath: "existing", Mode: 0755, ModeOverride: true},
			},
			contents: []string{"bar"},
			modes:    []os.FileMode{0755},
		},
	}
	for i, in := range testdata {
		in := in
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			actual, err := processTemplates(cfg, in.templates)
			assert.NoError(t, err)
			assert.Len(t, actual, len(in.templates))
			for i, a := range actual {
				current := in.templates[i]
				assert.Equal(t, in.contents[i], a.Contents)
				assert.Equal(t, current.Mode, a.Mode)
				if len(in.targets) > 0 {
					assert.Equal(t, in.targets[i], a.Target)
				}
				if current.TargetPath != "-" && current.Name != "<arg>" {
					_, err = current.loadContents(nil)
					assert.NoError(t, err)

					n, err := current.Target.Write([]byte("hello world"))
					assert.NoError(t, err)
					assert.Equal(t, 11, n)

					info, err := fs.Stat(current.TargetPath)
					assert.NoError(t, err)
					assert.Equal(t, iohelpers.NormalizeFileMode(in.modes[i]), info.Mode())
				}
			}
			fs.Remove("out")
		})
	}
}

func TestCreateOutFile(t *testing.T) {
	origfs := fs
	defer func() { fs = origfs }()
	fs = afero.NewMemMapFs()
	_ = fs.Mkdir("in", 0755)

	_, err := createOutFile("in", 0644, false)
	assert.Error(t, err)
	assert.IsType(t, &os.PathError{}, err)
}
