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
}

func NewNginx(rootPath string, isDev bool) *nginx {
	return &nginx{rootPath: rootPath, isDev: isDev}
}

func (n *nginx) getFullName(name string) string {
	fullPath := n.rootPath + "/conf/" + name + ".conf"
	if name == "main" {
		fullPath = n.rootPath + "/nginx.conf"
	}
	return fullPath
}
func (n *nginx) runNginxCommand(args []string) string {
	executable := "docker"
	args = append([]string{"exec", "-t", "nginx", "nginx"}, args...)
	
	cmd := exec.Command(executable, args...)
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("certbot error: %v\n", err)
		return ""
	}
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
func (n *nginx) AddConfig(name string, content string) (string, error) {
	err := n.SetConfig(name, content)
	if err != nil {
		return "", err
	}
	return name, nil
}
func (n *nginx) RemoveConfig(name string) error {
	if name == "main" {
		return nil
	}
	fullPath := n.getFullName(name)
	err := os.Remove(fullPath)
	return err
}

func (n *nginx) SetConfig(name string, content string) error {
	fullPath := n.getFullName(name)
	return os.WriteFile(fullPath, []byte(content), 0644)
}

func (n *nginx) GetListOfConfigs() ([]string, error) {
	var names []string
	files, err := os.ReadDir(n.rootPath + "/conf")
	if err != nil {
		return names, err
	}
	for _, file := range files {
		name := strings.TrimSuffix(file.Name(), ".conf")
		names = append(names, name)
	}
	return names, nil
}
