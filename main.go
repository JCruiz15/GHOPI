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

// TODO - Make the documentation.

// TODO - Testing

// TODO - Not only check tokens in refresh, WHEN?

// TODO - Add license when commiting to main

// TODO - Redireccionar a index cuando la url esté mal escrita.

/*Function init does somthing*/
func init() {
	log.SetFormatter(&easy.Formatter{
		TimestampFormat: "02/01/2006-15:04:05",
		LogFormat:       "[%lvl%]\t%time%\t--\t%msg%\n",
	})

	log.SetOutput(io.MultiWriter(
		&lumberjack.Logger{
			Filename:   "outputs.log",
			MaxSize:    1,
			MaxBackups: 3,
			MaxAge:     15,
			Compress:   true,
		},
		os.Stdout,
	))

	if err := godotenv.Load(); err != nil {
		log.Fatal("No .env file found")
	}
}

func main() {
	var github oauths.Github = *oauths.NewGithub()
	var openproject oauths.Openproject = *oauths.NewOpenproject()

	fileServer := http.FileServer(http.Dir("./static"))

	http.Handle("/static/", http.StripPrefix("/static", fileServer))

	http.HandleFunc("/", index)
	http.HandleFunc("/docs", instructions)
	http.HandleFunc("/logs", logs)
	http.HandleFunc("/config-openproject", configOP)
	http.HandleFunc("/config-github", configGH)

	http.HandleFunc("/api/get-logs", getLogs)
	http.HandleFunc("/api/get-config", getConfig)
	http.HandleFunc("/api/openproject", PostOpenProject)
	http.HandleFunc("/api/github", PostGithub)
	http.HandleFunc("/api/refresh", refreshProxy)

	http.HandleFunc("/github/login", github.LoginHandler)
	http.HandleFunc("/github/login/callback", github.CallbackHandler)
	http.HandleFunc("/github/loggedin",
		func(w http.ResponseWriter, r *http.Request) {
			github.LoggedinHandler(w, r, nil, "")
		})
	http.HandleFunc("/github/webhook", githubWebhook)

	http.HandleFunc("/op/login", openproject.LoginHandler)
	http.HandleFunc("/op/login/callback", openproject.CallbackHandler)
	http.HandleFunc("/op/loggedin",
		func(w http.ResponseWriter, r *http.Request) {
			openproject.LoggedinHandler(w, r, nil, "")
		})
	http.HandleFunc("/op/save-url", saveOPurl)
	http.HandleFunc("/-/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	port := 5050
	log.Info(fmt.Sprintf("Application running on port %d", port))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

/*Function init does somthing*/
func renderTemplate(w http.ResponseWriter, tmpl string) {
	t := template.Must(template.New(tmpl).ParseFiles("templates/base.html", "templates/"+tmpl+".html"))
	t.ExecuteTemplate(w, "base", tmpl)
}

func index(w http.ResponseWriter, _ *http.Request) {
	renderTemplate(w, "index")
	// if !utils.CheckConnectionGithub() || !utils.CheckConnectionOpenProject() {
	// 	log.Warn("Github or Open Project token is not working or it has expired. Log in sign in to use the app.")
	// }
}

func instructions(w http.ResponseWriter, _ *http.Request) {
	renderTemplate(w, "instructions")
}

func configOP(w http.ResponseWriter, _ *http.Request) {
	renderTemplate(w, "openproject_config")
	// if !utils.CheckConnectionGithub() || !utils.CheckConnectionOpenProject() {
	// 	log.Warn("Github or Open Project token is not working or it has expired. Log in sign in to use the app.")
	// }
}

func configGH(w http.ResponseWriter, _ *http.Request) {
	renderTemplate(w, "github_config")
	// if !utils.CheckConnectionGithub() || !utils.CheckConnectionOpenProject() {
	// 	log.Warn("Github or Open Project token is not working or it has expired. Log in sign in to use the app.")
	// }
}

func logs(w http.ResponseWriter, _ *http.Request) {
	renderTemplate(w, "log")
	// if !utils.CheckConnectionGithub() || !utils.CheckConnectionOpenProject() {
	// 	log.Warn("Github or Open Project token is not working or it has expired. Log in sign in to use the app.")
	// }
}

func getLogs(w http.ResponseWriter, _ *http.Request) {
	file, err := os.Open("outputs.log")
	utils.Check(err, "error", "Error 500. Logs file could not be open")
	defer file.Close() // TODO - errcheck

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
		w.Write(lines[i]) // TODO - errcheck
	}
}

func getConfig(w http.ResponseWriter, _ *http.Request) {
	var config *gabs.Container

	if _, err := os.Stat(utils.Config_path); err == nil {
		config, err = gabs.ParseJSONFile(utils.Config_path)
		utils.Check(err, "error", "Error 500. Config file could not be read")
	} else {
		config = gabs.New()
	}
	config.Delete("github-token")      // TODO - errcheck
	config.Delete("openproject-token") // TODO - errcheck

	w.Header().Set("Content-Type", "application/json")
	w.Write(config.EncodeJSON()) // TODO - errcheck
}

func githubWebhook(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var orgName string = ""

	if r.Method == "POST" {
		jsonOrgName, _ := io.ReadAll(r.Body)
		var errorJSON error
		orgName, errorJSON = jsonparser.GetString(jsonOrgName, "organizationName")
		utils.Check(errorJSON, "error", "Error 500. The server did not received organization name correctly")
	}

	if orgName != "" {
		URL := fmt.Sprintf("https://%s%s", r.Host, "/api/github")

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
		utils.Check(err, "error", "Error 500. Config file could not be opened. Config file may not exists")
		config, _ := io.ReadAll(f)
		token, _ := jsonparser.GetString(config, "github-token")
		req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

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
		w.Write(r) // TODO - errcheck

	} else {
		output := map[string]interface{}{
			"message": "Webhook creation failed: Wrong http method or Organization name not received",
			"status":  http.StatusInternalServerError,
		}
		resul, _ := json.Marshal(output)
		w.Write(resul) // TODO - errcheck
	}

}

func saveOPurl(_ http.ResponseWriter, r *http.Request) {
	type save_json struct {
		OP_url string `json:"op_url"`
	}
	b_body, err := io.ReadAll(r.Body)
	utils.Check(err, "error", "Error 500. Internal server error. Open Project URL was not sent correctly and it could not be read")
	var body save_json
	json.Unmarshal(b_body, &body) // TODO - errcheck

	var config *gabs.Container

	if _, err := os.Stat(utils.Config_path); err == nil {
		config, err = gabs.ParseJSONFile(utils.Config_path)
		utils.Check(err, "error", "Error 500. Config file could not be read")
	} else {
		config = gabs.New()
	}
	config.Set(body.OP_url, "openproject-url") // TODO - errcheck

	f, err := os.Create(utils.Config_path)
	utils.Check(err, "Error", "Error creating config file on its destination path ('./.config')")
	defer f.Close()                       // TODO - errcheck
	f.Write(config.BytesIndent("", "\t")) // TODO - errcheck
}

func PostOpenProject(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
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

func requestOpenProject(data []byte) {
	action, _ := jsonparser.GetString(data, "action")
	log.Info(fmt.Sprintf("Open Project POST received. Action: '%s' ", action))
	repo, err := jsonparser.GetString(data, "work_package", utils.GetCustomFields().RepoField)
	if err != nil {
		log.Warn("Github repository URL was not found. Check if the repository exists or if the master github user has access to this repository")
		return
	}
	wp_type, err2 := jsonparser.GetString(data, "work_package", "_embedded", "type", "name")
	utils.Check(err2, "error", "Type of task was not found on Open Project post")

	if wp_type == "Task" {
		switch {
		case strings.Contains(string(repo), "github"):
			utils.GithubOptions(data)
		default:
			log.Warn("Repository website not supported")
		}
	}
}

func PostGithub(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		byte_body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		go utils.OpenProjectOptions(byte_body)
		t, _ := jsonparser.GetString(byte_body, "ref_type")
		fullname, _ := jsonparser.GetString(byte_body, "repository", "fullname")
		user, _ := jsonparser.GetString(byte_body, "sender", "login")

		log.Info(fmt.Sprintf("Github POST received. Post type: '%s'; Repository: '%s'; User: '%s' ", t, fullname, user))
	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

func refreshProxy(w http.ResponseWriter, _ *http.Request) { // TODO - Read lastRefresh in config not in lastRefresh.txt
	var lastRefresh time.Time
	var config *gabs.Container

	if !utils.CheckConnectionGithub() || !utils.CheckConnectionOpenProject() {
		w.Write([]byte("Error 400. The connection with Open Project or Github is not done or has expired. Log in them before trying to refresh"))
		return
	}

	if _, err := os.Stat(utils.Config_path); err == nil {
		config, err = gabs.ParseJSONFile(utils.Config_path)
		utils.Check(err, "Error", "Error 500. Config file could not be read")
		read_ls := config.Search("lastSync").Data().(string)

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
		config.Set(lastRefresh.Format("2006-01-02T15:04:05Z"), "lastSync") // TODO - errcheck
	}
	f, err := os.Create(utils.Config_path)
	utils.Check(err, "fatal", "Error 500. Config file could not be created on refresh")
	defer f.Close()                       // TODO - errcheck
	f.Write(config.BytesIndent("", "\t")) // TODO - errcheck

	c := make(chan string)
	go utils.Refresh(lastRefresh, c)
	result := <-c

	w.Write([]byte(result)) // TODO - errcheck
}
