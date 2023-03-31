package oauths

import "net/http"

type Application interface {
	LoggedinHandler(http.ResponseWriter, *http.Request, map[string]string, string)
	LoginHandler(http.ResponseWriter, *http.Request)
	getAccessToken(string, string) string
	getData(string) map[string]string
	CallbackHandler(http.ResponseWriter, *http.Request)
}

type AbstractApplication struct {
	Application
}