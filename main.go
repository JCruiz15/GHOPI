package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"text/template"
	"time"

	"web-service-gin/applications"

	"github.com/joho/godotenv"
)

// TODO - Make the decisions via Objects oriented. Request abstract class with children.

// TODO - Rename a task when its branch is renamed and viceversa.

// TODO - Receive webhooks from gitlab.

// TODO - Make the documentation

// TODO - Create new url for each new page opened

// TODO - When refresh create a new branch for every new task

// TODO - Api security, prevent XSS, Session tokens etc.

// TODO - Create post buttons for user inputs

func init() {
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
	http.HandleFunc("/api/openproject", postOpenProject)
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
	fmt.Printf("Application started on port %d\n", port)
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

func postOpenProject(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		b := json.NewDecoder(r.Body)
		body := map[string]interface{}{}
		b.Decode(&body)
		go requestOpenProject(body)
		fmt.Fprint(w, "POST received")
	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

func requestOpenProject(body map[string]interface{}) {
	fmt.Print("DONE")
}

func refresh(w http.ResponseWriter, r *http.Request) {
	var t time.Duration
	t, _ = time.ParseDuration("3s")
	time.Sleep(t)

	fmt.Println(r.Method)
}
