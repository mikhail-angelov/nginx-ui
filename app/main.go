package main

import (
	"embed"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/mikhail-angelov/nginx-ui/app/server"
)

//go:embed ui/*
var embedFs embed.FS

const IS_AUTH = true

func main() {
	isDev := flag.Bool("dev", false, "a bool")
	configDir := flag.String("configDir", "temp", "Directory for storing configuration files")
	email := flag.String("email", "test@test.com", "Email address for auth and certificate registration")
	pass := flag.String("pass", "1", "password for auth")
	port := flag.String("port", "3005", "http port")

	flag.Parse()
	cert := server.NewCert("certs", *email, *isDev)
	nginx := server.NewNginx(*configDir, *isDev)
	service := server.NewService(cert, *configDir+"/conf", embedFs)
	web := server.NewWeb(nginx, service, *email, *pass, embedFs)

	log.Printf("Server started on :%s port ✅", *port)
	// make sure to use the cert manager's HTTP handler is expose on 80 port for http-01 challenge
	// .well-known/acme-challenge ... path
	log.Fatal(http.ListenAndServe(":"+*port, cert.GetCertManager().HTTPHandler(web.GetRouter())))

	os.Exit(1)
}
