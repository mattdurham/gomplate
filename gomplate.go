// Package gomplate  is a template renderer which supports a number of datasources,
// and includes hundreds of built-in functions.
package gomplate

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/hairyhenderson/gomplate/v3/data"
	"github.com/hairyhenderson/gomplate/v3/internal/config"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spf13/afero"
)

// Gomplate -
type Gomplate struct {
	Tmplctx         interface{}
	funcMap         template.FuncMap
	nestedTemplates templateAliases
	rootTemplate    *template.Template

	leftDelim, rightDelim string
}

// runTemplate -
func (g *Gomplate) runTemplate(_ context.Context, t *Tplate) error {
	tmpl, err := t.ToGoTemplate(g)
	if err != nil {
		return err
	}

	// nolint: gocritic
	switch t.Target.(type) {
	case io.Closer:
		if t.Target != os.Stdout {
			// nolint: errcheck
			defer t.Target.(io.Closer).Close()
		}
	}
	err = tmpl.Execute(t.Target, g.Tmplctx)
	return err
}

type templateAliases map[string]string

// NewGomplate -
func NewGomplate(funcMap template.FuncMap, leftDelim, rightDelim string, nested templateAliases, tctx interface{}) *Gomplate {
	return &Gomplate{
		leftDelim:       leftDelim,
		rightDelim:      rightDelim,
		funcMap:         funcMap,
		nestedTemplates: nested,
		Tmplctx:         tctx,
	}
}

func parseTemplateArgs(templateArgs []string) (templateAliases, error) {
	nested := templateAliases{}
	for _, templateArg := range templateArgs {
		err := parseTemplateArg(templateArg, nested)
		if err != nil {
			return nil, err
		}
	}
	return nested, nil
}

func parseTemplateArg(templateArg string, ta templateAliases) error {
	parts := strings.SplitN(templateArg, "=", 2)
	pth := parts[0]
	alias := ""
	if len(parts) > 1 {
		alias = parts[0]
		pth = parts[1]
	}

	switch fi, err := fs.Stat(pth); {
	case err != nil:
		return err
	case fi.IsDir():
		files, err := afero.ReadDir(fs, pth)
		if err != nil {
			return err
		}
		prefix := pth
		if alias != "" {
			prefix = alias
		}
		for _, f := range files {
			if !f.IsDir() { // one-level only
				ta[path.Join(prefix, f.Name())] = path.Join(pth, f.Name())
			}
		}
	default:
		if alias != "" {
			ta[alias] = pth
		} else {
			ta[pth] = pth
		}
	}
	return nil
}

// RunTemplates - run all Gomplate templates specified by the given configuration
//
// Deprecated: use Run instead
func RunTemplates(o *Config) error {
	cfg, err := o.toNewConfig()
	if err != nil {
		return err
	}
	return Run(context.Background(), cfg)
}

// Run all Gomplate templates specified by the given configuration
func Run(ctx context.Context, cfg *config.Config) error {
	log := zerolog.Ctx(ctx)

	Metrics = newMetrics()
	defer runCleanupHooks()

	d := data.FromConfig(ctx, cfg)
	log.Debug().Str("data", fmt.Sprintf("%+v", d)).Msg("created data from config")

	addCleanupHook(d.Cleanup)
	nested, err := parseTemplateArgs(cfg.Templates)
	if err != nil {
		return err
	}
	c, err := createTmplContext(ctx, cfg.Context, d)
	if err != nil {
		return err
	}
	funcMap := CreateFuncs(ctx, d)
	err = BindPlugins(ctx, cfg, funcMap)
	if err != nil {
		return err
	}
	g := NewGomplate(funcMap, cfg.LDelim, cfg.RDelim, nested, c)

	return g.runTemplates(ctx, cfg)
}

func (g *Gomplate) runTemplates(ctx context.Context, cfg *config.Config) error {
	start := time.Now()
	tmpl, err := gatherTemplates(cfg, chooseNamer(cfg, g))
	Metrics.GatherDuration = time.Since(start)
	if err != nil {
		Metrics.Errors++
		return fmt.Errorf("failed to gather templates for rendering: %w", err)
	}
	Metrics.TemplatesGathered = len(tmpl)
	start = time.Now()
	defer func() { Metrics.TotalRenderDuration = time.Since(start) }()
	for _, t := range tmpl {
		tstart := time.Now()
		err := g.runTemplate(ctx, t)
		Metrics.RenderDuration[t.Name] = time.Since(tstart)
		if err != nil {
			Metrics.Errors++
			return fmt.Errorf("failed to render template %s: %w", t.Name, err)
		}
		Metrics.TemplatesProcessed++
	}
	return nil
}

func chooseNamer(cfg *config.Config, g *Gomplate) func(string) (string, error) {
	if cfg.OutputMap == "" {
		return simpleNamer(cfg.OutputDir)
	}
	return mappingNamer(cfg.OutputMap, g)
}

func simpleNamer(outDir string) func(inPath string) (string, error) {
	return func(inPath string) (string, error) {
		outPath := filepath.Join(outDir, inPath)
		return filepath.Clean(outPath), nil
	}
}

func mappingNamer(outMap string, g *Gomplate) func(string) (string, error) {
	return func(inPath string) (string, error) {
		out := &bytes.Buffer{}
		t := &Tplate{
			Name:     "<OutputMap>",
			Contents: outMap,
			Target:   out,
		}
		tpl, err := t.ToGoTemplate(g)
		if err != nil {
			return "", err
		}
		tctx := &Tmplctx{}
		// nolint: gocritic
		switch c := g.Tmplctx.(type) {
		case *Tmplctx:
			for k, v := range *c {
				if k != "in" && k != "ctx" {
					(*tctx)[k] = v
				}
			}
		}
		(*tctx)["ctx"] = g.Tmplctx
		(*tctx)["in"] = inPath

		err = tpl.Execute(t.Target, tctx)
		if err != nil {
			return "", errors.Wrapf(err, "failed to render outputMap with ctx %+v and inPath %s", tctx, inPath)
		}

		return filepath.Clean(strings.TrimSpace(out.String())), nil
	}
}
