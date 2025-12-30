package main

import (
	"embed"
	"log"
	"net/http"
	"strconv"
	"time"

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

	log.Printf("Server started (dev:%s) on 127.0.0.1:%s port ✅", strconv.FormatBool(config.IsDev), config.Port)

	// make sure to use the cert manager's HTTP handler is expose on 80 port for http-01 challenge
	// .well-known/acme-challenge ... path
	httpServer := &http.Server{
		Addr:         "127.0.0.1:" + config.Port,
		Handler:      cert.GetCertManager().HTTPHandler(web.GetRouter()),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting server on %s", httpServer.Addr)
	log.Panic(httpServer.ListenAndServe())
}
