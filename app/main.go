package main

import (
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/mikhail-angelov/nginx-ui/app/server"
)

const IS_AUTH = true

func main() {
	router := server.NewRouter()
	templates := server.LoadTemplates("ui/templates/*.tmpl")

	router.GET(IS_AUTH, "/test/:id", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(r.Context().Value(server.ContextKey("id")))      // logs the first path parameter
		
		data := map[string]interface{}{
			"Name": r.Context().Value(server.ContextKey("id")),
		}

		templates.Render(w, "index", data)
	})
	router.GET(IS_AUTH, "/", func(w http.ResponseWriter, r *http.Request) {

		data := map[string]interface{}{
			"Name": "test",
		}

		templates.Render(w, "index", data)

	})

	router.GET(false, "/login", func(w http.ResponseWriter, r *http.Request) {
		claims, err := server.GetAuthCookie(r)
		if err == nil && claims != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		templates.Render(w, "login", nil)
	})
	router.POST(false, "/login", func(w http.ResponseWriter, r *http.Request) {
		//validate email and password

		server.SetAuthCookie(w, r.FormValue("email"))
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
	router.POST(false, "/logout", func(w http.ResponseWriter, r *http.Request) {
		server.CleanAuthCookie(w)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	fmt.Println("Server listening on port 3005...")
	l, err := net.Listen("tcp", ":3005")
	if err != nil {
		fmt.Printf("error starting server: %s\n", err)
	}
	fmt.Println("🐱‍💻 BeanGo server started on", l.Addr().String())
	if err := http.Serve(l, router); err != nil {
		fmt.Printf("server closed: %s\n", err)
	}
	os.Exit(1)
}
