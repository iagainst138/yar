# yar
Yet Another Router.

Handles routing HTTP requests to functions based matching of static entries or regular expressions. Regular expressions are checked from longest to shortest.

#### Example

```
package main

import (
	"./yar"
	"fmt"
	"net/http"
)

type APIHandler struct {
	Router *yar.Router
}

func NewAPIHandler() *APIHandler {
	r := yar.NewRouter()
	a := APIHandler{
		Router: r,
	}
	r.NotFound = a.apiNotFound
	// add routes here
	r.HandleFunc("/api[/]*$", a.apiIndex)
	r.HandleFunc("/api/example$", a.someAPIFunc)
	return &a
}

func (a *APIHandler) apiHandler(w http.ResponseWriter, r *http.Request) {
	a.Router.ServeHTTP(w, r)
}

func (a *APIHandler) apiIndex(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Info on API.\n")
}

func (a *APIHandler) apiNotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(404)
	w.Write([]byte("404: Page not found - " + r.URL.Path + "\n"))
}
func (a *APIHandler) someAPIFunc(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "someAPIFunc: %s\n", r.URL.Path[1:])
}

func index(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.URL.Query())
	fmt.Fprintf(w, "index: %s\n", r.URL.Path)
}

func hello(w http.ResponseWriter, r *http.Request) {
	m, f := yar.Parse(r)
	fmt.Println(m)
	fmt.Println(f)
	fmt.Fprintf(w, "Hello: %s %s\n", m["first_name"], m["last_name"])
}

func notFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(404)
	w.Write([]byte("<div style='width:600px;margin:auto;text-align:center;font-size:200%;'><h1>Whoops!</h1>404: Page not found - " + r.URL.Path + "</div>\n"))
}

func main() {
	rtr := yar.NewRouter()
	rtr.Log = true
	rtr.NotFound = notFound

	api := NewAPIHandler()

	rtr.HandleFunc("/", index)

	rtr.HandleFunc("/api/*", api.apiHandler)
	rtr.HandleFunc("/hello/<first_name>/<last_name>$", hello)

	fmt.Println("RegEx Routes:")
	for _, route := range rtr.Routes {
		fmt.Println(route.Pattern.String())
	}

	fmt.Println("\nFixed routes:")
	for route := range rtr.FixedRoutes {
		fmt.Println(route)
	}

	fmt.Println("listening on 8080")
	http.ListenAndServe(":8080", rtr)
}
```
