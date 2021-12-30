package loader

import (
	"bytes"
	"context"
	"github.com/hairyhenderson/gomplate/v3"
	"github.com/hairyhenderson/gomplate/v3/data"
)

type ConfigLoader struct {
	data   *data.Data
	gplate *gomplate.Gomplate
}

func NewConfigLoader(ctx context.Context, sources map[string]*data.Source) *ConfigLoader {
	d := &data.Data{
		Ctx:     ctx,
		Sources: sources,
	}
	funcMap := gomplate.CreateFuncs(ctx, d)
	gplate := gomplate.NewGomplate(funcMap, "{{", "}}", nil, ctx)
	return &ConfigLoader{
		data:   d,
		gplate: gplate,
	}
}

func (c *ConfigLoader) GenerateTemplate(name string, template string) (string, error) {
	tpl := &gomplate.Tplate{
		Name:         name,
		TargetPath:   "",
		Target:       nil,
		Contents:     template,
		Mode:         0,
		ModeOverride: false,
	}
	goTemplate, err := tpl.ToGoTemplate(c.gplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	tctx := &gomplate.Tmplctx{}
	// nolint: gocritic
	switch c := c.gplate.Tmplctx.(type) {
	case *gomplate.Tmplctx:
		for k, v := range *c {
			if k != "in" && k != "ctx" {
				(*tctx)[k] = v
			}
		}
	}
	(*tctx)["ctx"] = gomplate.Tmplctx{}
	err = goTemplate.Execute(&buf, tctx)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
