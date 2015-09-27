package yar // Yet Another Router

import (
	"errors"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

const (
	ParamRegex = "<([A-z0-9_]*?)>"	// regexp to match variable declarations
	ParamMatch = "([A-z0-9_].*?)"	// regexp to extract variables from the URI
)

// Route is a route that contains a regexp and func to call
type Route struct {
	Pattern *regexp.Regexp
	Func    http.HandlerFunc
}

// ParameterRoute is a route that has variables in the URI
type ParameterRoute struct {
	Func     http.HandlerFunc
	VarNames []string
	Regexp   *regexp.Regexp
}

// Extracts the "variable form" from the url and prepends them to the RawQuery
// of the http.Request object
// The user can then call the Parse function to get these form.
// NOTE: The form will appear in the Form field of the http.Request, if its a
// GET request the value will be the first in the slice but if its a PUT or POST
// it will the last.
func (pr *ParameterRoute) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := pr.Regexp.FindStringSubmatch(r.URL.Path)[1:]
	form := url.Values{}
	for i, vn := range pr.VarNames {
		form.Add(vn, vars[i])
	}
	// idea got from here - https://github.com/bmizerany/pat/blob/master/mux.go
	r.URL.RawQuery = form.Encode() + "&" + r.URL.RawQuery
	pr.Func(w, r)
}

// Routes is an array of routes that is sorted by regex length
type Routes []*Route

func (r Routes) Len() int {
	return len(r)
}

func (r Routes) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r Routes) Less(i, j int) bool {
	return len(r[i].Pattern.String()) > len(r[j].Pattern.String())
}

// Router handles HTTP requests and works out what functions
// should be called based on matching
type Router struct {
	// a map of strings to handler functions
	FixedRoutes map[string]http.HandlerFunc
	// a length sorted list of regexps
	Routes Routes
	// should trailing / be stripped from path
	Strip bool
	// log requests?
	Log         bool
	CheckRegexp bool
	// 404 handler, defaults to http.NotFound
	NotFound http.HandlerFunc
}

// NewRouter returns a Router
func NewRouter() *Router {
	return &Router{
		FixedRoutes: map[string]http.HandlerFunc{},
		Routes:      Routes{},
		Strip:       false,
		Log:         false,
		CheckRegexp: true,
		NotFound:    http.NotFound,
	}
}

func (rtr *Router) HandleFunc(pattern string, f http.HandlerFunc) {
	re := regexp.MustCompile(ParamRegex)
	vars := re.FindAllString(pattern, -1)
	if len(vars) > 0 {
		rtr.addProcessedParameterRoute(pattern, re, f)
	} else if rtr.CheckRegexp {
		quoted := regexp.QuoteMeta(pattern)
		if quoted == pattern {
			rtr.addFixedRoute(pattern, f)
		} else {
			rtr.addRoute(pattern, f)
		}
	} else {
		rtr.addFixedRoute(pattern, f)
	}
}

func (rtr *Router) addFixedRoute(pattern string, f http.HandlerFunc) error {
	if _, exists := rtr.FixedRoutes[pattern]; exists {
		return errors.New("Key exists: " + pattern)
	}
	rtr.FixedRoutes[pattern] = f
	return nil
}

func (rtr *Router) addRoute(pattern string, f http.HandlerFunc) error {
	re := regexp.MustCompile(pattern)
	for _, r := range rtr.Routes {
		if r.Pattern.String() == re.String() {
			return errors.New("Key exists: " + pattern)
		}
	}
	rtr.Routes = append(rtr.Routes, &Route{re, f})
	sort.Sort(rtr.Routes)
	return nil
}

func (rtr *Router) addParameterRoute(pattern string, f http.HandlerFunc) {
	re := regexp.MustCompile(ParamRegex)
	rtr.addProcessedParameterRoute(pattern, re, f)
}

func (rtr *Router) addProcessedParameterRoute(pattern string, re *regexp.Regexp, f http.HandlerFunc) {
	varNames := []string{}
	vars := re.FindAllString(pattern, -1)
	newPattern := pattern
	for _, vn := range vars {
		newPattern = strings.Replace(newPattern, vn, ParamMatch, 1)
		varNames = append(varNames, vn[1:len(vn)-1]) // strip the < and > characters
	}
	pr := ParameterRoute{f, varNames, regexp.MustCompile(newPattern)}
	rtr.addRoute(newPattern, pr.ServeHTTP)
}

func (rtr *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	logMsg := "requested: " + path
	// only strip "/" if its not the entire path
	if rtr.Strip && len(path) > 1 && strings.HasSuffix(path, "/") {
		// is this just overhead?
		path = strings.TrimSuffix(path, "/")
		logMsg += " (stripped to: " + path + ")"
	}
	if rtr.Log {
		log.Println(logMsg)
	}
	if f, ok := rtr.FixedRoutes[path]; ok == true {
		f(w, r)
		return
	} else {
		for _, rr := range rtr.Routes {
			if rr.Pattern.MatchString(path) {
				rr.Func(w, r)
				return
			}
		}
	}
	rtr.NotFound(w, r)
}

// Parse parses r.URL.Query to extract the stored variables
func Parse(r *http.Request) (map[string]string, map[string][]string) {
	m := map[string]string{}
	for k, v := range r.URL.Query() {
		m[k] = v[0]
	}

	r.ParseForm()
	form := map[string][]string{}

	for k, values := range r.Form {
		mValues, exists := m[k]
		if !exists {
			form[k] = values
		} else if len(mValues) > 1 {
			for i, v := range values {
				if m[k] == v {
					form[k] = append(values[:i], values[i+1:]...)
					break
				}
			}
		}
	}

	return m, form
}
