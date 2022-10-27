package functions

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/buger/jsonparser"
	log "github.com/sirupsen/logrus"
)

var repoField string = "customField1"
var githubUserField string = "customField2"
var GH_ORG string = "Khaos-Slaves" // TODO - Change this variable with website

func Check(err error, level string) {
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
	}
}

func get_token(usr string) (string, error) {
	tokenF, err := os.Open(fmt.Sprintf(".tokens/github-%s", usr))
	if err != nil {
		return "", errors.New("token not found or expired: log in with our website ([url])")
	}
	defer tokenF.Close()
	token, _ := io.ReadAll(tokenF)
	return string(token), nil
}

// ====== GITHUB ======

func Github_options(data []byte) {
	action, errAction := jsonparser.GetString(data, "action")
	Check(errAction, "warning")

	switch action {
	case "work_package:created":
		github_createBranch(data)
		// github_writePermission(data)
	case "work_package:updated":
		status, errStatus := jsonparser.GetString(data, "work_package", "_embedded", "status", "name")
		Check(errStatus, "warning")
		fmt.Println(status)
		switch status {
		case "On hold":
			// github_writePermission(data)
			github_createPR(data)
		case "Closed", "Rejected":
			// github_readPermission(data)
		default:
			// github_writePermission(data)
		}
	}
}

func github_createBranch(data []byte) int {
	// Get repository
	repo, err := jsonparser.GetString(data, "work_package", repoField)
	Check(err, "warning")
	r := strings.Split(repo, "/")
	repoName := r[len(r)-1]

	// Get name of task
	branchName, err2 := jsonparser.GetString(data, "work_package", "subject")
	Check(err2, "warning")

	// Get Last commit
	sha := get_lastcommit(data, repoName)

	// Post new branch
	body := map[string]string{
		"ref": fmt.Sprintf("refs/heads/%s", branchName),
		"sha": sha,
	}
	requestJSON, _ := json.Marshal(body)

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("https://api.github.com/repos/%s/%s/git/refs", GH_ORG, repoName),
		bytes.NewBuffer(requestJSON),
	)
	Check(err, "error")

	admin, _ := jsonparser.GetString(data, "work_package", "_embedded", "author", githubUserField)
	token, err := get_token(admin)
	Check(err, "error")
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

	resp, err := http.DefaultClient.Do(req)
	Check(err, "fatal")

	return resp.StatusCode
}

func get_lastcommit(data []byte, repoName string) string {
	req, _ := http.NewRequest(
		"GET",
		fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/main", GH_ORG, repoName),
		strings.NewReader(""),
	)

	admin, _ := jsonparser.GetString(data, "work_package", "_embedded", "author", githubUserField)
	token, err := get_token(admin)
	Check(err, "error")
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Panic("Request failed")
	}

	respbody, err := io.ReadAll(resp.Body)
	Check(err, "error")
	sha, err_notFound := jsonparser.GetString(respbody, "sha")
	Check(err_notFound, "warn")

	return sha
}

func github_createPR(data []byte) int {
	// Get repository
	repo, err := jsonparser.GetString(data, "work_package", repoField)
	Check(err, "warning")
	r := strings.Split(repo, "/")
	repoName := r[len(r)-1]

	// Get name of task
	subject, err2 := jsonparser.GetString(data, "work_package", "subject")
	Check(err2, "warning")

	// Get id of task
	id, err3 := jsonparser.GetInt(data, "work_package", "id")
	Check(err3, "warning")

	// Get description of task
	desc, err4 := jsonparser.GetString(data, "work_package", "description", "raw")
	Check(err4, "warning")

	// Get branch
	branch := get_branch(data, repoName, subject)

	// Body for request

	bodyMap := map[string]string{
		"title": fmt.Sprintf("%s[%d]", subject, id),
		"body":  desc,
		"head":  branch,
		"base":  "main",
	}
	requestJSON, _ := json.Marshal(bodyMap)

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls", GH_ORG, repoName),
		bytes.NewBuffer(requestJSON),
	)
	if err != nil {
		log.Panic("Request creation failed")
	}

	admin, _ := jsonparser.GetString(data, "work_package", "_embedded", "author", githubUserField)
	token, err := get_token(admin)
	Check(err, "error")
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

	resp, err := http.DefaultClient.Do(req)
	Check(err, "fatal")

	return resp.StatusCode

}

func get_branch(data []byte, repoName string, subject string) string {
	req, _ := http.NewRequest(
		"GET",
		fmt.Sprintf("https://api.github.com/repos/%s/%s/branches/%s", GH_ORG, repoName, subject),
		strings.NewReader(""),
	)

	admin, _ := jsonparser.GetString(data, "work_package", "_embedded", "author", githubUserField)
	token, err := get_token(admin)
	Check(err, "error")
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

	resp, err := http.DefaultClient.Do(req)
	Check(err, "fatal")
	respbody, err := io.ReadAll(resp.Body)
	Check(err, "error")
	var branch string
	branch, err_notFound := jsonparser.GetString(respbody, "name")
	if err_notFound != nil {
		github_createBranch(data)
		branch = subject
	}

	return branch
}

func github_readPermission(data []byte) {
	panic("read permision unimplemented")
}

func github_writePermission(data []byte) {
	panic("write permision unimplemented")
}
