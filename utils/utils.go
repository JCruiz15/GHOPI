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

func APIkeyCheck(r *http.Request) bool {

	auth := r.Header.Get("Authentication")
	key, ok := strings.CutPrefix(auth, "Bearer ")
	if ok {
		return key == os.Getenv("API_KEY")
	}
	return false
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
			Check(err, "error", "Error 500. Config file could not be opened when GitHub post was received. Config file may not exist")
			defer f.Close()
			config, _ := io.ReadAll(f)
			token, err := jsonparser.GetString(config, "openproject-token")
			Check(err, "error", "Error when getting Open Project token, check if the field exists in .config file and if it is correctly cumplimented. Try to log in again on Open Project")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

			resp, _ := http.DefaultClient.Do(req)
			respbody, err := io.ReadAll(resp.Body)
			Check(err, "error", "Error when reading response on assignee reference")
			assignee, err := jsonparser.GetString(respbody, GetCustomFields().GithubUserField)
			Check(err, "error", "Error when getting github user from Open project. Check if the github user custom field is correctly created. Refresh when all changes are done to check custom fields.")

			link := fmt.Sprintf("%s/compare/%s...%s?quick_pull=1&title=%s&assignees=%s", repo, targetBranch, sourceBranch, fmt.Sprintf("[%d] Merge %s into %s ", id, sourceBranch, targetBranch), assignee)

			msg := "When the task is finish click in the following link to create a pull request for your task. " + link + ""
			openprojectMsg(msg, int(id))

		case "work_package:updated":
			status, errStatus := jsonparser.GetString(data, "work_package", "_embedded", "status", "name")
			Check(errStatus, "warning", "Work package status was not found in the body of an Open Project post received. It will give permission to the user by default.")
			switch status {
			case "In progress":
				go githubWritePermission(data)
			case "Closed", "Rejected":
				go githubRemoveUserFromOP(data)
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
		pr_url, _ := jsonparser.GetString(data, "pull_request", "url")
		action, _ := jsonparser.GetString(data, "action")
		switch action {
		case "opened":
			//openprojectChangeStatus(data, 15) // The status ID may be changed if needed
			openprojectPRmsg(
				data,
				fmt.Sprintf(`[%s] Pull request was opened ('%s')`, pr_title, pr_url),
			)
		case "closed":
			//openprojectChangeStatus(data, 13) // The status ID may be changed if needed
			merged, err := jsonparser.GetBoolean(data, "pull_request", "merged")
			Check(err, "error", "ERROR 404: Could not find if a pull request is merged. Assuming it was not.")
			if merged {
				openprojectPRmsg(
					data,
					fmt.Sprintf(`[%s] Pull request was merged. Task may be closed ('%s')`, pr_title, pr_url),
				)
			} else {
				openprojectPRmsg(
					data,
					fmt.Sprintf(`[%s] Pull request was closed. Task may be rejected ('%s')`, pr_title, pr_url),
				)
			}

		case "reopened":
			//openprojectChangeStatus(data, 15) // The status ID may be changed if needed
			openprojectPRmsg(
				data,
				fmt.Sprintf(`[%s] Pull request was reopened. Task may be reopened too ('%s')`, pr_title, pr_url),
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
	if Check(err, "error", "Error 500. Config file could not be opened checking GitHub connection. Config file may not exist") {
		return false
	}
	token := config.Search("github-token")
	user := config.Search("github-user")
	if token == nil || user == nil {
		log.Warn("Error when obtaining GitHub token and user. Log in GitHub to use the app")
		return false
	}

	if token.Data().(string) == "" || user.Data().(string) == "" {
		log.Warn("Error when obtaining GitHub token and user. Log in GitHub to use the app")
		config.Set("", "github-token")
		config.Set("", "github-user")

		f, err := os.Create(Config_path)
		Check(err, "Error", "Error 500. Config file could not be created. Config file may not exist")
		defer f.Close()
		f.Write(config.BytesIndent("", "\t"))

		return false
	}

	// Checking the ratelimit if it is smaller than 5000 the token does not have permissions and it is not valid.
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("https://api.github.com/users/%s", user.Data().(string)),
		nil,
	)
	if Check(err, "error", "") {
		return false
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", token.Data().(string)))

	resp, _ := http.DefaultClient.Do(req)
	ratelimit := resp.Header.Get("x-ratelimit-limit")

	if ratelimit == "5000" || resp.StatusCode != 200 {
		return true
	} else {
		log.Warn(fmt.Sprintf("GitHub did not return a valid response when checking token status. Response: %s; Status code: %d", resp.Status, resp.StatusCode))

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
	if Check(err, "error", "Error 500. Config file could not be opened checking Open Project connection. Config file may not exist") {
		return false
	}
	token := config.Search("openproject-token")
	OP_url := config.Search("openproject-url")
	if token == nil || OP_url == nil {
		log.Warn("Error when obtaining Open Project token and url. Log in Open Project to use the app")
		return false
	}

	if token.Data().(string) == "" || OP_url.Data().(string) == "" {
		log.Warn("Error when obtaining Open Project token and url. Log in Open Project to use the app")

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
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/api/v3/users/me", OP_url.Data().(string)),
		nil,
	)
	Check(err, "error", "")

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.Data().(string)))

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
		log.Warn(fmt.Sprintf("Open Project did not return a valid response when checking token status. Response: %s; Status code: %d", resp.Status, resp.StatusCode))

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
