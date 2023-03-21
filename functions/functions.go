package functions

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

var op_url string = "http://localhost:8080"

var f, _ = os.Open(".config/config.json")
var config, _ = io.ReadAll(f)
var RepoField, _ = jsonparser.GetString(config, "customFields", "work_packages", "repoField")
var SourceBranchField, _ = jsonparser.GetString(config, "customFields", "work_packages", "sourceBranchField")
var TargetBranchField, _ = jsonparser.GetString(config, "customFields", "work_packages", "targetBranchField")
var GithubUserField, _ = jsonparser.GetString(config, "customFields", "users", "githubUserField")

func Check(err error, level string) bool {
	if err != nil {
		switch strings.ToLower(level) {
		case "info":
			log.Info(err.Error())
		case "warning":
			log.Warn(err.Error())
		case "error":
			log.Error(err.Error())
		case "fatal":
			log.Fatal(err.Error())
		}
		return true
	} else {
		return false
	}
}

func Get_OP_uri() string {
	var config *gabs.Container
	config_path := ".config/config.json"
	config, err := gabs.ParseJSONFile(config_path)
	Check(err, "error")
	val, ok := config.Path("openproject-url").Data().(string)
	if ok {
		return val
	} else {
		return "http://localhost:8080"
	}
}

// ====== From OPENPROJECT To GITHUB ======

func Github_options(data []byte) {
	action, errAction := jsonparser.GetString(data, "action")
	Check(errAction, "warning")
	repo, _ := jsonparser.GetString(data, "work_package", RepoField)
	if repo != "" {
		switch action {
		case "work_package:created":
			github_createBranch(data)
			go github_writePermission(data)

			id, _ := jsonparser.GetInt(data, "work_package", "id")
			targetBranch, _ := jsonparser.GetString(data, "work_package", TargetBranchField)
			sourceBranch, _ := jsonparser.GetString(data, "work_package", SourceBranchField)

			assigneeRef, _ := jsonparser.GetString(data, "work_package", "_links", "assignee", "href")

			op_url = Get_OP_uri()
			req, _ := http.NewRequest(
				"GET",
				fmt.Sprintf("%s/%s", op_url, assigneeRef),
				strings.NewReader(""),
			)

			f, err := os.Open(".config/config.json")
			Check(err, "error")
			config, _ := io.ReadAll(f)
			token, err := jsonparser.GetString(config, "github-token")
			Check(err, "error")
			req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

			resp, _ := http.DefaultClient.Do(req)
			respbody, err := io.ReadAll(resp.Body)
			Check(err, "error")
			assignee, err := jsonparser.GetString(respbody, GithubUserField)
			Check(err, "error")

			link := fmt.Sprintf("%s/compare/%s...%s?quick_pull=1&title=%s&assignees=%s", repo, targetBranch, sourceBranch, fmt.Sprintf("%s-[%d]", sourceBranch, id), assignee)

			msg := "When the task is finish click in the following link to create a pull request for your task. " + link + ""
			openproject_msg(msg, int(id))

		case "work_package:updated":
			status, errStatus := jsonparser.GetString(data, "work_package", "_embedded", "status", "name")
			Check(errStatus, "warning")
			switch status {
			case "In progress":
				go github_writePermission(data)
			case "Closed", "Rejected":
				go github_readPermission(data)
			default:
				go github_writePermission(data)
			}
		}
	} else {
		msg := "Task created and received successfully"
		id, _ := jsonparser.GetInt(data, "work_package", "id")
		openproject_msg(msg, int(id))
	}
}

// ====== From GITHUB To OPENPROJECT ======

func Openproject_options(data []byte) {
	all := make(map[string]interface{})
	json.Unmarshal(data, &all)

	if _, ok := all["pull_request"]; ok {
		pr_title, _ := jsonparser.GetString(data, "pull_request", "title")
		action, _ := jsonparser.GetString(data, "action")
		fmt.Println(action)
		switch action {
		case "opened":
			// openproject_change_status(data, 7)
			openproject_PR_msg(
				data,
				fmt.Sprintf("[%s] Pull request was opened", pr_title),
			)

		case "synchronize":
			openproject_change_status(data, 12)
			openproject_PR_msg(
				data,
				fmt.Sprintf("[%s] Pull request was merged. Task has been closed", pr_title),
			)
		case "closed":
			// openproject_change_status(data, 12)
			openproject_PR_msg(
				data,
				fmt.Sprintf("[%s] Pull request was closed. Task may be closed too", pr_title),
			)
		case "reopened":
			// openproject_change_status(data, 13)
			openproject_PR_msg(
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
