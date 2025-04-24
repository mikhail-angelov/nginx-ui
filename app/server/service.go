package server

import (
	"embed"
	"errors"
	"html/template"
	"io/fs"
	"log"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type domainRecord struct {
	domain string
	name   string
}

type Service struct {
	cacheDir string
	domains  []string
	cert     *Cert
	nginx    *nginx
	isDev    bool
	embedFs  embed.FS
}

func NewService(nginx *nginx, cert *Cert, config *Config, embedFs embed.FS) *Service {
	cacheDir:= config.ConfigDir + "/conf"
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

func (s *Service) GetDomains() []string {
	return s.domains
}

func (s *Service) AddDomain(domain string) (error, string) {
	log.Printf("Adding domain: %s", domain)
	if contains(s.domains, domain) {
		log.Printf("Domain %s already exists", domain)
		return errors.New("Domain already exists"), ""
	}
	if !isValidDomain(domain) {
		log.Printf("Invalid domain name: %s", domain)
		return errors.New("Invalid domain name"), ""
	}
	if !isDomainResolvable(domain) {
		log.Printf("Domain %s is not resolvable", domain)
		return errors.New("Domain is not resolvable"), ""
	}

	err := os.Mkdir(s.cacheDir+"/"+domain, 0755)
	if err != nil {
		log.Printf("Failed to create directory %s: %v", s.cacheDir+"/"+domain, err)
		return err, ""
	}

	// Generate nginx.conf for the new domain
	templatePaths, err := fs.Glob(s.embedFs, "ui/configs/nginx.tmpl")
	if err != nil || len(templatePaths) == 0 {
		return errors.New("template file not found"), ""
	}
	templatePath := templatePaths[0]
	err = s.generateNginxConfig(domain, templatePath)
	if err != nil {
		log.Printf("Failed to generate nginx.conf for %s: %v", domain, err)
		os.RemoveAll(s.cacheDir + "/" + domain)
		return err, ""
	}
	
	// Generate SSL certificate for the new domain
	err = s.cert.GetCertificate(domain, s.cacheDir+"/"+domain)
	if err != nil {
		log.Printf("Failed to get certificate for %s: %v", domain, err)
		os.RemoveAll(s.cacheDir + "/" + domain)
		return err, ""
	}

	s.domains = append(s.domains, domain)

	//read config
	content, err := s.nginx.GetConfig(domain)

	return err, content
}

func (s *Service) RemoveDomain(domain string) error {
	if !contains(s.domains, domain) {
		return errors.New("Domain does not exist")
	}

	err := os.RemoveAll(s.cacheDir + "/" + domain)
	if err != nil {
		log.Printf("Failed to remove directory %s: %v", domain, err)
		return err
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

func (s *Service) generateNginxConfig(domain string, templatePath string) error {
	tmpl, err := template.ParseFS(s.embedFs, templatePath)
	if err != nil {
		log.Printf("Failed to parse template %s: %v", templatePath, err)
		return err
	}

	outputPath := filepath.Join(s.cacheDir, domain, "nginx.conf")
	outputFile, err := os.Create(outputPath)
	if err != nil {
		log.Printf("Failed to create file %s: %v", outputPath, err)
		return err
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
		return err
	}

	return nil
}

func getDirectories(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("Failed to read directory %s: %v", dir, err)
		return nil, err
	}

	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry.Name())
		}
	}

	return dirs, nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
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
