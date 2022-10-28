package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	"web-service-gin/applications"
	"web-service-gin/functions"

	"github.com/buger/jsonparser"
	"github.com/joho/godotenv"

	log "github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
	"gopkg.in/natefinch/lumberjack.v2"
)

// TODO - Rename a task when its branch is renamed and viceversa.

// TODO - Receive webhooks from gitlab.

// TODO - Make the documentation

// TODO - When refresh create a new branch for every new task

// TODO - Api security, prevent XSS, Session tokens etc.

// TODO - Create post buttons for user input

// TODO - Log archive to store changes and show it through webpage

// TODO - Create webhook and customFields through web pushing a button

// TODO - Instead of save Org name in variable, get it from repo url

var repoField string = "customField1"

func init() {
	log.SetFormatter(&easy.Formatter{
		TimestampFormat: "02/01/2006-15:04:05",
		LogFormat:       "[%lvl%] %time% %msg%\n",
	})

	log.SetOutput(io.MultiWriter(
		&lumberjack.Logger{
			Filename:   "outputs.log",
			MaxSize:    10,
			MaxBackups: 3,
			MaxAge:     28,
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
	http.HandleFunc("/instructions", instructions)
	http.HandleFunc("/api/openproject", PostOpenProject)
	http.HandleFunc("/api/refresh", refresh)

	http.HandleFunc("/github/login", github.LoginHandler)
	http.HandleFunc("/github/login/callback", github.CallbackHandler)
	http.HandleFunc("/github/loggedin",
		func(w http.ResponseWriter, r *http.Request) {
			github.LoggedinHandler(w, r, "", "")
		})

	http.HandleFunc("/op/login", openproject.LoginHandler)
	http.HandleFunc("/op/login/callback", openproject.CallbackHandler)
	http.HandleFunc("/op/loggedin",
		func(w http.ResponseWriter, r *http.Request) {
			openproject.LoggedinHandler(w, r, "", "")
		})

	port := 5002
	log.Info(fmt.Sprintf("Application running on port %d", port))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

func renderTemplate(w http.ResponseWriter, tmpl string) {
	t := template.Must(template.New(tmpl).ParseFiles("templates/base.html", "templates/"+tmpl+".html"))
	t.ExecuteTemplate(w, "base", tmpl)
}

func index(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "index")
}

func instructions(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "instructions")
}

func PostOpenProject(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		byte_body, err := io.ReadAll(r.Body)
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
	repo, err := jsonparser.GetString(data, "work_package", repoField)
	functions.Check(err, "warn")
	kind, err2 := jsonparser.GetString(data, "work_package", "_embedded", "type", "name")
	functions.Check(err2, "warn")

	if kind == "Task" {
		switch {
		case strings.Contains(string(repo), "github"):
			functions.Github_options(data)
		default:
			log.Warn("Repository URL not found or website not supported")
		}
	}
}

func refresh(w http.ResponseWriter, r *http.Request) {
	var t time.Duration
	t, _ = time.ParseDuration("3s")
	time.Sleep(t)

	fmt.Println(r.Method)
}
