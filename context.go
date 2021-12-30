package gomplate

import (
	"context"
	"os"
	"strings"

	"github.com/hairyhenderson/gomplate/v3/data"
	"github.com/hairyhenderson/gomplate/v3/internal/config"
)

// context for templates
type Tmplctx map[string]interface{}

// Env - Map environment variables for use in a template
func (c *Tmplctx) Env() map[string]string {
	env := make(map[string]string)
	for _, i := range os.Environ() {
		sep := strings.Index(i, "=")
		env[i[0:sep]] = i[sep+1:]
	}
	return env
}

func createTmplContext(ctx context.Context, contexts map[string]config.DataSource, d *data.Data) (interface{}, error) {
	var err error
	tctx := &Tmplctx{}
	for a := range contexts {
		if a == "." {
			return d.Datasource(a)
		}
		(*tctx)[a], err = d.Datasource(a)
		if err != nil {
			return nil, err
		}
	}
	return tctx, nil
}
