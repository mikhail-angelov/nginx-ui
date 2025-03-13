package server

import "flag"

type Config struct {
	IsDev      bool
	IsDocker   bool
	ConfigDir  string
	Email      string
	Pass       string
	Port       string
	RemoteHost string
}

func LoadConfig() *Config {
	isDev := flag.Bool("dev", false, "a bool")
	isDocker := flag.Bool("docker", false, "a bool")
	configDir := flag.String("configDir", "temp", "Directory for storing configuration files")
	email := flag.String("email", "test@test.com", "Email address for auth and certificate registration")
	pass := flag.String("pass", "1", "password for auth")
	port := flag.String("port", "3005", "http port")
	remoteHost := flag.String("remoteHost", "", "Remote host to install Nginx")

	flag.Parse()

	return &Config{
		IsDev:      *isDev,
		IsDocker:   *isDocker,
		ConfigDir:  *configDir,
		Email:      *email,
		Pass:       *pass,
		Port:       *port,
		RemoteHost: *remoteHost,
	}
}
