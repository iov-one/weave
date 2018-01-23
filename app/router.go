package app

import (
	"regexp"

	"github.com/confio/weave"
)

var isRoute = regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString

type Router interface {
	AddRoute(r string, h weave.Handler)
	Route(path string) (h weave.Handler)
}

type route struct {
	r string
	h weave.Handler
}

type router struct {
	routes []route
}

func NewRouter() Router {
	return &router{
		routes: make([]route, 0),
	}
}

func (rtr *router) AddRoute(r string, h weave.Handler) {
	if !isRoute(r) {
		panic("route expressions can only contain alphanumeric characters or underscore")
	}
	rtr.routes = append(rtr.routes, route{r, h})
}

// TODO handle expressive matches.
func (rtr *router) Route(path string) (h weave.Handler) {
	for _, route := range rtr.routes {
		if route.r == path {
			return route.h
		}
	}
	return nil
}
