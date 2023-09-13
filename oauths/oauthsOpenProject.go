package oauths

import (
	"GHOPI/utils"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/buger/jsonparser"
	log "github.com/sirupsen/logrus"
)

/*
Map that stores the client and secret IDs
*/
type Openproject struct {
	*AbstractApplication
	states   map[string]string
	clientID string
	secretID string
}

/*
Function NewOpenproject() instantiates the Openproject application with the client and secret IDs and returns an instance of the Application interface.
*/
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

/*
Function LoggedinHandler uses the Data from the callback of Open Project and stores the Open Project user and token in 'config.json'.
*/
func (op *Openproject) LoggedinHandler(
	w http.ResponseWriter,
	r *http.Request,
	Data map[string]string,
	Token string,
	refreshToken string,
	expirationDate string,
) {
	if Data == nil {
		fmt.Fprint(w, "UNAUTHORIZED")
		return
	} else {
		var config *gabs.Container

		if _, err := os.Stat(utils.Config_path); err == nil {
			config, err = gabs.ParseJSONFile(utils.Config_path)
			utils.Check(err, "Error", "Error 500. Config file could not be read")
		} else {
			config = gabs.New()
		}

		config.Set(Data["name"], "openproject-user")
		config.Set(Token, "openproject-token")
		config.Set(refreshToken, "openproject-refresh-token")
		config.Set(expirationDate, "openproject-expiration")

		f, err := os.Create(utils.Config_path)
		utils.Check(err, "Error", "Error 500. Config file could not be created. Config file may not exists")
		defer f.Close()
		f.Write(config.BytesIndent("", "\t"))
	}

	go utils.CheckCustomFields()
	subpath := utils.GetSubpath()
	http.Redirect(w, r, fmt.Sprintf("%s/config-openproject", subpath), http.StatusMovedPermanently)
}

/*
Function LoginHandler creates the URL that redirects to Open Project with the permissions needed for GHOPI.
*/
func (op *Openproject) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var URL string = "http://localhost:8089"
	subpath := utils.GetSubpath()
	if strings.Contains(r.Host, "localhost") {
		URL = fmt.Sprintf("http://%s%s", r.Host, subpath)
	} else {
		URL = fmt.Sprintf("https://%s%s", r.Host, subpath)
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

/*
Function getAccessToken uses the information from the callback given by Open Project to obtain the access token.

It returns the access token as a string.
*/
func (op *Openproject) getAccessToken(code string, URL string) (string, string, string) {
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
		log.Panic(err.Error())
	}

	respbody, _ := io.ReadAll(resp.Body)

	type AccessTokenResponse struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpireTime   int64  `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		Scope        string `json:"scope"`
		Date         string `json:"created_at"`
	}

	var finalresp AccessTokenResponse
	json.Unmarshal(respbody, &finalresp)
	twohours, _ := time.ParseDuration(fmt.Sprintf("%ds", finalresp.ExpireTime))
	expirationDate := time.Now().Add(twohours).Format("2006-01-02T15:04:05Z")

	return finalresp.AccessToken, finalresp.RefreshToken, expirationDate
}

/*
Function getData uses access token to obtain information about the Open Project user that will be used in LoggedinHandler

It returns the Data as a map[string]string.
*/
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
	json.Unmarshal(respbody, &jsonMap)
	return jsonMap
}

/*
Function CallbackHandler is the function that receives the information from GitHub and calls the function LoggedI
*/
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

	var URL string = "http://localhost:8089"
	subpath := utils.GetSubpath()
	if strings.Contains(r.Host, "localhost") {
		URL = fmt.Sprintf("http://%s%s", r.Host, subpath)
	} else {
		URL = fmt.Sprintf("https://%s%s", r.Host, subpath)
	}

	AccessToken, RefreshToken, expirationDate := op.getAccessToken(code, URL)
	Data := op.getData(AccessToken)

	op.LoggedinHandler(w, r, Data, AccessToken, RefreshToken, expirationDate)
}

func (op *Openproject) RefreshAuth() {
	utils.OP_url = utils.GetOPuri()

	f, err := os.Open(".config/config.json")
	if utils.Check(err, "error", "Error 500. Config file could not be opened. Config file may not exist") {
		return
	}
	configFile, _ := io.ReadAll(f)
	refresh_token, err := jsonparser.GetString(configFile, "openproject-refresh-token")
	if utils.Check(err, "error", "Error 404: Open Project refresh token missing in config file. Log in again in Open Project") {
		return
	}
	defer f.Close()

	requestBody := url.Values{}
	requestBody.Set("grant_type", "refresh_token")
	requestBody.Set("client_id", op.clientID)
	requestBody.Set("client_secret", op.secretID)
	requestBody.Set("refresh_token", refresh_token)
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
		log.Panic(err.Error())
	}

	respbody, _ := io.ReadAll(resp.Body)
	type AccessTokenResponse struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpireTime   int64  `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		Scope        string `json:"scope"`
		Date         string `json:"created_at"`
	}

	var finalresp AccessTokenResponse
	json.Unmarshal(respbody, &finalresp)

	var config *gabs.Container
	if _, err := os.Stat(utils.Config_path); err == nil {
		config, err = gabs.ParseJSONFile(utils.Config_path)
		utils.Check(err, "Error", "Error 500. Config file could not be read")
	} else {
		config = gabs.New()
	}

	twohours, _ := time.ParseDuration(fmt.Sprintf("%ds", finalresp.ExpireTime))
	expirationDate := time.Now().Add(twohours).Format("2006-01-02T15:04:05Z")

	config.Set(finalresp.AccessToken, "openproject-token")
	config.Set(finalresp.RefreshToken, "openproject-refresh-token")
	config.Set(expirationDate, "openproject-expiration")

	f, err = os.Create(utils.Config_path)
	if utils.Check(err, "Error", "Error 500. Config file could not be created. Config file may not exists") {
		return
	}
	defer f.Close()
	f.Write(config.BytesIndent("", "\t"))
	log.Info("Open Project access token was successfully refreshed")
}
