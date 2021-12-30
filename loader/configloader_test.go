package loader

import (
	"context"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestConfigLoader(t *testing.T) {
	cl := NewConfigLoader(context.Background(), nil)
	out, err := cl.GenerateTemplate("test", "name: bob")
	assert.NoError(t, err)
	assert.True(t, out == "name: bob")
}

func TestConfigLoaderWithEnv(t *testing.T) {
	os.Setenv("USER", "dave")
	defer os.Unsetenv("USER")

	cl := NewConfigLoader(context.Background(), nil)
	out, err := cl.GenerateTemplate("test", "name: {{env.Getenv \"USER\"}}")
	assert.NoError(t, err)
	assert.True(t, out == "name: dave")

}
