package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/mikhail-angelov/nginx-ui/app/server"
)

const IS_AUTH = true

func main() {
	isDev := flag.Bool("d", false, "a bool")
	flag.Parse()
	router := server.NewRouter()
	templates := server.LoadTemplates()
	nginx := server.NewNginx("temp", *isDev)

	router.GET(IS_AUTH, "/test/:id", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(r.Context().Value(server.ContextKey("id"))) // logs the first path parameter

		data := map[string]interface{}{
			"Name": r.Context().Value(server.ContextKey("id")),
		}

		templates.Render(w, "main", data)
	})
	router.GET(IS_AUTH, "/", func(w http.ResponseWriter, r *http.Request) {
		data := make(map[string]interface{})
		claim := r.Context().Value(server.ContextKey("claims"))
		fmt.Println(claim)
		error := ""
		if claim == nil {
			data["IsAuth"] = false
		} else {
			configs, err := nginx.GetListOfConfigs()
			if err != nil {
				configs = []string{}
				error = err.Error()
			}

			data["IsAuth"] = true
			data["Configs"] = configs
			data["Error"] = error
		}
		templates.Render(w, "index", data)

	})
	router.GET(IS_AUTH, "/configs", func(w http.ResponseWriter, r *http.Request) {
		data := make(map[string]interface{})
		claim := r.Context().Value(server.ContextKey("claims"))
		fmt.Println(claim)
		error := ""
		if claim == nil {
			data["IsAuth"] = false
		} else {
			configs, err := nginx.GetListOfConfigs()
			if err != nil {
				configs = []string{}
				error = err.Error()
			}

			data["IsAuth"] = true
			data["Configs"] = configs
			data["Error"] = error
		}
		templates.SubRender(w, "index", "configs", data)

	})
	router.GET(IS_AUTH, "/edit/:file", func(w http.ResponseWriter, r *http.Request) {
		error := ""
		path := r.Context().Value(server.ContextKey("path")).(map[string]string)
		name := path["file"]

		content, err := nginx.GetConfig(name)
		if err != nil {
			error = err.Error()
		}
		configs, err := nginx.GetListOfConfigs()
		if err != nil {
			configs = []string{}
			error = err.Error()
		}

		data := map[string]interface{}{
			"Configs": configs,
			"Name":    name,
			"Content": content,
			"Error":   error,
		}

		
		templates.SubRender(w, "index", "editor", data)
	})

	router.POST(IS_AUTH, "/add", func(w http.ResponseWriter, r *http.Request) {
		error := ""

		now := time.Now()
		name := r.Header.Get("HX-Prompt")
		if name == "" {
		name = now.Format("2024-10-01-15-04-05")
		}
		_, err := nginx.AddConfig(name, "")
		if err != nil {
			error = err.Error()
		}
		configs, err := nginx.GetListOfConfigs()
		if err != nil {
			configs = []string{}
			error = err.Error()
		}
		data := map[string]interface{}{
			"Configs": configs,
			"Name":    name,
			"Content": "",
			"Error":   error,
		}
		w.Header().Set("HX-Trigger", "refreshConfigs")
		templates.SubRender(w, "index", "editor", data)
	})
	router.POST(IS_AUTH, "/validate/:file", func(w http.ResponseWriter, r *http.Request) {
		error := ""
		path := r.Context().Value(server.ContextKey("path")).(map[string]string)
		name := path["file"]
		content := r.FormValue("content")
		err := nginx.CheckNewConfig(name, content)
		status := "valid"
		if err != nil {
			error = err.Error()
			status = "invalid"
		}

		data := map[string]interface{}{
			"Status": status,
			"Error":  error,
		}

		templates.SubRender(w, "index", "status", data)
	})
	router.POST(IS_AUTH, "/save/:file", func(w http.ResponseWriter, r *http.Request) {
		error := ""
		path := r.Context().Value(server.ContextKey("path")).(map[string]string)
		name := path["file"]
		content := r.FormValue("content")
		err := nginx.SetConfig(name, content)
		status := "valid"
		if err != nil {
			error = err.Error()
			status = "invalid"
		}

		data := map[string]interface{}{
			"Status": status,
			"Error":  error,
		}

		templates.SubRender(w, "index", "status", data)
	})
	router.POST(IS_AUTH, "/remove/:file", func(w http.ResponseWriter, r *http.Request) {
		error := ""
		path := r.Context().Value(server.ContextKey("path")).(map[string]string)
		name := path["file"]
		err := nginx.RemoveConfig(name)
		if err != nil {
			error = err.Error()
		}

		configs, err := nginx.GetListOfConfigs()
		if err != nil {
			configs = []string{}
			error = err.Error()
		}

		data := map[string]interface{}{
			"IsAuth": true,
			"Configs": configs,
			"Error":   error,
		}

		w.Header().Set("HX-Trigger", "refreshConfigs")
		templates.SubRender(w, "index", "dashboard", data)
	})


	router.POST(false, "/login", func(w http.ResponseWriter, r *http.Request) {
		//validate email and password
		email:=r.FormValue("email")
		password:=r.FormValue("password")
		data := make(map[string]interface{})
		fmt.Println(email+":"+password)
		error := ""
		if password == "" {
			data["IsAuth"] = false
			data["Error"] = "Password is required"
			templates.SubRender(w, "index", "login", data)
		} else {
			configs, err := nginx.GetListOfConfigs()
			if err != nil {
				configs = []string{}
				error = err.Error()
			}

			data["IsAuth"] = true
			data["Configs"] = configs
			data["Error"] = error
			server.SetAuthCookie(w, r.FormValue("email"))
			templates.SubRender(w, "index", "main", data)
		}
	})

	router.POST(false, "/logout", func(w http.ResponseWriter, r *http.Request) {
		server.CleanAuthCookie(w)
		data := make(map[string]interface{})
		data["IsAuth"] = false
		templates.SubRender(w, "index", "login", data)
	})

	l, err := net.Listen("tcp", ":3005")
	if err != nil {
		fmt.Printf("error starting server: %s\n", err)
	}
	fmt.Println("🐱‍💻 server started on", l.Addr().String())
	if err := http.Serve(l, router); err != nil {
		fmt.Printf("server closed: %s\n", err)
	}
	os.Exit(1)
}
