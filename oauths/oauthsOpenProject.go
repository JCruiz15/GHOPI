package oauths

import (
	"GHOPI/utils"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/Jeffail/gabs/v2"
)

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

		config.Set(Data["name"], "openproject-user") // TODO - errcheck
		config.Set(Token, "openproject-token")       // TODO - errcheck

		f, err := os.Create(utils.Config_path)
		utils.Check(err, "Error", "Error 500. Config file could not be created. Config file may not exists")
		defer f.Close()                       // TODO - errcheck
		f.Write(config.BytesIndent("", "\t")) // TODO - errcheck
	}

	go utils.CheckCustomFields()

	http.Redirect(w, r, "/config-openproject", http.StatusMovedPermanently)
}

func (op *Openproject) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var URL string = "http://localhost:5050"
	if strings.Contains(r.Host, "localhost") {
		URL = fmt.Sprintf("http://%s", r.Host)
	} else {
		URL = fmt.Sprintf("https://%s", r.Host)
	}
	// s := uuid.New().String()
	// op.states[fmt.Sprint(len(op.states))] = s

	utils.OP_url = utils.GetOPuri()

	redirectURL := fmt.Sprintf(
		"%s/oauth/authorize?response_type=code&client_id=%s&scope=%s&redirect_uri=%s&prompt=consent",
		utils.OP_url,
		op.clientID,
		"api_v3",
		fmt.Sprintf("%s/op/login/callback", URL),
		//fmt.Sprintf("http://localhost:5002/op/login/callback?state=%s", s),
	)

	http.Redirect(w, r, redirectURL, http.StatusMovedPermanently)
}

func (op *Openproject) getAccessToken(code string, URL string) string {
	utils.OP_url = utils.GetOPuri()

	requestBody := url.Values{}
	requestBody.Set("grant_type", "authorization_code")
	requestBody.Set("client_id", op.clientID)
	requestBody.Set("client_secret", op.secretID)
	requestBody.Set("code", code)
	requestBody.Set("redirect_uri", fmt.Sprintf("%s/op/login/callback", URL))
	requestBodyEnc := requestBody.Encode()

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/oauth/token", utils.OP_url),
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
	json.Unmarshal(respbody, &finalresp) // TODO - errcheck

	return finalresp.AccessToken
}

func (op *Openproject) getData(accessToken string) map[string]string {
	utils.OP_url = utils.GetOPuri()

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/api/v3/users/me", utils.OP_url),
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
	json.Unmarshal(respbody, &jsonMap) // TODO - errcheck
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

	var URL string = "http://localhost:5050"
	if strings.Contains(r.Host, "localhost") {
		URL = fmt.Sprintf("http://%s", r.Host)
	} else {
		URL = fmt.Sprintf("https://%s", r.Host)
	}

	AccessToken := op.getAccessToken(code, URL)
	Data := op.getData(AccessToken)

	op.LoggedinHandler(w, r, Data, AccessToken)
}
