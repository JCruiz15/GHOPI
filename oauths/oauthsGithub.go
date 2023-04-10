package oauths

import (
	"GHOPI/utils"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/Jeffail/gabs/v2"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// var randomS uuid.UUID

type Github struct {
	*AbstractApplication
	states   map[string]string
	clientID string
	secretID string
}

func NewGithub() *Github {
	a := &AbstractApplication{}
	s := make(map[string]string)

	clientID, exists := os.LookupEnv("GITHUB_CLIENTID")
	if !exists {
		log.Fatal("Github Client ID not defined in .env file")
	}
	clientSecret, exists := os.LookupEnv("GITHUB_SECRETID")
	if !exists {
		log.Fatal("Github Secret ID not defined in .env file")
	}
	r := &Github{a, s, clientID, clientSecret}

	a.Application = r
	return r
}

func (gh Github) LoggedinHandler(w http.ResponseWriter, r *http.Request, Data map[string]string, Token string) {
	fmt.Println(r.URL.String())
	if Data == nil {
		fmt.Fprint(w, "UNAUTHORIZED") // TODO - errcheck
		return
	} else {
		var config *gabs.Container

		if _, err := os.Stat(utils.Config_path); err == nil {
			config, err = gabs.ParseJSONFile(utils.Config_path)
			utils.Check(err, "Error", "Error 500. Config file could not be read")
		} else {
			config = gabs.New()
		}

		config.Set(Data["login"], "github-user") // TODO - errcheck
		config.Set(Token, "github-token")        // TODO - errcheck

		f, err := os.Create(utils.Config_path)
		utils.Check(err, "Error", "Error 500. Config file could not be created. Config file may not exists")
		defer f.Close()                       // TODO - errcheck
		f.Write(config.BytesIndent("", "\t")) // TODO - errcheck
	}

	log.Info("Github log in has been successful")
	http.Redirect(w, r, "/config-github", http.StatusMovedPermanently)
}

func (gh *Github) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var URL string
	if strings.Contains(r.Host, "localhost") {
		URL = fmt.Sprintf("http://%s", r.Host)
	} else {
		URL = fmt.Sprintf("https://%s", r.Host)
	}
	s := uuid.New().String()
	gh.states[fmt.Sprint(len(gh.states))] = s

	redirectURL := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&scope=%s&redirect_uri=%s",
		gh.clientID,
		"admin:org%20repo%20admin:org_hook",
		fmt.Sprintf("%s/github/login/callback&state=%s", URL, s),
	)

	http.Redirect(w, r, redirectURL, http.StatusMovedPermanently)
}

func (gh Github) getAccessToken(code string, URL string) string {
	requestBodyMap := map[string]string{
		"client_id":     gh.clientID,
		"client_secret": gh.secretID,
		"code":          code,
	}

	requestJSON, _ := json.Marshal(requestBodyMap)

	req, err := http.NewRequest(
		"POST",
		"https://github.com/login/oauth/access_token",
		bytes.NewBuffer(requestJSON),
	)
	utils.Check(err, "error", "Github API request creation to get oauth access token failed")

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	utils.Check(err, "error", "Github API call to get oauth access token failed")

	respbody, _ := io.ReadAll(resp.Body)

	type AccessTokenResponse struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
	}

	var finalresp AccessTokenResponse
	json.Unmarshal(respbody, &finalresp) // TODO - errcheck
	return finalresp.AccessToken
}

func (gh Github) getData(accessToken string) map[string]string {
	req, err := http.NewRequest(
		"GET",
		"https://api.github.com/user",
		nil,
	)
	utils.Check(err, "error", "Github API request creation to get oauth access token failed")

	authorizationHeaderValue := fmt.Sprintf("token %s", accessToken)
	req.Header.Set("Authorization", authorizationHeaderValue)

	resp, err := http.DefaultClient.Do(req)
	utils.Check(err, "error", "Github API call to get oauth access token failed")

	respbody, _ := io.ReadAll(resp.Body)
	var jsonMap map[string]string
	json.Unmarshal(respbody, &jsonMap) // TODO - errcheck
	return jsonMap
}

func (gh Github) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	security := r.URL.Query().Get("state")
	for key, value := range gh.states {
		if value == security {
			log.Info("Github log in is secure")
			delete(gh.states, key)
		}
	}

	URL := fmt.Sprintf("https://%s", r.Host)
	fmt.Println(URL)

	AccessToken := gh.getAccessToken(code, URL)
	Data := gh.getData(AccessToken)

	gh.LoggedinHandler(w, r, Data, AccessToken)
}
