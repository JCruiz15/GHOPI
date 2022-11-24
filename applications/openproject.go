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
	"web-service-gin/functions"

	"github.com/Jeffail/gabs/v2"
	"github.com/google/uuid"
)

// var randomS uuid.UUID

// TODO - Change obtention of clientDI and states

type Openproject struct {
	*AbstractApplication
	states   map[string]string
	clientID string
	secretID string
}

func NewOpenproject() *Openproject {
	a := &AbstractApplication{}
	s := make(map[string]string)

	clientID, exists := os.LookupEnv("OPENPROJECT_CLIENTID")
	if !exists {
		log.Fatal("OpenProject Client ID not defined in .env file")
	}
	clientSecret, exists := os.LookupEnv("OPENPROJECT_SECRETID")
	if !exists {
		log.Fatal("OpenProject Secret ID not defined in .env file")
	}
	r := &Openproject{a, s, clientID, clientSecret}

	a.Application = r
	return r
}

func (op *Openproject) LoggedinHandler(w http.ResponseWriter, r *http.Request, Data map[string]string, Token string) {
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

		config.Set(Data["name"], "openproject-user")
		config.Set(Token, "openproject-token")

		f, err := os.Create(config_path)
		functions.Check(err, "Error")
		defer f.Close()
		f.Write(config.BytesIndent("", "\t"))
	}

	// w.Header().Set("Content-type", "application/json")
	// var prettyJSON bytes.Buffer
	// parser := json.Indent(&prettyJSON, []byte(Data), "", "\t")
	// if parser != nil {
	// 	log.Panic("JSON parse error")
	// }

	http.Redirect(w, r, "/", http.StatusMovedPermanently)
}

func (op *Openproject) LoginHandler(w http.ResponseWriter, r *http.Request) {

	s := uuid.New().String()
	op.states[fmt.Sprint(len(op.states))] = s

	redirectURL := fmt.Sprintf(
		"http://localhost:8080/oauth/authorize?response_type=code&client_id=%s&scope=%s&redirect_uri=%s&prompt=consent",
		op.clientID,
		"api_v3",
		"http://localhost:5002/op/login/callback",
		//fmt.Sprintf("http://localhost:5002/op/login/callback?state=%s", s),
	)

	http.Redirect(w, r, redirectURL, http.StatusMovedPermanently)
}

func (op *Openproject) getAccessToken(code string) string {
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
	req.Header.Set("Content-Length", strconv.Itoa(len(requestBodyEnc)))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Panic("Request failed")
	}

	respbody, _ := io.ReadAll(resp.Body)

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

func (op *Openproject) getData(accessToken string) map[string]string {
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

func (op *Openproject) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	// security := r.URL.Query().Get("state")
	// secured := false
	// for key, value := range op.states {
	// 	if value == security {
	// 		fmt.Println("Secure transaction")
	// 		secured = true
	// 		delete(op.states, key)
	// 	}
	// }
	// fmt.Println(secured)
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

	AccessToken := op.getAccessToken(code)
	Data := op.getData(AccessToken)

	op.LoggedinHandler(w, r, Data, AccessToken)
}
