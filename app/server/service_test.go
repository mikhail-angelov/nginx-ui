package server

import (
	"embed"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewService(t *testing.T) {
	cacheDir := "../temp/testdata"

	// Create the cache directory for testing
	err := os.MkdirAll(cacheDir, 0755)
	assert.NoError(t, err, "Failed to create cache directory")
	defer os.RemoveAll(cacheDir)

	var efs embed.FS
	service := NewService(nil, cacheDir, efs)

	assert.NotNil(t, service, "Expected service to be initialized")
	assert.Equal(t, cacheDir, service.cacheDir, "Expected cacheDir to be set correctly")
}


func TestGenerateNginxConfig(t *testing.T) {
	cacheDir := "../temp/testdata"

	// Create the cache directory for testing
	err := os.MkdirAll(cacheDir, 0755)
	assert.NoError(t, err, "Failed to create cache directory")
	// defer os.RemoveAll(cacheDir)

	var efs embed.FS
	service := NewService(nil, cacheDir, efs)
	domain := "example.com"
	os.Mkdir(cacheDir+"/"+domain, 0755) //room to save config file

	// Generate nginx.conf for the domain
	err = service.generateNginxConfig(domain, "../ui/configs/nginx.tmpl")
	assert.NoError(t, err, "Failed to generate nginx.conf")

}
