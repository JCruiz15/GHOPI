package applications

import "net/http"

type _SECRETS struct {
	CLIENT_ID string
	SECRET_ID string
}

type Application interface {
	LoggedinHandler(http.ResponseWriter, *http.Request, string, string)
	getSecrets() *_SECRETS
	LoginHandler(http.ResponseWriter, *http.Request)
	getAccessToken(string) string
	getData(string) string
	CallbackHandler(http.ResponseWriter, *http.Request)
}

type AbstractApplication struct {
	Application
}
