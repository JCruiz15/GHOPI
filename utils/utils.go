/*
Utils contains all the functions which uses the logic of the API.

· globalVariables.go: Defines constants and structs which are used along all the project.

· utils.go: Defines functions which are general and common to all the views and functions of the API.

· utilsGithub.go: Defines all the functions necessary to handle the information given by Github.

· utilsOpenProject.go: Defines all the functions necessary to handle the information given by Open Project.

· utils.Refresh.go: Defines the logic behind the Refresh function of the API.
*/
package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/Jeffail/gabs/v2"
	"github.com/buger/jsonparser"
	log "github.com/sirupsen/logrus"
)

/*
Check returns true when `err != nil` and false if not.

It recieves a `level` argument which can be 'info', 'warning', 'error' or 'fatal'; and depending on which one it receives it will log a message of such type. If the word inserted is misspelled it logs nothing and returns true.

It also uses `msg` to log the error with the message. If `msg==""` it logs the error message by default
*/
func Check(err error, level string, msg string) bool {
	if err != nil {
		if msg == "" {
			msg = err.Error()
		}
		switch strings.ToLower(level) {
		case "info":
			log.Info(msg)
		case "warning":
			log.Warn(msg)
		case "error":
			log.Error(msg)
		case "fatal":
			log.Fatal(msg)
		default:
		}
		return true
	} else {
		return false
	}
}

/*
Checks the config file looking for the Open Project url.
If the config file does not exist it returns an error.
If it does not find the url inside the file, it returns 'http://localhost:8080' by default.

Then, if it finds it, it returns the url as a string.
*/
func GetOPuri() string {
	var config *gabs.Container
	config_path := ".config/config.json"
	config, err := gabs.ParseJSONFile(config_path)
	Check(err, "error", "Error when parsing config file. Check if it exists and if it is alright")
	val, ok := config.Path("openproject-url").Data().(string)
	if ok {
		return val
	} else {
		return "http://localhost:8080"
	}
}

// ====== From OPENPROJECT To GITHUB ======

/*
GithubOptions receives the json body of a POST from Open Project as a bytes array.
It parses the json and calls to the Github function needed.

If the case is a work package creation, it will call githubCreateBranch(data) and githubWritePermission(data), then it sends a message into the Open Project task with a link to create the Pull Request.

If the case is a work package update, it check the status of the task and change the Github permissions depending on it.

If the task does not have repository it will only send a message into the Open Project task telling that the task was received.
*/
func GithubOptions(data []byte) {
	action, errAction := jsonparser.GetString(data, "action")
	Check(errAction, "warning", "'Action' field was not found in Github post JSON")
	repo, _ := jsonparser.GetString(data, "work_package", GetCustomFields().RepoField)
	if repo != "" {
		switch action {
		case "work_package:created":
			githubCreateBranch(data)
			go githubWritePermission(data)

			id, _ := jsonparser.GetInt(data, "work_package", "id")
			targetBranch, _ := jsonparser.GetString(data, "work_package", GetCustomFields().TargetBranchField)
			sourceBranch, _ := jsonparser.GetString(data, "work_package", GetCustomFields().SourceBranchField)

			assigneeRef, _ := jsonparser.GetString(data, "work_package", "_links", "assignee", "href")

			OP_url = GetOPuri()
			req, _ := http.NewRequest(
				"GET",
				fmt.Sprintf("%s/%s", OP_url, assigneeRef),
				strings.NewReader(""),
			)

			f, err := os.Open(Config_path)
			Check(err, "error", "Error 500. Config file could not be opened. Config file may not exists")
			defer f.Close()
			config, _ := io.ReadAll(f)
			token, err := jsonparser.GetString(config, "github-token")
			Check(err, "error", "Error when getting github token check if the field exists in .config file and if it is correctly cumplimented. Try to log in again on Github")
			req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

			resp, _ := http.DefaultClient.Do(req)
			respbody, err := io.ReadAll(resp.Body)
			Check(err, "error", "Error when reading response on assignee reference")
			assignee, err := jsonparser.GetString(respbody, GetCustomFields().GithubUserField)
			Check(err, "error", "Error when getting github user from Open project. Check if the github user custom field is correctly created. Refresh when all changes are done to check custom fields.")

			link := fmt.Sprintf("%s/compare/%s...%s?quick_pull=1&title=%s&assignees=%s", repo, targetBranch, sourceBranch, fmt.Sprintf("%s-[%d]", sourceBranch, id), assignee)

			msg := "When the task is finish click in the following link to create a pull request for your task. " + link + ""
			openprojectMsg(msg, int(id))

		case "work_package:updated":
			status, errStatus := jsonparser.GetString(data, "work_package", "_embedded", "status", "name")
			Check(errStatus, "warning", "Work package status was not found in the body of an Open Project post received. It will give permission to the user by default.")
			switch status {
			case "In progress":
				go githubWritePermission(data)
			case "Closed", "Rejected":
				go githubReadPermission(data)
			default:
				go githubWritePermission(data)
			}
		}
	} else {
		msg := "Task created and received successfully"
		id, _ := jsonparser.GetInt(data, "work_package", "id")
		openprojectMsg(msg, int(id))
	}
}

// ====== From GITHUB To OPENPROJECT ======

/*
OpenProjectOptions receives the json body of a POST from Github as a bytes array.
It parses the json and calls to the Open Project function needed.

If it receives a pull request POST it will check the action of the pull request
and will send a different message into the Open Project task reporting the state of the Pull Request.
It will also change the task status if the Pull Request was merged into closed status (id: 12).

If it receives a deleting branch POST, it will send a message into Open Project reporting the branch removal.
*/
func OpenProjectOptions(data []byte) {
	all := make(map[string]interface{})
	json.Unmarshal(data, &all)

	if _, ok := all["pull_request"]; ok {
		pr_title, _ := jsonparser.GetString(data, "pull_request", "title")
		action, _ := jsonparser.GetString(data, "action")
		switch action {
		case "opened":
			// openproject_change_status(data, 7)
			openprojectPRmsg(
				data,
				fmt.Sprintf("[%s] Pull request was opened", pr_title),
			)

		case "synchronize":
			openprojectChangeStatus(data, 12)
			openprojectPRmsg(
				data,
				fmt.Sprintf("[%s] Pull request was merged. Task has been closed", pr_title),
			)
		case "closed":
			// openproject_change_status(data, 12)
			openprojectPRmsg(
				data,
				fmt.Sprintf("[%s] Pull request was closed. Task may be closed too", pr_title),
			)
		case "reopened":
			// openproject_change_status(data, 13)
			openprojectPRmsg(
				data,
				fmt.Sprintf("[%s] Pull request was reopened. Task may be reopened too", pr_title),
			)
		}
	} else if _, ok := all["deleted"]; ok {
		deleted, _ := jsonparser.GetBoolean(data, "deleted")
		if deleted {
			b_title, _ := jsonparser.GetString(data, "ref")
			b := strings.Split(b_title, "/")
			branch := b[len(b)-1]

			openprojectPRmsg(
				data,
				fmt.Sprintf("Branch %s was deleted. This task may be rejected", branch),
			)
		}
	}
}

// ====== PINGS ======

/*
Checks the status of the Github token. It will return true if it
is valid and false if it is not. If the token is not valid it will delete Github user and
token from the config file, so the user must log in again.
*/
func CheckConnectionGithub() bool {

	config, err := gabs.ParseJSONFile(Config_path)
	if Check(err, "error", "Error 500. Config file could not be opened. Config file may not exists") {
		return false
	}
	token := config.Search("github-token").Data().(string)
	user := config.Search("github-user").Data().(string)

	if token == "" || user == "" {
		log.Warn("Error when obtaining github token and user. Log in in github to use the app")
		config.Set("", "github-token")
		config.Set("", "github-user")

		f, err := os.Create(Config_path)
		Check(err, "Error", "Error 500. Config file could not be created. Config file may not exists")
		defer f.Close()
		f.Write(config.BytesIndent("", "\t"))

		return false
	}

	// Checking the ratelimit if it is smaller than 5000 the token does not have permissions and it is not valid.
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("https://api.github.com/users/%s", user),
		nil,
	)
	Check(err, "error", "")

	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

	resp, _ := http.DefaultClient.Do(req)
	ratelimit := resp.Header.Get("x-ratelimit-limit")

	if ratelimit == "5000" || resp.StatusCode != 200 {
		return true
	} else {
		config.Set("", "github-token")
		config.Set("", "github-user")

		f, err := os.Create(Config_path)
		Check(err, "Error", "Error 500. Config file could not be created. Config file may not exists")
		defer f.Close()
		f.Write(config.BytesIndent("", "\t"))
		return false
	}

}

/*
Checks the status of the Open Project token. It will return true if it
is valid and false if it is not. If the token is not valid it will delete Open Project user, token
and url from the config file, so the user must log in again.
*/
func CheckConnectionOpenProject() bool {
	config, err := gabs.ParseJSONFile(Config_path)
	if Check(err, "error", "Error 500. Config file could not be opened. Config file may not exists") {
		return false
	}
	token := config.Search("openproject-token").Data().(string)
	OP_url = config.Search("openproject-url").Data().(string)

	if token == "" || OP_url == "" {
		log.Warn("Error when obtaining Open Project token and url. Log in in Open Project to use the app")

		config.Set("", "openproject-token")
		config.Set("", "openproject-user")
		config.Set("", "openproject-url")

		f, err := os.Create(Config_path)
		Check(err, "Error", "Error 500. Config file could not be created. Config file may not exists")
		defer f.Close()
		f.Write(config.BytesIndent("", "\t"))

		return false
	}

	// If the name returned is not anonymus the token is valid, other way is not and will return false.
	req, _ := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/api/v3/users/me", OP_url),
		nil,
	)
	Check(err, "error", "")

	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Warn("Error 404. Open Project url is not well written or it does not exist. Check the Open Project url status")

		config.Set("", "openproject-token")
		config.Set("", "openproject-user")
		config.Set("", "openproject-url")

		f, err := os.Create(Config_path)
		Check(err, "Error", "Error 500. Config file could not be created. Config file may not exists")
		defer f.Close()
		f.Write(config.BytesIndent("", "\t"))

		return false
	}
	if resp.StatusCode != 200 {
		log.Warn("Open Project did not return a valid response when checking token status")

		config.Set("", "openproject-token")
		config.Set("", "openproject-user")
		config.Set("", "openproject-url")

		f, err := os.Create(Config_path)
		Check(err, "Error", "Error 500. Config file could not be created. Config file may not exists")
		defer f.Close()
		f.Write(config.BytesIndent("", "\t"))

		return false
	}

	body, err := io.ReadAll(resp.Body)
	Check(err, "error", "Error when reading body to check Open Project token status")

	me, err := jsonparser.GetString(body, "name")
	if Check(err, "error", "") {
		config.Set("", "openproject-token")
		config.Set("", "openproject-user")
		config.Set("", "openproject-url")

		f, err := os.Create(Config_path)
		Check(err, "Error", "Error 500. Config file could not be created. Config file may not exists")
		defer f.Close()
		f.Write(config.BytesIndent("", "\t"))

		return false
	}
	if me == "Anonymus" {
		config.Set("", "openproject-token")
		config.Set("", "openproject-user")
		config.Set("", "openproject-url")

		f, err := os.Create(Config_path)
		Check(err, "Error", "Error 500. Config file could not be created. Config file may not exists")
		defer f.Close()
		f.Write(config.BytesIndent("", "\t"))

		return false
	}
	return true
}
