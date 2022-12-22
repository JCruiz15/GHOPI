package functions

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/buger/jsonparser"
	log "github.com/sirupsen/logrus"
)

var repoField string = "customField1"
var githubUserField string = "customField2"
var op_url string = "http://localhost:8080"

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

// ====== GITHUB ======

func Github_options(data []byte) {
	action, errAction := jsonparser.GetString(data, "action")
	Check(errAction, "warning")
	repo, _ := jsonparser.GetString(data, "work_package", repoField) //TODO
	if repo != "" {
		switch action {
		case "work_package:created":
			github_createBranch(data)
			go github_writePermission(data)

			id, _ := jsonparser.GetInt(data, "work_package", "id")
			targetBranch, _ := jsonparser.GetString(data, "work_package", "customField4")
			sourceBranch, _ := jsonparser.GetString(data, "work_package", "customField5")
			link := fmt.Sprintf("%s/compare/%s...%s?quick_pull=1&title=%s", repo, targetBranch, sourceBranch, fmt.Sprintf("%s - [%d]", sourceBranch, id))

			msg := "When the task is finish click in the following link to create a pull request for your task. [Create pull request in github](" + link + ")"
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

// ====== OPENPROJECT ======

func Openproject_options(data []byte) {
	all := make(map[string]interface{})
	json.Unmarshal(data, &all)
	// fmt.Println(all)

	if _, ok := all["pull_request"]; ok {
		pr_title, _ := jsonparser.GetString(data, "pull_request", "title")
		action, _ := jsonparser.GetString(data, "action")
		switch action {
		case "opened":
			openproject_change_status(data, 7)
			openproject_PR_msg(
				data,
				fmt.Sprintf("[%s] Pull request was opened", pr_title),
			)
		case "closed":
			openproject_change_status(data, 12)
			openproject_PR_msg(
				data,
				fmt.Sprintf("[%s] Pull request was closed. Task may be closed too", pr_title),
			)
		case "reopened":
			openproject_change_status(data, 13)
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
