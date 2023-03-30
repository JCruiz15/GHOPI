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
		}
		return true
	} else {
		return false
	}
}

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

			f, err := os.Open(".config/config.json")
			Check(err, "error", "Error 500. Config file could not be opened. Config file may not exists")
			defer f.Close() // TODO - errcheck
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

func OpenProjectOptions(data []byte) {
	all := make(map[string]interface{})
	json.Unmarshal(data, &all) // TODO - errcheck

	if _, ok := all["pull_request"]; ok {
		pr_title, _ := jsonparser.GetString(data, "pull_request", "title")
		action, _ := jsonparser.GetString(data, "action")
		fmt.Println(action)
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

			// openproject_task_msg(
			// 	data,
			// 	fmt.Sprintf("[%s] Branch was deleted. This task may be rejected", branch),
			// )

			log.Info(fmt.Sprintf("Branch [%s] has been deleted", branch))
		}
	}
}
