package applications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

// TODO - Change directories for openproject

type OpenProject struct {
	*AbstractApplication
}

func NewOpenProject() *OpenProject {
	a := &AbstractApplication{}
	r := &OpenProject{a}
	a.Application = r
	return r
}

func (op OpenProject) LoggedinHandler(w http.ResponseWriter, r *http.Request, Data string, AccessToken string) {
	if Data == "" {
		fmt.Fprint(w, "UNAUTHORIZED")
		return
	}

	w.Header().Set("Content-type", "application/json")
	var prettyJSON bytes.Buffer
	parser := json.Indent(&prettyJSON, []byte(Data), "", "\t")
	if parser != nil {
		log.Panic("JSON parse error")
	}

	fmt.Fprint(w, prettyJSON.String())
}

func (op OpenProject) getSecrets() *_SECRETS {

	clientID, exists := os.LookupEnv("OPENPROJECT_CLIENTID")
	if !exists {
		log.Fatal("Open Project Client ID not defined in .env file")
	}
	clientSecret, exists := os.LookupEnv("OPENPROJECT_SECRETID")
	if !exists {
		log.Fatal("Open Project Secret ID not defined in .env file")
	}

	return &_SECRETS{CLIENT_ID: clientID, SECRET_ID: clientSecret}

}

func (op OpenProject) LoginHandler(w http.ResponseWriter, r *http.Request) {
	clientID := OpenProject.getSecrets(OpenProject{}).CLIENT_ID

	redirectURL := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&scope=%s&redirect_uri=%s",
		clientID,
		"admin:org%20repo",
		"http://localhost:5002/github/login/callback", // TODO - Change root url, dinamic
	)
	http.Redirect(w, r, redirectURL, http.StatusMovedPermanently)
}

func (op OpenProject) getAccessToken(code string) string {
	clientID := OpenProject.getSecrets(OpenProject{}).CLIENT_ID
	secretID := OpenProject.getSecrets(OpenProject{}).SECRET_ID

	requestBodyMap := map[string]string{
		"client_id":     clientID,
		"client_secret": secretID,
		"code":          code,
	}

	requestJSON, _ := json.Marshal(requestBodyMap)

	req, err := http.NewRequest(
		"POST",
		"https://github.com/login/oauth/access_token",
		bytes.NewBuffer(requestJSON),
	)
	if err != nil {
		log.Panic("Request creation failed")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Panic("Request failed")
	}

	respbody, _ := io.ReadAll(resp.Body)

	type AccessTokenResponse struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
	}

	var finalresp AccessTokenResponse
	json.Unmarshal(respbody, &finalresp)

	return finalresp.AccessToken
}

func (op OpenProject) getData(accessToken string) string {
	req, err := http.NewRequest(
		"GET",
		"https://api.github.com/user",
		nil,
	)
	if err != nil {
		log.Panic("API Request creation failed")
	}

	authorizationHeaderValue := fmt.Sprintf("token %s", accessToken)
	req.Header.Set("Authorization", authorizationHeaderValue)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Panic("Request failed")
	}

	respbody, _ := io.ReadAll(resp.Body)

	return string(respbody)
}

func (op OpenProject) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	AccessToken := OpenProject.getAccessToken(OpenProject{}, code)
	Data := OpenProject.getData(OpenProject{}, AccessToken)

	OpenProject.LoggedinHandler(OpenProject{}, w, r, Data, AccessToken)
}
