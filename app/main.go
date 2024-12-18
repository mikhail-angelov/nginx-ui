package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/mikhail-angelov/nginx-ui/app/server"
)

const IS_AUTH = true

func main() {
	isDev := flag.Bool("d", false, "a bool")
	configDir := flag.String("configDir", "temp", "Directory for storing configuration files")
	email := flag.String("email", "mikhail.angelov@gmail.com", "Email address for certificate registration")
	port := flag.String("port", "3005", "http port")

	flag.Parse()
	cert := server.NewCert("certs", *email, *isDev)
	nginx := server.NewNginx(*configDir, *isDev)
	service := server.NewService(cert, *configDir+"/conf")
	web := server.NewWeb(nginx, service)

	log.Printf("Server started on :%s port ✅", *port)
	// make sure to use the cert manager's HTTP handler is expose on 80 port for http-01 challenge
	// .well-known/acme-challenge ... path
	log.Fatal(http.ListenAndServe(":"+*port, cert.GetCertManager().HTTPHandler(web.GetRouter())))

	os.Exit(1)
}
