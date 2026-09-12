package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerate(t *testing.T) {
	err := Generate(context.Background(), GeneratorOptions{
		GenerateMain:     true,
		GenerateServer:   true,
		GenerateService:  true,
		GenerateData:     true,
		GenerateMakefile: true,
		GenerateConfigs:  true,

		ProjectModule: "github.com/gowind-example",
		ProjectName:   "gowind-example",
		ServiceName:   "user",

		Servers:   []string{"rest", "grpc"},
		DbClients: []string{"ent", "redis"},

		OutputPath: t.TempDir(),
	})
	assert.Nil(t, err)
}

func TestHasBFFService(t *testing.T) {
	tests := []struct {
		name     string
		servers  []string
		expected bool
	}{
		{
			name:     "grpc only",
			servers:  []string{"grpc"},
			expected: false,
		},
		{
			name:     "rest only",
			servers:  []string{"rest"},
			expected: true,
		},
		{
			name:     "grpc and rest",
			servers:  []string{"grpc", "rest"},
			expected: true,
		},
		{
			name:     "empty",
			servers:  []string{},
			expected: false,
		},
		{
			name:     "grpc rest uppercase",
			servers:  []string{"gRPC", "REST"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := GeneratorOptions{Servers: tt.servers}
			assert.Equal(t, tt.expected, opts.HasBFFService())
		})
	}
}

func TestAppendServiceName(t *testing.T) {
	g := NewGenerator()
	tmp := t.TempDir()

	err := g.appendServiceName(tmp, "test", "user", false)
	assert.Nil(t, err)

	err = g.appendServiceName(tmp, "test", "order", false)
	assert.Nil(t, err)

	err = g.appendServiceName(tmp, "test", "admin", true)
	assert.Nil(t, err)

	err = g.appendServiceName(tmp, "test", "front", true)
	assert.Nil(t, err)
}

func TestWriteMakefile(t *testing.T) {
	g := NewGenerator()

	err := g.writeMakefile(t.TempDir())
	assert.Nil(t, err)
}

func TestWriteConfigs(t *testing.T) {
	g := NewGenerator()

	err := g.writeConfigs(t.TempDir())
	assert.Nil(t, err)
}
