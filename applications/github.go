package applications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"web-service-gin/functions"

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
	if Data == nil {
		fmt.Fprint(w, "UNAUTHORIZED")
		return
	} else {
		var config *gabs.Container
		config_path := ".config/config.json"

		if _, err := os.Stat(config_path); err == nil {
			config, err = gabs.ParseJSONFile(config_path)
			functions.Check(err, "Error")
		} else {
			config = gabs.New()
		}

		config.Set(Data["login"], "github-user")
		config.Set(Token, "github-token")

		f, err := os.Create(config_path)
		functions.Check(err, "Error")
		defer f.Close()
		f.Write(config.BytesIndent("", "\t"))
	}

	http.Redirect(w, r, "/", http.StatusMovedPermanently)
}

func (gh *Github) LoginHandler(w http.ResponseWriter, r *http.Request) {

	s := uuid.New().String()
	gh.states[fmt.Sprint(len(gh.states))] = s

	redirectURL := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&scope=%s&redirect_uri=%s",
		gh.clientID,
		"admin:org%20repo%20admin:org_hook",
		// "http://localhost:5002/github/login/callback",
		fmt.Sprintf("http://localhost:5002/github/login/callback&state=%s", s), // TODO - Change root url, dinamic
	)

	http.Redirect(w, r, redirectURL, http.StatusMovedPermanently)
}

func (gh Github) getAccessToken(code string) string {
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

func (gh Github) getData(accessToken string) map[string]string {
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
	var jsonMap map[string]string
	json.Unmarshal(respbody, &jsonMap)
	return jsonMap
}

func (gh Github) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	security := r.URL.Query().Get("state")
	secured := false
	for key, value := range gh.states {
		if value == security {
			fmt.Println("Secure transaction")
			secured = true
			delete(gh.states, key)
		}
	}
	fmt.Printf("secured? %t\n", secured)
	// if !secured {
	// 	_, err := os.Open(".tokens.txt")
	// 	if err != nil {
	// 		fmt.Println("Error - no access")
	// 		return
	// 		// log.Panic("Third party system access attempt")
	// 	}
	// 	// log.Panic("Third party system connection")
	// 	fmt.Print("ERROR")
	// 	fmt.Println(gh.states)
	// }

	AccessToken := gh.getAccessToken(code)
	Data := gh.getData(AccessToken)

	gh.LoggedinHandler(w, r, Data, AccessToken)
}
