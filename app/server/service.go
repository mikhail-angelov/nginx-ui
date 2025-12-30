package server

import (
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

// Service manages domains, configurations, and certificate renewal.
type Service struct {
	cacheDir string
	domains  []string
	cert     *Cert
	nginx    *Nginx
	isDev    bool
	embedFs  embed.FS
}

// NewService creates a new Service instance with the given dependencies.
func NewService(nginx *Nginx, cert *Cert, config *Config, embedFs embed.FS) *Service {
	cacheDir := config.ConfigDir + "/conf"
	_, err := os.Stat(cacheDir)
	if os.IsNotExist(err) {
		panic("Cache directory does not exist")
	}

	domains, err := getDirectories(cacheDir)
	if err != nil {
		log.Panicf("Failed to get directories: %v", err)
	}

	service := &Service{nginx: nginx, cert: cert, cacheDir: cacheDir, domains: domains, embedFs: embedFs, isDev: config.IsDev}
	go func() {
		for {
			service.checkAndRefreshCertificates()
			time.Sleep(24 * time.Hour)
		}
	}()

	return service
}

// GetDomains returns the list of managed domains.
func (s *Service) GetDomains() []string {
	return s.domains
}

// AddDomain adds a new domain to be managed.
// Returns the configuration content and an error if any.
func (s *Service) AddDomain(domain string) (string, error) {
	log.Printf("Adding domain: %s", domain)
	if slices.Contains(s.domains, domain) {
		log.Printf("Domain %s already exists", domain)
		return "", errors.New("domain already exists")
	}
	if !isValidDomain(domain) {
		log.Printf("Invalid domain name: %s", domain)
		return "", errors.New("invalid domain name")
	}
	if !isDomainResolvable(domain) {
		log.Printf("Domain %s is not resolvable", domain)
		return "", errors.New("domain is not resolvable")
	}

	err := os.Mkdir(s.cacheDir+"/"+domain, 0o755) //nolint:gosec // domain directories need to be readable by nginx
	if err != nil {
		log.Printf("Failed to create directory %s: %v", s.cacheDir+"/"+domain, err)
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Generate nginx.conf for the new domain
	templatePaths, err := fs.Glob(s.embedFs, "ui/configs/nginx.tmpl")
	if err != nil || len(templatePaths) == 0 {
		return "", errors.New("template file not found")
	}
	templatePath := templatePaths[0]
	err = s.generateNginxConfig(domain, templatePath)
	if err != nil {
		log.Printf("Failed to generate nginx.conf for %s: %v", domain, err)
		if rmErr := os.RemoveAll(s.cacheDir + "/" + domain); rmErr != nil {
			log.Printf("Failed to remove directory after template generation failure: %v", rmErr)
		}
		return "", fmt.Errorf("failed to generate nginx config: %w", err)
	}
	// Generate SSL certificate for the new domain
	err = s.cert.GetCertificate(domain, s.cacheDir+"/"+domain)
	if err != nil {
		log.Printf("Failed to get certificate for %s: %v", domain, err)
		if rmErr := os.RemoveAll(s.cacheDir + "/" + domain); rmErr != nil {
			log.Printf("Failed to remove directory after certificate failure: %v", rmErr)
		}
		return "", fmt.Errorf("failed to get certificate: %w", err)
	}

	s.domains = append(s.domains, domain)

	// read config
	content, err := s.nginx.GetConfig(domain)
	if err != nil {
		return "", fmt.Errorf("failed to get config: %w", err)
	}

	return content, nil
}

// RemoveDomain removes a domain from management.
func (s *Service) RemoveDomain(domain string) error {
	if !slices.Contains(s.domains, domain) {
		return errors.New("domain does not exist")
	}

	err := os.RemoveAll(s.cacheDir + "/" + domain)
	if err != nil {
		log.Printf("Failed to remove directory %s: %v", domain, err)
		return fmt.Errorf("failed to remove directory: %w", err)
	}

	s.domains = remove(s.domains, domain)

	return nil
}

func (s *Service) checkAndRefreshCertificates() {
	isRefreshedCertificates := false
	for _, domain := range s.domains {
		certPath := s.cacheDir + "/" + domain + "/fullchain.pem"
		expirationTime, certDomain, issuer := GetExpireTime(certPath)
		log.Printf("Certificate for %s/%s expires on %s, issuer: %s", domain, certDomain, expirationTime, issuer)
		if expirationTime == nil ||
			expirationTime.Sub(time.Now().UTC()).Hours() < (7*24) ||
			(!s.isDev && strings.Contains(issuer, "stg") ||
				(certDomain != domain)) {
			log.Printf("Certificate for %s is expired or will expire soon, refreshing...", domain)

			isValid := isDomainResolvable(domain)
			if !isValid {
				log.Printf("Domain %s is not resolvable, skipping certificate refresh", domain)
				continue
			}

			err := s.cert.GetCertificate(domain, s.cacheDir+"/"+domain)
			isRefreshedCertificates = true
			if err != nil {
				log.Printf("Failed to get certificate for %s: %v", domain, err)
			}
		} else {
			log.Printf("Certificate for %s is still valid", domain)
		}
	}
	if isRefreshedCertificates {
		s.nginx.RefreshConfig()
	}
}

// generateNginxConfig generates an nginx configuration file for the given domain using the template.
func (s *Service) generateNginxConfig(domain, templatePath string) error {
	tmpl, err := template.ParseFS(s.embedFs, templatePath)
	if err != nil {
		log.Printf("Failed to parse template %s: %v", templatePath, err)
		return fmt.Errorf("failed to parse template: %w", err)
	}

	outputPath := filepath.Join(s.cacheDir, domain, "nginx.conf")
	outputFile, err := os.Create(outputPath) //nolint:gosec // output path is constructed from validated domain name
	if err != nil {
		log.Printf("Failed to create file %s: %v", outputPath, err)
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer outputFile.Close()

	data := struct {
		Domain  string
		Path    string
		Backend string
	}{
		Domain:  domain,
		Path:    filepath.Join(s.cacheDir, domain),
		Backend: "http://localhost:3000",
	}
	err = tmpl.Execute(outputFile, data)
	if err != nil {
		log.Printf("Failed to execute template %s, %s: %v", domain, templatePath, err)
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

func getDirectories(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("Failed to read directory %s: %v", dir, err)
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry.Name())
		}
	}

	return dirs, nil
}

func remove(slice []string, item string) []string {
	var result []string
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}
func isValidDomain(domain string) bool {
	// Define a regular expression pattern for a valid domain
	const domainPattern = `^(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`
	re := regexp.MustCompile(domainPattern)
	return re.MatchString(domain)
}
func isDomainResolvable(domain string) bool {
	_, err := net.LookupHost(domain)
	if err != nil {
		log.Printf("Failed to resolve domain %s: %v", domain, err)
		return false
	}
	return true
}
