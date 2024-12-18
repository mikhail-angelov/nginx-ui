package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCert(t *testing.T) {
	cacheDir := "./testdata"
	email := "test@test.com"
	isDebug := true

	cert := NewCert(cacheDir, email, isDebug)

	assert.NotNil(t, cert.certManager, "Expected certManager to be initialized")
	assert.NotNil(t, cert.certManager.Cache, "Expected certManager.Cache to be initialized")
	assert.Equal(t, email, cert.certManager.Email, "Expected certManager.Email to be %s, got %s", email, cert.certManager.Email)

	if isDebug {
		assert.NotNil(t, cert.certManager.Client, "Expected certManager.Client to be initialized in debug mode")
	}
}

func TestGetCertManager(t *testing.T) {
	cacheDir := "./testdata"
	email := "test@test.com"
	isDebug := true

	cert := NewCert(cacheDir, email, isDebug)
	manager := cert.GetCertManager()

	assert.Equal(t, cert.certManager, manager, "Expected GetCertManager to return certManager")
}
