package main

import (
    "fmt"
    "net/http"

    "github.com/mikhail-angelov/nginx-ui/app/server"
  )
  
  func main() {
    router := NewRouter()

    router.GET("/test/:id", func(w http.ResponseWriter, r *http.Request) {
        fmt.Println(r.Context().Value(ContextKey("id"))) // logs the first path parameter
        fmt.Println(r.Context().Value(ContextKey("otherid"))) // logs the second path parameter
        fmt.Println(r.FormValue("name")) // logs a query parameter
      })
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
      fmt.Fprint(w, "Hello, World!")
    })
    fmt.Println("Server listening on port 3005...")
    http.ListenAndServe(":3005", nil)
  }
