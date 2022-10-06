package applications

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// TODO - Change directories for openproject

type OpenProject struct {
	*AbstractApplication
	states   map[string]string
	clientID string
	secretID string
}

func NewOpenProject() *OpenProject {
	a := &AbstractApplication{}
	s := make(map[string]string)

	clientID, exists := os.LookupEnv("OPENPROJECT_CLIENTID")
	if !exists {
		log.Fatal("Open Project Client ID not defined in .env file")
	}
	clientSecret, exists := os.LookupEnv("OPENPROJECT_SECRETID")
	if !exists {
		log.Fatal("Open Project Secret ID not defined in .env file")
	}

	r := &OpenProject{a, s, clientID, clientSecret}
	a.Application = r
	return r
}

func (op OpenProject) LoggedinHandler(w http.ResponseWriter, r *http.Request, Data string, Token string) {
	if Data == "" {
		fmt.Fprint(w, "UNAUTHORIZED")
		return
	}

	http.Redirect(w, r, "http://localhost:5002", http.StatusMovedPermanently)
}

func (op OpenProject) LoginHandler(w http.ResponseWriter, r *http.Request) {
	s := uuid.New().String()
	op.states[fmt.Sprint(len(op.states))] = s
	redirectURL := fmt.Sprintf(
		"http://localhost:8080/oauth/authorize?response_type=code&client_id=%s&scope=%s&redirect_uri=%s&prompt=consent", // TODO - Change localhost, for user input
		op.clientID,
		"api_v3",
		fmt.Sprintf("http://localhost:5002/github/login/callback&state=%s", s), // TODO - Change root url, dinamic
	)
	http.Redirect(w, r, redirectURL, http.StatusMovedPermanently)
}

func (op OpenProject) getAccessToken(code string) string {
	requestBody := url.Values{}
	requestBody.Set("grant_type", "authorization_code")
	requestBody.Set("client_id", op.clientID)
	requestBody.Set("client_secret", op.secretID)
	requestBody.Set("code", code)
	requestBody.Set("redirect_uri", "http://localhost:5002/op/login/callback") // TODO - Change root url, dinamic
	requestBodyEnc := requestBody.Encode()

	req, err := http.NewRequest(
		"POST",
		"http://localhost:8080/oauth/token",
		strings.NewReader(requestBodyEnc),
	)
	if err != nil {
		log.Panic("Request creation failed")
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(requestBodyEnc)))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Panic("Request failed")
	}

	respbody, _ := io.ReadAll(resp.Body)
	fmt.Println(resp.StatusCode)

	type AccessTokenResponse struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpireTime   string `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		Scope        string `json:"scope"`
		Date         string `json:"created_at"`
	}

	var finalresp AccessTokenResponse
	json.Unmarshal(respbody, &finalresp)

	return finalresp.AccessToken
}

func (op OpenProject) getData(accessToken string) map[string]string {
	req, err := http.NewRequest(
		"GET",
		"http://localhost:8080/api/v3/users/me",
		nil,
	)
	if err != nil {
		log.Panic("API Request creation failed")
	}

	authorizationHeaderValue := fmt.Sprintf("Bearer %s", accessToken)
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

func (op OpenProject) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	security := r.URL.Query().Get("state")
	secured := false
	for key, value := range op.states {
		if value == security {
			fmt.Println("Secure transaction")
			secured = true
			delete(op.states, key)
		}
	}
	if !secured {
		log.Panic("Third party system connection")
	}
	AccessToken := OpenProject.getAccessToken(OpenProject{}, code)
	Data := OpenProject.getData(OpenProject{}, AccessToken)
	OpenProject.LoggedinHandler(OpenProject{}, w, r, Data["user"], AccessToken)
}
