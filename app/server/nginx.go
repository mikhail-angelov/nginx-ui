package server

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// Nginx represents an nginx configuration manager.
type Nginx struct {
	rootPath string
	isDev    bool
	isDocker bool
}

// NewNginx creates a new Nginx instance with the given configuration.
func NewNginx(config *Config) *Nginx {
	return &Nginx{rootPath: config.ConfigDir, isDev: config.IsDev, isDocker: config.IsDocker}
}

func (n *Nginx) getFullName(domain string) string {
	if domain == "main" {
		return n.rootPath + "/nginx.conf"
	}
	return n.rootPath + "/conf/" + domain + "/nginx.conf"
}
func (n *Nginx) runNginxCommand(args []string) string {
	executable := "nginx"
	if n.isDocker {
		executable = "docker"
		args = append([]string{"exec", "-t", "nginx", "nginx"}, args...)
	}
	cmd := exec.Command(executable, args...)
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("nginx run command error: %v\n", err)
		return ""
	}
	log.Printf("nginx run command output: %v\n", string(stdoutStderr))
	return string(stdoutStderr)
}

// CheckNewConfig validates a new configuration for a domain.
func (n *Nginx) CheckNewConfig(name, newContent string) error {
	fullPath := n.getFullName(name)
	err := os.Rename(fullPath, fullPath+".orig")
	if err != nil {
		return fmt.Errorf("failed to rename config: %w", err)
	}
	defer os.Rename(fullPath+".orig", fullPath)
	err = n.SetConfig(name, newContent)
	if err != nil {
		return err
	}
	status := n.runNginxCommand([]string{"-t"})
	if strings.Contains(status, "syntax is ok") {
		return nil
	}
	return errors.New("invalid config")
}

// GetConfig retrieves the configuration content for a domain.
func (n *Nginx) GetConfig(name string) (string, error) {
	fullPath := n.getFullName(name)
	content, err := os.ReadFile(fullPath) //nolint:gosec // file path is constructed from validated domain name
	if err != nil {
		return "", fmt.Errorf("failed to read config: %w", err)
	}
	return string(content), nil
}

// SetConfig writes the configuration content for a domain and refreshes nginx.
func (n *Nginx) SetConfig(name, content string) error {
	fullPath := n.getFullName(name)
	err := os.WriteFile(fullPath, []byte(content), 0o644) //nolint:gosec // nginx configuration files need to be readable by nginx process
	if err != nil {
		log.Printf("Failed to write config %s: %v", fullPath, err)
		return fmt.Errorf("failed to write config: %w", err)
	}
	n.RefreshConfig()
	return nil
}

// RefreshConfig reloads nginx configuration if syntax is valid.
func (n *Nginx) RefreshConfig() {
	log.Println("reloading nginx config")
	status := n.runNginxCommand([]string{"-t"})
	if strings.Contains(status, "syntax is ok") {
		n.runNginxCommand([]string{"-s", "reload"})
		log.Println("nginx config is reloaded")
	}
}
