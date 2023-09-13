/*Package main implements all the logic to launch the API and the web UI*/
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"GHOPI/oauths"
	"GHOPI/utils"

	"github.com/buger/jsonparser"
	"github.com/joho/godotenv"

	"github.com/Jeffail/gabs/v2"
	log "github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
	"gopkg.in/natefinch/lumberjack.v2"
)

// TODO - IMPLEMENT API KEY METHOD FOR SOME FUNCTIONS

/*Function init sets up logs format and log file metadata. Also loads .env file.*/
func init() {
	log.SetFormatter(&easy.Formatter{
		TimestampFormat: "02/01/2006-15:04:05",
		LogFormat:       "[%lvl%]\t%time%\t--\t%msg%\n",
	})

	log.SetOutput(io.MultiWriter(
		&lumberjack.Logger{
			Filename:   ".config/outputs.log",
			MaxSize:    1,
			MaxBackups: 5,
			MaxAge:     30,
			Compress:   true,
		},
		os.Stdout,
	))

	if err := godotenv.Load(); err != nil {
		environment := os.Environ()
		for _, variable_tuple := range environment {
			keyval := strings.Split(variable_tuple, "=")
			_, ok := os.LookupEnv(keyval[0])
			if !ok {
				log.Fatal("Environment variables are missing. Fill up a .env file or check your docker environment variables")
			}
		}
	}
}

/*Function main handles every endpoint for the app and launches it*/
func main() {
	var github oauths.Github = *oauths.NewGithub()
	var openproject oauths.Openproject = *oauths.NewOpenproject()
	subpath := utils.GetSubpath()

	fileServer := http.FileServer(http.Dir("./static"))

	http.Handle(fmt.Sprintf("%s/static/", subpath), http.StripPrefix(fmt.Sprintf("%s/static", subpath), fileServer))

	http.HandleFunc(fmt.Sprintf("%s/", subpath), index)
	http.HandleFunc(fmt.Sprintf("%s/docs", subpath), instructions)
	http.HandleFunc(fmt.Sprintf("%s/logs", subpath), logs)
	http.HandleFunc(fmt.Sprintf("%s/config-openproject", subpath), configOP)
	http.HandleFunc(fmt.Sprintf("%s/config-github", subpath), configGH)

	http.HandleFunc(fmt.Sprintf("%s/api/get-logs", subpath), getLogs)
	http.HandleFunc(fmt.Sprintf("%s/api/get-config", subpath), getConfig)
	http.HandleFunc(fmt.Sprintf("%s/api/openproject", subpath), PostOpenProject)
	http.HandleFunc(fmt.Sprintf("%s/api/github", subpath), PostGithub)
	http.HandleFunc(fmt.Sprintf("%s/api/refresh", subpath), refreshProxy)
	http.HandleFunc(fmt.Sprintf("%s/api/reset/refresh", subpath), resetRefreshDate)
	http.HandleFunc(fmt.Sprintf("%s/api/check-custom-fields", subpath), checkFields)

	http.HandleFunc(fmt.Sprintf("%s/github/login", subpath), github.LoginHandler)
	http.HandleFunc(fmt.Sprintf("%s/github/login/callback", subpath), github.CallbackHandler)
	http.HandleFunc(fmt.Sprintf("%s/github/loggedin", subpath),
		func(w http.ResponseWriter, r *http.Request) {
			github.LoggedinHandler(w, r, nil, "", "", "")
		})
	http.HandleFunc(fmt.Sprintf("%s/github/webhook", subpath), githubWebhook)

	http.HandleFunc(fmt.Sprintf("%s/op/login", subpath), openproject.LoginHandler)
	http.HandleFunc(fmt.Sprintf("%s/op/login/callback", subpath), openproject.CallbackHandler)
	http.HandleFunc(fmt.Sprintf("%s/op/loggedin", subpath),
		func(w http.ResponseWriter, r *http.Request) {
			openproject.LoggedinHandler(w, r, nil, "", "", "")
		})
	http.HandleFunc(fmt.Sprintf("%s/op/save-url", subpath), saveOPurl)
	http.HandleFunc(fmt.Sprintf("%s/-/health", subpath), func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	p, exist := os.LookupEnv("PORT")
	if !exist {
		p = "8080"
	}
	port, _ := strconv.Atoi(p)
	log.Info(fmt.Sprintf("Application running on port %d", port))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

/*Function renderTemplate executes html templates to show them on the website*/
func renderTemplate(w http.ResponseWriter, tmpl string, data map[string]interface{}) {
	t := template.Must(template.New(tmpl).ParseFiles("templates/base.html", "templates/"+tmpl+".html"))
	t.ExecuteTemplate(w, "base", data)
}

/*Function index renders index.html template*/
func index(w http.ResponseWriter, _ *http.Request) {
	subpath := utils.GetSubpath()
	data := map[string]interface{}{
		"subpath": subpath,
	}
	renderTemplate(w, "index", data)
	// if !utils.CheckConnectionGithub() || !utils.CheckConnectionOpenProject() {
	// 	log.Warn("Github or Open Project token is not working or it has expired. Log in sign in to use the app.")
	// }
}

/*Function instructions renders instructions.html template*/
func instructions(w http.ResponseWriter, _ *http.Request) {
	subpath := utils.GetSubpath()
	data := map[string]interface{}{
		"subpath": subpath,
	}
	renderTemplate(w, "instructions", data)
}

/*Function configOP renders openproject_config.html template*/
func configOP(w http.ResponseWriter, _ *http.Request) {
	subpath := utils.GetSubpath()
	data := map[string]interface{}{
		"subpath": subpath,
	}
	renderTemplate(w, "openproject_config", data)
	// if !utils.CheckConnectionGithub() || !utils.CheckConnectionOpenProject() {
	// 	log.Warn("Github or Open Project token is not working or it has expired. Log in sign in to use the app.")
	// }
}

/*Function configGH renders github_config.html template*/
func configGH(w http.ResponseWriter, _ *http.Request) {
	subpath := utils.GetSubpath()
	data := map[string]interface{}{
		"subpath": subpath,
	}
	renderTemplate(w, "github_config", data)
	// if !utils.CheckConnectionGithub() || !utils.CheckConnectionOpenProject() {
	// 	log.Warn("Github or Open Project token is not working or it has expired. Log in sign in to use the app.")
	// }
}

/*Function logs renders log.html template*/
func logs(w http.ResponseWriter, _ *http.Request) {
	subpath := utils.GetSubpath()
	data := map[string]interface{}{
		"subpath": subpath,
	}
	renderTemplate(w, "log", data)
	// if !utils.CheckConnectionGithub() || !utils.CheckConnectionOpenProject() {
	// 	log.Warn("Github or Open Project token is not working or it has expired. Log in sign in to use the app.")
	// }
}

/*Function getLogs reads output.txt and returns its information as an html plain text*/
func getLogs(w http.ResponseWriter, _ *http.Request) {
	file, err := os.Open(".config/outputs.log")
	utils.Check(err, "error", "Error 500. Logs file could not be open")
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// optionally, resize scanner's capacity for lines over 64K, see next example
	var lines [][]byte
	for scanner.Scan() {
		l := scanner.Text()
		bracket1 := strings.Index(l, "[") + 1
		bracket2 := strings.Index(l, "]") + 1
		endDate := strings.Index(l, "\t--\t")
		var formated_line string
		var type_color string
		switch l[bracket1:bracket2] {
		case "INFO]":
			type_color = "cyan"
		case "WARNING]":
			type_color = "orange"
		case "ERROR]":
			type_color = "red"
		case "FATAL]":
			type_color = "darkviolet"
		default:
			type_color = "white"
		}
		if endDate == -1 {
			formated_line = `<span style="color:` + type_color + `; font-family: monospace;">` + l[:bracket1] + l[bracket1:bracket2] + `</span>` + l[bracket2:]
		} else {
			formated_line = `<span style="color:` + type_color + `; font-family: monospace;">` + l[:bracket1] + l[bracket1:bracket2] + `</span><span id="date" style="color:grey; font-family: monospace;">` + l[bracket2:endDate] + `</span>` + l[endDate:]
		}
		lines = append(lines, []byte(formated_line), []byte("<br>"))
	}
	if err := scanner.Err(); err != nil {
		log.Error("Error 500. Error while reading log file. The file may have not been found")
	}

	for i := len(lines) - 1; i >= 0; i-- {
		w.Write(lines[i])
	}
}

/*Function getConfig reads .config/config.json file and sends the information as a POST without the tokens information*/
func getConfig(w http.ResponseWriter, _ *http.Request) {
	var config *gabs.Container

	if _, err := os.Stat(utils.Config_path); err == nil {
		config, err = gabs.ParseJSONFile(utils.Config_path)
		utils.Check(err, "error", "Error 500. Config file could not be read")
	} else {
		config = gabs.New()
	}
	config.Delete("github-token")
	config.Delete("openproject-token")

	w.Header().Set("Content-Type", "application/json")
	w.Write(config.EncodeJSON())
}

/*Function githubWebhook uses GitHub API to create a webhook with the organization sent*/
func githubWebhook(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var orgName string = ""

	if r.Method == "POST" {
		jsonOrgName, _ := io.ReadAll(r.Body)
		var errorJSON error
		orgName, errorJSON = jsonparser.GetString(jsonOrgName, "organizationName")
		utils.Check(errorJSON, "error", "Error 500. The server did not received organization name correctly")
	}
	subpath := utils.GetSubpath()

	if orgName != "" {
		URL := fmt.Sprintf("https://%s%s%s", r.Host, subpath, "/api/github")

		body := map[string]interface{}{
			"name": "web",
			"events": [9]string{
				"create",
				"delete",
				"membership",
				"pull_request",
				"pull_request_review",
				"pull_request_review_comment",
				"pull_request_review_thread",
				"push",
				"repository",
			},
			"active": true,
			"config": map[string]string{
				"url":          URL,
				"content_type": "json",
			},
		}
		bJSON, _ := json.Marshal(body)

		req, err := http.NewRequest(
			"POST",
			fmt.Sprintf("https://api.github.com/orgs/%s/hooks", orgName),
			bytes.NewBuffer(bJSON),
		)
		if err != nil {
			log.Fatal("Github API Request creation failed on webhook creation")
		}

		req.Header.Set("Accept", "application/vnd.github+json")

		f, err := os.Open(".config/config.json")
		utils.Check(err, "error", "Error 500. Config file could not be opened. Config file may not exist")
		config, _ := io.ReadAll(f)
		token, _ := jsonparser.GetString(config, "github-token")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatal("Github API Request failed on webhook creation. This issue is due to Github availability")
		}
		r, _ := io.ReadAll(resp.Body)

		name, getErr := jsonparser.GetString(r, "name")
		utils.Check(getErr, "error", "Error on reading if the webhook was correctly created")

		if name == "web" {
			id, _ := jsonparser.GetInt(r, "id")
			log.Info(fmt.Sprintf("Github webhook created successfully with id %d", id))
		}
		w.Write(r)

	} else {
		output := map[string]interface{}{
			"message": "Webhook creation failed: Wrong http method or Organization name not received",
			"status":  http.StatusInternalServerError,
		}
		resul, _ := json.Marshal(output)
		w.Write(resul)
	}

}

/*Function saveOPurl saves the Open Project instance URL into the config.json file using the information in the OP_url variable recieved*/
func saveOPurl(_ http.ResponseWriter, r *http.Request) {
	type save_json struct {
		OP_url string `json:"op_url"`
	}
	b_body, err := io.ReadAll(r.Body)
	utils.Check(err, "error", "Error 500. Internal server error. Open Project URL was not sent correctly and it could not be read")
	var body save_json
	json.Unmarshal(b_body, &body)

	var config *gabs.Container

	if _, err := os.Stat(utils.Config_path); err == nil {
		config, err = gabs.ParseJSONFile(utils.Config_path)
		utils.Check(err, "error", "Error 500. Config file could not be read")
	} else {
		config = gabs.New()
	}
	config.Set(body.OP_url, "openproject-url")

	f, err := os.Create(utils.Config_path)
	utils.Check(err, "Error", "Error creating config file on its destination path ('./.config')")
	defer f.Close()
	f.Write(config.BytesIndent("", "\t"))
}

/*Function PostOpenProject receives Open Project POSTs and calls requestOpenProject to control it*/
func PostOpenProject(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if utils.CheckExpirationDate() {
			var openproject oauths.Openproject = *oauths.NewOpenproject()
			openproject.RefreshAuth()
		}
		byte_body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Fatal("Error 500. Internal Server Error. On Open Project post receiving an unexpected error has occured.")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		go requestOpenProject(byte_body)
	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

/*Function requestOpenProject deals with Open Project POSTs depending on the type of POST*/
func requestOpenProject(data []byte) {
	action, _ := jsonparser.GetString(data, "action")
	log.Info(fmt.Sprintf("Open Project POST received. Action: '%s' ", action))

	wp_type, err2 := jsonparser.GetString(data, "work_package", "_embedded", "type", "name")
	utils.Check(err2, "error", "Type of task was not found on Open Project post")

	if strings.ToLower(wp_type) == "task" {
		repo, err := jsonparser.GetString(data, "work_package", utils.GetCustomFields().RepoField)
		utils.Check(err, "warning", "Github repository URL was not found. Check if the repository exists or if the master github user has access to this repository")
		switch {
		case strings.Contains(string(repo), "github"):
			utils.GithubOptions(data)
		default:
			log.Warn("Repository website not supported")
		}
	}
}

/*Function PostGithub receives GitHub POSTs and calls utils.OpenProjectOptions to control it*/
func PostGithub(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if utils.CheckExpirationDate() {
			var openproject oauths.Openproject = *oauths.NewOpenproject()
			openproject.RefreshAuth()
		}
		byte_body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		go utils.OpenProjectOptions(byte_body)
		event := r.Header.Get("X-GitHub-Event")
		repo_fullname, _ := jsonparser.GetString(byte_body, "repository", "full_name")
		user, _ := jsonparser.GetString(byte_body, "sender", "login")
		organization, _ := jsonparser.GetString(byte_body, "organization", "login")

		log.Info(fmt.Sprintf("Github POST received. Post event: '%s'; Organization: %s; Repository: '%s'; User: '%s' ", event, organization, repo_fullname, user))
	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

/*Function refreshProxy reads lastRefresh.txt before calling the utils.Refresh function and do the synchronization*/
func refreshProxy(w http.ResponseWriter, _ *http.Request) {
	var lastRefresh time.Time
	var config *gabs.Container

	if utils.CheckExpirationDate() {
		var openproject oauths.Openproject = *oauths.NewOpenproject()
		openproject.RefreshAuth()
	}

	if !utils.CheckConnectionGithub() || !utils.CheckConnectionOpenProject() {
		w.Write([]byte("Error 400. The connection with Open Project or Github is not done or has expired. Log in them before trying to refresh"))
		return
	}

	if _, err := os.Stat(utils.Config_path); err == nil {
		config, err = gabs.ParseJSONFile(utils.Config_path)
		utils.Check(err, "Error", "Error 500. Config file could not be read")
		read_ls := config.Search("lastSync").Data()
		if read_ls != nil {
			read_ls := read_ls.(string)
			if string(read_ls) != "" {
				var parseError error
				lastRefresh, parseError = time.Parse("2006-01-02T15:04:05Z", string(read_ls))
				utils.Check(parseError, "error", "The last synchronization date could not be parsed correctly. Check if it has the correct format, which is 'YYYY-MM-DDTHH:mm:ssZ'")
			} else {
				lastRefresh = time.Date(2000, 1, 1, 0, 0, 1, 0, time.UTC) // Set lastRefresh to 2000-01-01T00:00:01.0Z
				config.Set(lastRefresh.Format("2006-01-02T15:04:05Z"), "lastSync")
			}
		} else {
			lastRefresh = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC) // Set lastRefresh to 2000-01-01T00:00:00.0Z
			config := gabs.New()
			config.Set(lastRefresh.Format("2006-01-02T15:04:05Z"), "lastSync")
		}
	} else {
		lastRefresh = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC) // Set lastRefresh to 2000-01-01T00:00:00.0Z
		config := gabs.New()
		config.Set(lastRefresh.Format("2006-01-02T15:04:05Z"), "lastSync")
	}
	f, err := os.Create(utils.Config_path)
	utils.Check(err, "error", "Error 500. Config file could not be created on refresh")
	defer f.Close()
	f.Write(config.BytesIndent("", "\t"))

	c := make(chan string)
	go utils.Refresh(lastRefresh, c)
	result := <-c

	w.Write([]byte(result))
}

/*
Function resetRefreshDate deletes the latest refresh date stored in config.json
*/
func resetRefreshDate(_ http.ResponseWriter, _ *http.Request) {

	config, err := gabs.ParseJSONFile(utils.Config_path)
	utils.Check(err, "Error", "Error 500. Config file could not be read")

	config.Set("", "lastSync")

	f, err := os.Create(utils.Config_path)
	utils.Check(err, "Error", "Error creating config file on its destination path ('./.config')")
	defer f.Close()

	f.Write(config.BytesIndent("", "\t"))
}

/*
Function checkFields calls utils.CheckCustomFields() whenever is needed and checks the Open Project custom fields on demand.
*/
func checkFields(w http.ResponseWriter, _ *http.Request) {
	if utils.CheckExpirationDate() {
		var openproject oauths.Openproject = *oauths.NewOpenproject()
		openproject.RefreshAuth()
	}

	if !utils.CheckConnectionOpenProject() {
		w.Write([]byte("Error: Connection with Open Project missing. Log in the app to check the custom fields"))
		return
	}

	c := make(chan []string)
	go utils.CheckCustomFields(c)
	result := <-c
	msg := "All custom fields are correctly implemented"
	if len(result) > 0 {
		indent, _ := json.MarshalIndent(result, "", "\t")
		msg = fmt.Sprintf("Error: The following custom fields are missing --> %s", indent)
	}
	w.Write([]byte(msg))
}
