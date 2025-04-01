package server

import (
	"errors"
	"log"
	"os"
	"os/exec"
	"strings"
)

type nginx struct {
	rootPath string
	isDev    bool
	isDocker bool
}

func NewNginx(config *Config) *nginx {
	return &nginx{rootPath: config.ConfigDir, isDev: config.IsDev, isDocker: config.IsDocker}
}

func (n *nginx) getFullName(domain string) string {
	if domain == "main" {
		return n.rootPath + "/nginx.conf"
	}
	return n.rootPath + "/conf/" + domain + "/nginx.conf"
}
func (n *nginx) runNginxCommand(args []string) string {
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

func (n *nginx) CheckNewConfig(name string, newContent string) error {
	fullPath := n.getFullName(name)
	err := os.Rename(fullPath, fullPath+".orig")
	if err != nil {
		return err
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

func (n *nginx) GetConfig(name string) (string, error) {
	fullPath := n.getFullName(name)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (n *nginx) SetConfig(name string, content string) error {
	fullPath := n.getFullName(name)
	err := os.WriteFile(fullPath, []byte(content), 0644)
	if err != nil {
		log.Printf("Failed to write config %s: %v", fullPath, err)
		return err
	}
	n.RefreshConfig()
	return nil
}

func (n *nginx) RefreshConfig()  {
	log.Println("reloading nginx config")
	status := n.runNginxCommand([]string{"-t"})
	if strings.Contains(status, "syntax is ok") {
		n.runNginxCommand([]string{"-s", "reload"})
		log.Println("nginx config is reloaded")
	}
}
