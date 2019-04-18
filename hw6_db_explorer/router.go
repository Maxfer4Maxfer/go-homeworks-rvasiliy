package main

import (
	"fmt"
	"net/http"
	"strings"
)

type route struct {
	handler http.HandlerFunc
	path    string
}

type router struct {
	routes map[string][]*route
}

func newRouter() *router {
	r := &router{
		routes: make(map[string][]*route),
	}
	r.routes["GET"] = make([]*route, 0)
	r.routes["PUT"] = make([]*route, 0)
	r.routes["POST"] = make([]*route, 0)
	r.routes["DELETE"] = make([]*route, 0)
	return r
}

func (rt *router) setHandler(h http.HandlerFunc, path string, method string) error {
	path = strings.Trim(path, "/")
	switch method {
	case "GET":
		rt.routes["GET"] = append(rt.routes["GET"], &route{h, path})
	case "PUT":
		rt.routes["PUT"] = append(rt.routes["PUT"], &route{h, path})
	case "POST":
		rt.routes["POST"] = append(rt.routes["POST"], &route{h, path})
	case "DELETE":
		rt.routes["DELETE"] = append(rt.routes["DELETE"], &route{h, path})
	default:
		return fmt.Errorf("Unsupported method: %s", method)
	}
	return nil
}

func pathMatch(p1 []string, p2 []string) bool {
	// fmt.Println(p1, p2, len(p1), len(p2))
	switch len(p1) {
	case 1:
		if p1[0] == "" && p2[0] == "" {
			// fmt.Println("1")
			return true
		}
		if p1[0] != "" && p2[0] != "" && len(p2) == 1 {
			// fmt.Println("2")
			return true
		}
	default:
		if len(p1) == len(p2) {
			// fmt.Println("3")
			return true
		}
	}
	return false
}

func (rt *router) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	urlPath := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

	for _, route := range rt.routes[r.Method] {
		rPath := strings.Split(route.path, "/")
		if pathMatch(rPath, urlPath) {
			route.handler.ServeHTTP(w, r)
			return
		}
	}
}
