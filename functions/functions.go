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

	switch action {
	case "work_package:created":
		github_createBranch(data)
		github_writePermission(data)
	case "work_package:updated":
		status, errStatus := jsonparser.GetString(data, "work_package", "_embedded", "status", "name")
		Check(errStatus, "warning")
		switch status {
		case "On hold":
			github_writePermission(data)
			github_createPR(data)
		case "Closed", "Rejected":
			github_readPermission(data)
		default:
			github_writePermission(data)
		}
	}
}

// ====== OPENPROJECT ======

func Openproject_options(data []byte) {
	all := make(map[string]interface{})
	json.Unmarshal(data, &all)

	if _, ok := all["pull_request"]; ok {
		pr_title, _ := jsonparser.GetString(data, "pull_request", "title")
		action, _ := jsonparser.GetString(data, "action")
		switch action {
		case "opened":
			openproject_PR_msg(
				data,
				fmt.Sprintf("[%s] Pull request was opened", pr_title),
			)
		case "closed":
			openproject_PR_msg(
				data,
				fmt.Sprintf("[%s] Pull request was closed. Task may be closed too", pr_title),
			)
		case "reopened":
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
