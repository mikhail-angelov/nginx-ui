package server

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var config = &Config{
	IsDev:     true,
	IsDocker:  true,
	ConfigDir: "./testdata",
	Email:     "test@test.com",
	Pass:      "1",
	Port:      "3005",
}

// MockCertManager is a mock implementation of ICertManager
type MockCertManager struct {
	mock.Mock
}

func (m *MockCertManager) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	args := m.Called(hello)
	err := args.Error(1)
	if err != nil {
		return args.Get(0).(*tls.Certificate), fmt.Errorf("mock error: %w", err)
	}
	return args.Get(0).(*tls.Certificate), nil
}

func (m *MockCertManager) HTTPHandler(fallback http.Handler) http.Handler {
	args := m.Called(fallback)
	return args.Get(0).(http.Handler)
}

func TestNewCert(t *testing.T) {
	cert := NewCert(config)
	assert.NotNil(t, cert.cm, "Expected certManager to be initialized")
}

func TestGetCertManager(t *testing.T) {
	cert := NewCert(config)
	manager := cert.GetCertManager()

	assert.Equal(t, cert.cm, manager, "Expected GetCertManager to return certManager")
}

func TestGetCertificate(t *testing.T) {
	cacheDir := t.TempDir()
	mockManager := new(MockCertManager)

	// Create a test certificate
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	// Create a minimal x509 certificate for the Leaf field
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		DNSNames:     []string{"example.com"},
		NotAfter:     time.Now().Add(24 * time.Hour),
		Issuer: pkix.Name{
			CommonName: "Test CA",
		},
	}

	certDER, _ := x509.CreateCertificate(rand.Reader, template, template, &privKey.PublicKey, privKey)
	x509Cert, _ := x509.ParseCertificate(certDER)

	cert := &tls.Certificate{
		Certificate: [][]byte{certDER, []byte("second-cert")},
		PrivateKey:  privKey,
		Leaf:        x509Cert,
	}

	// Set up the mock to return the test certificate
	mockManager.On("GetCertificate", mock.Anything).Return(cert, nil)

	c := &Cert{cm: mockManager}

	// Call the GetCertificate method
	err := c.GetCertificate("example.com", cacheDir)
	require.NoError(t, err, "Expected no error from GetCertificate")

	// Verify that the certificate and key files were created
	fullchainPath := filepath.Join(cacheDir, "fullchain.pem")
	privkeyPath := filepath.Join(cacheDir, "privkey.pem")

	_, err = os.Stat(fullchainPath)
	require.NoError(t, err, "Expected fullchain.pem to be created")

	_, err = os.Stat(privkeyPath)
	require.NoError(t, err, "Expected privkey.pem to be created")

	// Verify the contents of the fullchain.pem file
	fullchainPEM, err := os.ReadFile(fullchainPath)
	require.NoError(t, err, "Expected to read fullchain.pem")
	assert.Contains(t, string(fullchainPEM), "CERTIFICATE", "Expected fullchain.pem to contain a certificate")

	// Verify that fullchainPEM contains two "BEGIN CERTIFICATE" substrings
	assert.Equal(t, 2, strings.Count(string(fullchainPEM), "BEGIN CERTIFICATE"), "Expected fullchain.pem to contain two certificates")

	// Verify the contents of the privkey.pem file
	privkeyPEM, err := os.ReadFile(privkeyPath)
	require.NoError(t, err, "Expected to read privkey.pem")
	assert.Contains(t, string(privkeyPEM), "RSA PRIVATE KEY", "Expected privkey.pem to contain a private key")

	// Verify that the mock was called as expected
	mockManager.AssertExpectations(t)

}
