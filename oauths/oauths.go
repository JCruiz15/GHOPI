/*
oauths contains all the functions needed to do the OAuth protocol with Open Project and GitHub,
and creates instances of Open Project and GitHub protocols separatedly.

· oauthsGithub.go: Defines the functions to do the autentication protocol with GitHub.

· oauthOpenProject.go: Defines the functions to do the autentication protocol with Open Project.
*/
package oauths

import "net/http"

/*
Interface that defines which functions will be shared between every instance and which are needed to do the OAuth protocol.
*/
type Application interface {
	LoggedinHandler(http.ResponseWriter, *http.Request, map[string]string, string, string, string)
	LoginHandler(http.ResponseWriter, *http.Request)
	getAccessToken(string, string) (string, string, string)
	getData(string) map[string]string
	CallbackHandler(http.ResponseWriter, *http.Request)
}

type AbstractApplication struct {
	Application
}
