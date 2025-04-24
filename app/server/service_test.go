package server

import (
	"embed"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

//go:embed testdata/*
var embedFs embed.FS

func TestNewService(t *testing.T) {
	cacheDir := "../temp/testdata"

	// Create the cache directory for testing
	err := os.MkdirAll(cacheDir, 0755)
	assert.NoError(t, err, "Failed to create cache directory")
	defer os.RemoveAll(cacheDir)

	var efs embed.FS
	service := NewService(nil, nil, config, efs)

	assert.NotNil(t, service, "Expected service to be initialized")
	assert.Equal(t, cacheDir, service.cacheDir, "Expected cacheDir to be set correctly")
}


func TestGenerateNginxConfig(t *testing.T) {
    cacheDir := "../temp/testdata"
    domain := "example.com"
    templatePath := "testdata/nginx.tmpl"

    // Create the cache directory for testing
    err := os.MkdirAll(filepath.Join(cacheDir, domain), 0755)
    assert.NoError(t, err, "Failed to create cache directory")
    defer os.RemoveAll(cacheDir)

    // Create a Service instance
    service := &Service{
        cacheDir: cacheDir,
		embedFs:  embedFs,
    }

    // Generate nginx.conf for the domain
    err = service.generateNginxConfig(domain, templatePath)
    assert.NoError(t, err, "Failed to generate nginx.conf")

    // Check if the nginx.conf file was created
    nginxConfPath := filepath.Join(cacheDir, domain, "nginx.conf")
    _, err = os.Stat(nginxConfPath)
    assert.NoError(t, err, "Expected nginx.conf file to be created")

    // Optionally, read the file and check its contents
    content, err := os.ReadFile(nginxConfPath)
    assert.NoError(t, err, "Failed to read nginx.conf file")
    assert.Contains(t, string(content), "server_name example.com;", "Expected nginx.conf to contain the domain name")
}
