package applications

import "net/http"

type Application interface {
	LoggedinHandler(http.ResponseWriter, *http.Request, string, string)
	LoginHandler(http.ResponseWriter, *http.Request)
	getAccessToken(string) string
	getData(string) map[string]string
	CallbackHandler(http.ResponseWriter, *http.Request)
}

type AbstractApplication struct {
	Application
}
