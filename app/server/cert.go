package server

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

type ICertManager interface {
	GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error)
	HTTPHandler(fallback http.Handler) http.Handler
}

type Cert struct {
	cm ICertManager
}

func NewCert(config *Config) *Cert {
	certManager := &autocert.Manager{
		// HostPolicy: autocert.HostWhitelist(domains...), // no need white list, accept all domains
		Prompt: autocert.AcceptTOS,
		Cache:  autocert.DirCache(config.ConfigDir + "/certs"),
		Email:  config.Email,
	}
	if config.IsDev {
		log.Printf("Cert manager is in dev mode, using staging server")
		certManager.Client = &acme.Client{
			DirectoryURL: "https://acme-staging-v02.api.letsencrypt.org/directory",
		}
	}else{
		log.Printf("Cert manager is in production mode, using production server")
		certManager.Client = &acme.Client{
			DirectoryURL: "https://acme-v02.api.letsencrypt.org/directory",
		}
	}
	return &Cert{cm: certManager}
}

func (c *Cert) GetCertManager() ICertManager {
	return c.cm
}

func (c *Cert) GetCertificate(domain string, cacheDir string) error {
	cert, err := c.cm.GetCertificate(&tls.ClientHelloInfo{ServerName: domain})
	if err != nil {
		// http.Error(w, "Failed to get certificate", http.StatusInternalServerError)
		log.Printf("Failed to get certificate: %s:%v", domain, err)
		return err
	}
	log.Printf("Certificate for %s obtained successfully: NotAfter=%s, Issuer=%s", domain, cert.Leaf.NotAfter, cert.Leaf.Issuer)

	// Save the certificate and key to the cacheDir
	fullchainPath := filepath.Join(cacheDir, "fullchain.pem")
	privkeyPath := filepath.Join(cacheDir, "privkey.pem")
	chainPath := filepath.Join(cacheDir, "chain.pem")

	// Convert the certificate to PEM format
	fullchainPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]})
	privkeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(cert.PrivateKey.(*rsa.PrivateKey))})

	// Write the fullchain and private key to files
	err = os.WriteFile(fullchainPath, fullchainPEM, 0644)
	if err != nil {
		log.Printf("Failed to write fullchain to %s: %v", fullchainPath, err)
		return err
	}

	err = os.WriteFile(privkeyPath, privkeyPEM, 0600)
	if err != nil {
		log.Printf("Failed to write private key to %s: %v", privkeyPath, err)
		return err
	}

	// Write the chain to a separate file, and append it to the fullchain
	if len(cert.Certificate) > 1 {
		chainPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[1]})
		err = os.WriteFile(chainPath, chainPEM, 0644)
		if err != nil {
			log.Printf("Failed to write chain to %s: %v", chainPath, err)
			return err
		}
		err = os.WriteFile(fullchainPath, append(fullchainPEM, chainPEM...), 0644)
		if err != nil {
			log.Printf("Failed to append chain to fullchain: %v", err)
			return err
		}
	}

	log.Printf("Certificate and private key have been saved for %s.", domain)
	return nil
}

func GetExpireTime(file string) (*time.Time, string, string) {
	certData, err := os.ReadFile(file)
	if err != nil {
		log.Printf("[Cert]: failed to read %s from disk: %v", file, err)
		return nil, "", ""
	}

	certificates, err := parsePEMBundle(certData)
	if err != nil {
		log.Printf("[Cert]: failed to parsePEMBundle: %s", err)
		return nil, "", ""
	}

	if len(certificates) == 0 {
		log.Printf("no certs found")
		return nil, "", ""
	}

	// check if first cert is CA
	x509Cert := certificates[0]
	if x509Cert.IsCA {
		log.Printf("[Cert][%s] certificate bundle starts with a CA certificate", x509Cert.DNSNames)
		return nil, "", ""
	}

	return &x509Cert.NotAfter, x509Cert.DNSNames[0], x509Cert.IssuingCertificateURL[0]
}

// parsePEMBundle parses a certificate bundle from top to bottom and returns
// a slice of x509 certificates. This function will error if no certificates are found.
func parsePEMBundle(bundle []byte) ([]*x509.Certificate, error) {

	var (
		certificates []*x509.Certificate
		certDERBlock *pem.Block
	)

	for {
		certDERBlock, bundle = pem.Decode(bundle)
		if certDERBlock == nil {
			break
		}

		if certDERBlock.Type == "CERTIFICATE" {
			cert, err := x509.ParseCertificate(certDERBlock.Bytes)
			if err != nil {
				return nil, err
			}
			certificates = append(certificates, cert)
		}
	}

	if len(certificates) == 0 {
		return nil, errors.New("No certificates were found while parsing the bundle")
	}

	return certificates, nil
}
