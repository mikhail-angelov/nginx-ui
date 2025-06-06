package main

import (
	"embed"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/mikhail-angelov/nginx-ui/app/server"
)

//go:embed ui/*
var embedFs embed.FS

func main() {
	config := server.LoadConfig()

	cert := server.NewCert(config)
	nginx := server.NewNginx(config)
	service := server.NewService(nginx, cert, config, embedFs)
	web := server.NewWeb(nginx, service, config, embedFs)

	log.Printf("Server started (dev:%s) on :%s port ✅", strconv.FormatBool(config.IsDev), config.Port)
	// make sure to use the cert manager's HTTP handler is expose on 80 port for http-01 challenge
	// .well-known/acme-challenge ... path
	log.Panic(http.ListenAndServe(":"+config.Port, cert.GetCertManager().HTTPHandler(web.GetRouter())))

	os.Exit(1)
}
