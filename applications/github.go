package applications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// var randomS uuid.UUID

// TODO - Change obtention of clientDI and states

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

func (gh Github) LoggedinHandler(w http.ResponseWriter, r *http.Request, Data string, Token string) {
	if Data == "" {
		fmt.Fprint(w, "UNAUTHORIZED")
		return
	}

	// w.Header().Set("Content-type", "application/json")
	// var prettyJSON bytes.Buffer
	// parser := json.Indent(&prettyJSON, []byte(Data), "", "\t")
	// if parser != nil {
	// 	log.Panic("JSON parse error")
	// }

	http.Redirect(w, r, "http://localhost:5002", http.StatusMovedPermanently)
}

func (gh *Github) LoginHandler(w http.ResponseWriter, r *http.Request) {

	s := uuid.New().String()
	gh.states[fmt.Sprint(len(gh.states))] = s

	redirectURL := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&scope=%s&redirect_uri=%s",
		gh.clientID,
		"admin:org%20repo",
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
		fmt.Print(key)
		fmt.Print(" - ")
		fmt.Println(value)
		fmt.Println(security)
		fmt.Println(value == security)
		if value == security {
			fmt.Println("Secure transaction")
			secured = true
			delete(gh.states, key)
		}
	}
	fmt.Println(secured)
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

	fmt.Println(Data)
	// Escribir id del usuario de github
	wd, _ := os.Getwd()
	path := filepath.Join(wd, ".tokens", fmt.Sprintf("%s-github", Data["login"]))
	f, err := os.Create(path)
	if err != nil {
		fmt.Println(err.Error())
	}
	defer f.Close()

	n, _ := f.Write([]byte(AccessToken))
	fmt.Println(n)

	gh.LoggedinHandler(w, r, Data["login"], AccessToken)
}
