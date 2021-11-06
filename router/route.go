package router

import (
	"net/http"
)

type Route struct {
	method  string
	host    string
	path    string
	handler http.Handler
}
