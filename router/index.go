package router

import (
	"errors"
	"net/url"
	"strings"
)

var (
	ErrMethodNotAllowed = errors.New("method is not allowed for this router")
	ErrBadPath          = errors.New("every path definition must conform to [Method]:[Url]")
	ErrNilHandler       = errors.New("nill handler provided")
)

func isValidMethod(method string) bool {
	for _, m := range []string{"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS"} {
		if strings.EqualFold(m, method) {
			return true
		}
	}
	return false
}

func tokenize(path string) (string, string, string) {
	paths := strings.Split(path, ":")
	if len(paths) != 2 {
		panic(ErrBadPath)
	}

	pathMethod := paths[0]
	if !isValidMethod(pathMethod) {
		panic(ErrMethodNotAllowed)
	}

	pathUrl := paths[1]
	pathUrl = strings.TrimPrefix(pathUrl, "/")
	pathUrl = strings.TrimSuffix(pathUrl, "/")

	if pathUrl == "" {
		pathUrl = "/"
	}

	u, err := url.Parse(pathUrl)
	if err != nil {
		panic(err)
	}

	return pathMethod, u.Host, u.Path
}

func New() *Router {
	return &Router{}
}
