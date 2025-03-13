package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var config = &Config{
	IsDev:     true,
	IsDocker:  true,
	ConfigDir: "./testdata",
	Email:     "test@test.com",
	Pass:      "1",
	Port:      "3005",
}

func TestNewCert(t *testing.T) {
	cacheDir := "./testdata"

	cert := NewCert(cacheDir, config)

	assert.NotNil(t, cert.certManager, "Expected certManager to be initialized")
	assert.NotNil(t, cert.certManager.Cache, "Expected certManager.Cache to be initialized")
	assert.Equal(t, config.Email, cert.certManager.Email, "Expected certManager.Email to be %s, got %s", config.Email, cert.certManager.Email)

}

func TestGetCertManager(t *testing.T) {
	cacheDir := "./testdata"

	cert := NewCert(cacheDir, config)
	manager := cert.GetCertManager()

	assert.Equal(t, cert.certManager, manager, "Expected GetCertManager to return certManager")
}
