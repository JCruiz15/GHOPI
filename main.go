package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	"web-service-gin/applications"
	"web-service-gin/functions"

	"github.com/buger/jsonparser"
	"github.com/joho/godotenv"

	"github.com/Jeffail/gabs/v2"
	log "github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
	"gopkg.in/natefinch/lumberjack.v2"
)

// TODO - CUIDADO con las tareas SIN REPOSITORIO

// TODO - Rename a task when its branch is renamed and viceversa.

// TODO - Make the documentation

// TODO - Api security, prevent XSS, etc.

// TODO - Crear el webhook button para openproject

// TODO - Hacer una función para checkear el estado de los tokens

// TODO - Check if tasks closed need a branch too or not

func init() {
	log.SetFormatter(&easy.Formatter{
		TimestampFormat: "02/01/2006-15:04:05",
		LogFormat:       "[%lvl%] %time% %msg%\n",
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
	var github applications.Github = *applications.NewGithub()
	var openproject applications.Openproject = *applications.NewOpenproject()

	fileServer := http.FileServer(http.Dir("./static"))

	http.Handle("/static/", http.StripPrefix("/static", fileServer))

	http.HandleFunc("/", index)
	http.HandleFunc("/usermanual", instructions)
	http.HandleFunc("/logs", logs)

	http.HandleFunc("/config-openproject", config_op)
	http.HandleFunc("/config-github", config_gh)

	http.HandleFunc("/api/openproject", PostOpenProject)
	http.HandleFunc("/api/github", PostGithub)
	http.HandleFunc("/api/refresh", refresh_proxy)

	http.HandleFunc("/github/login", github.LoginHandler)
	http.HandleFunc("/github/login/callback", github.CallbackHandler)
	http.HandleFunc("/github/loggedin",
		func(w http.ResponseWriter, r *http.Request) {
			github.LoggedinHandler(w, r, nil, "")
		})
	http.HandleFunc("/github/webhook", github_webhook)

	http.HandleFunc("/op/login", openproject.LoginHandler)
	http.HandleFunc("/op/login/callback", openproject.CallbackHandler)
	http.HandleFunc("/op/loggedin",
		func(w http.ResponseWriter, r *http.Request) {
			openproject.LoggedinHandler(w, r, nil, "")
		})
	http.HandleFunc("/op/save-url", save_openproject_url)

	port := 5002
	log.Info(fmt.Sprintf("Application running on port %d", port))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))

}

func renderTemplate(w http.ResponseWriter, tmpl string) {
	t := template.Must(template.New(tmpl).ParseFiles("templates/base.html", "templates/"+tmpl+".html"))
	t.ExecuteTemplate(w, "base", tmpl)
}

func index(w http.ResponseWriter, _ *http.Request) {
	renderTemplate(w, "index")
	// http.Redirect(w, r, "/instructions", http.StatusSeeOther)
}

func instructions(w http.ResponseWriter, _ *http.Request) {
	renderTemplate(w, "instructions")
}

func config_op(w http.ResponseWriter, _ *http.Request) {
	renderTemplate(w, "openproject_config")
}

func config_gh(w http.ResponseWriter, _ *http.Request) {
	renderTemplate(w, "github_config")
}

func logs(w http.ResponseWriter, _ *http.Request) {
	renderTemplate(w, "log")
}

func github_webhook(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var orgName string = ""

	if r.Method == "POST" {
		jsonOrgName, _ := io.ReadAll(r.Body)
		var errorJSON error
		orgName, errorJSON = jsonparser.GetString(jsonOrgName, "organizationName")
		functions.Check(errorJSON, "error")
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
			log.Panic("API Request creation failed")
		}

		req.Header.Set("Accept", "application/vnd.github+json")

		f, err := os.Open(".config/config.json")
		functions.Check(err, "error")
		config, _ := io.ReadAll(f)
		token, _ := jsonparser.GetString(config, "github-token")
		req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Panic("Request failed")
		}
		r, _ := io.ReadAll(resp.Body)

		name, getErr := jsonparser.GetString(r, "name")
		functions.Check(getErr, "error")

		if name == "web" {
			id, _ := jsonparser.GetInt(r, "id")
			log.Info(fmt.Sprintf("Github webhook created with id %d", id))
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

func save_openproject_url(w http.ResponseWriter, r *http.Request) {
	type save_json struct {
		OP_url string `json:"op_url"`
	}
	b_body, err := io.ReadAll(r.Body)
	functions.Check(err, "error")
	var body save_json
	json.Unmarshal(b_body, &body)

	var config *gabs.Container
	config_path := ".config/config.json"

	if _, err := os.Stat(config_path); err == nil {
		config, err = gabs.ParseJSONFile(config_path)
		functions.Check(err, "Error")
	} else {
		config = gabs.New()
	}
	config.Set(body.OP_url, "openproject-url")

	f, err := os.Create(config_path)
	functions.Check(err, "Error")
	defer f.Close()
	f.Write(config.BytesIndent("", "\t"))
}

func PostOpenProject(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		byte_body, err := io.ReadAll(r.Body)
		// fmt.Println(string(byte_body))

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		go requestOpenProject(byte_body)
		log.Info("Open Projet POST received")
	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

func requestOpenProject(data []byte) {
	repo, err := jsonparser.GetString(data, "work_package", functions.RepoField)
	if err != nil {
		log.Warn("Repository URL not found")
		return
	}
	wp_type, err2 := jsonparser.GetString(data, "work_package", "_embedded", "type", "name")
	functions.Check(err2, "warn")

	if wp_type == "Task" {
		switch {
		case strings.Contains(string(repo), "github"):
			functions.Github_options(data)
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
		go functions.Openproject_options(byte_body)
		log.Info("Github POST received")
	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

func refresh_proxy(w http.ResponseWriter, _ *http.Request) {
	var lastRefresh time.Time
	lastRefresh_path := ".config/lastRefresh.txt"

	if _, err := os.Stat(lastRefresh_path); err == nil {
		lr, err := os.Open(lastRefresh_path)
		functions.Check(err, "Error")
		read_lr, _ := io.ReadAll(lr)
		if string(read_lr) != "" {
			var parseError error
			lastRefresh, parseError = time.Parse("2006-01-02T15:04:05Z", string(read_lr))
			functions.Check(parseError, "error")
		} else {
			lastRefresh = time.Date(2000, 1, 1, 0, 0, 1, 0, time.UTC) // Set lastRefresh to 2000-01-01T00:00:01.0Z
			os.WriteFile(lastRefresh_path, []byte(lastRefresh.Format("2006-01-02T15:04:05Z")), fs.FileMode(os.O_TRUNC))
		}
	} else {
		lastRefresh = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC) // Set lastRefresh to 2000-01-01T00:00:00.0Z
		_, err := os.Create(lastRefresh_path)
		functions.Check(err, "error")
		os.WriteFile(lastRefresh_path, []byte(lastRefresh.Format("2006-01-02T15:04:05Z")), fs.FileMode(os.O_TRUNC))
	}
	c := make(chan string)
	go functions.Refresh(lastRefresh, c)
	result := <-c

	w.Write([]byte(result))
}
