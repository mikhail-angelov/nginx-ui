package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewService(t *testing.T) {
	cacheDir := "../temp/testdata"

	// Create the cache directory for testing
	err := os.MkdirAll(cacheDir, 0755)
	assert.NoError(t, err, "Failed to create cache directory")
	defer os.RemoveAll(cacheDir)

	service := NewService(nil, cacheDir)

	assert.NotNil(t, service, "Expected service to be initialized")
	assert.Equal(t, cacheDir, service.cacheDir, "Expected cacheDir to be set correctly")
}

func TestAddDomain(t *testing.T) {
	cacheDir := "../temp/testdata"

	// Create the cache directory for testing
	err := os.MkdirAll(cacheDir, 0755)
	assert.NoError(t, err, "Failed to create cache directory")
	defer os.RemoveAll(cacheDir)

	service := NewService(nil, cacheDir)

	// Add a new domain
	domain := "example.com"
	err = service.AddDomain(domain)
	assert.NoError(t, err, "Failed to add domain")

	// Check if the domain was added
	assert.Contains(t, service.domains, domain, "Expected domain to be added")

	// Check if the domain directory was created
	domainDir := filepath.Join(cacheDir, domain)
	_, err = os.Stat(domainDir)
	assert.NoError(t, err, "Expected domain directory to be created")

	// Check if the nginx.conf file was created
	nginxConfPath := filepath.Join(domainDir, "nginx.conf")
	_, err = os.Stat(nginxConfPath)
	assert.NoError(t, err, "Expected nginx.conf file to be created")
}

func TestGenerateNginxConfig(t *testing.T) {
	cacheDir := "../temp/testdata"

	// Create the cache directory for testing
	err := os.MkdirAll(cacheDir, 0755)
	assert.NoError(t, err, "Failed to create cache directory")
	defer os.RemoveAll(cacheDir)

	service := NewService(nil, cacheDir)

	// Add a new domain
	domain := "example.com"
	err = service.AddDomain(domain)
	assert.NoError(t, err, "Failed to add domain")

	// Generate nginx.conf for the domain
	err = service.generateNginxConfig(domain)
	assert.NoError(t, err, "Failed to generate nginx.conf")

	// Check if the nginx.conf file was created
	nginxConfPath := filepath.Join(cacheDir, domain, "nginx.conf")
	_, err = os.Stat(nginxConfPath)
	assert.NoError(t, err, "Expected nginx.conf file to be created")
}
