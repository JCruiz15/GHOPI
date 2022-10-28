package functions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/buger/jsonparser"
	log "github.com/sirupsen/logrus"
)

func github_createBranch(data []byte) int {
	// Get repository
	repo, err := jsonparser.GetString(data, "work_package", repoField)
	Check(err, "warning")
	r := strings.Split(repo, "/")
	repoName := r[len(r)-1]
	GH_ORG := r[len(r)-2]

	// Get name of task
	branchName, err2 := jsonparser.GetString(data, "work_package", "subject")
	Check(err2, "warning")

	admin, _ := jsonparser.GetString(data, "work_package", "_embedded", "author", githubUserField)
	token, err := get_token(admin)
	if Check(err, "error") {
		return http.StatusNotFound
	}

	// Get Last commit
	sha := get_lastcommit(data, token, repoName, GH_ORG)

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

	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

	resp, err := http.DefaultClient.Do(req)
	Check(err, "fatal")

	return resp.StatusCode
}

func get_lastcommit(data []byte, token string, repoName string, GH_ORG string) string {
	req, _ := http.NewRequest(
		"GET",
		fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/main", GH_ORG, repoName),
		strings.NewReader(""),
	)

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
	GH_ORG := r[len(r)-1]

	// Get name of task
	subject, err2 := jsonparser.GetString(data, "work_package", "subject")
	Check(err2, "warning")

	// Get id of task
	id, err3 := jsonparser.GetInt(data, "work_package", "id")
	Check(err3, "warning")

	// Get description of task
	desc, err4 := jsonparser.GetString(data, "work_package", "description", "raw")
	Check(err4, "warning")

	admin, _ := jsonparser.GetString(data, "work_package", "_embedded", "author", githubUserField)
	token, err := get_token(admin)
	if Check(err, "error") {
		return http.StatusNotFound
	}

	// Get branch
	branch := get_branch(data, token, repoName, subject, GH_ORG)

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

	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

	resp, err := http.DefaultClient.Do(req)
	Check(err, "fatal")

	return resp.StatusCode

}

func get_branch(data []byte, token string, repoName string, subject string, GH_ORG string) string {
	req, _ := http.NewRequest(
		"GET",
		fmt.Sprintf("https://api.github.com/repos/%s/%s/branches/%s", GH_ORG, repoName, subject),
		strings.NewReader(""),
	)

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

func github_readPermission(data []byte) int {
	assigneeRef, err := jsonparser.GetString(data, "work_package", "_links", "assignee", "href")
	if Check(err, "error") {
		task, _ := jsonparser.GetString(data, "work_package", "subject")
		id, _ := jsonparser.GetInt(data, "work_package", "id")
		log.Error(fmt.Sprintf("Task %s(%d) have no assignee. Attach a new collaborator to assign new permissions.This error provoked the last one", task, id))
		return http.StatusNotFound
	}
	repoURL, err := jsonparser.GetString(data, "work_package", repoField)
	Check(err, "warning")
	r := strings.Split(repoURL, "/")
	repo := r[len(r)-1]
	GH_ORG := r[len(r)-2]

	req, _ := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/%s", op_url, assigneeRef),
		strings.NewReader(""),
	)

	admin, _ := jsonparser.GetString(data, "work_package", "_embedded", "author", githubUserField)
	token, err := get_token(admin)
	if Check(err, "error") {
		return http.StatusNotFound
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

	resp, err := http.DefaultClient.Do(req)
	Check(err, "fatal")
	respbody, err := io.ReadAll(resp.Body)
	Check(err, "error")
	user, err := jsonparser.GetString(respbody, githubUserField)
	Check(err, "error")

	body := map[string]string{
		"permission": "pull",
	}
	jsonBody, _ := json.Marshal(body)

	req_perm, _ := http.NewRequest(
		"PUT",
		fmt.Sprintf("https://api.github.com/repos/%s/%s/collaborators/%s", GH_ORG, repo, user),
		bytes.NewBuffer(jsonBody),
	)

	req_perm.Header.Set("Authorization", fmt.Sprintf("token %s", token))

	resp_perm, err := http.DefaultClient.Do(req_perm)
	Check(err, "fatal")

	log.Info(fmt.Sprintf("%s have got write permissions in %s repository", user, repo))
	return resp_perm.StatusCode
}

func github_writePermission(data []byte) int {
	assigneeRef, err := jsonparser.GetString(data, "work_package", "_links", "assignee", "href")
	if Check(err, "error") {
		task, _ := jsonparser.GetString(data, "work_package", "subject")
		id, _ := jsonparser.GetInt(data, "work_package", "id")
		log.Error(fmt.Sprintf("Task %s(%d) have no assignee. Attach a new collaborator to assign new permissions.This error provoked the last one", task, id))
		return http.StatusNotFound
	}
	repoURL, err := jsonparser.GetString(data, "work_package", repoField)
	Check(err, "warning")
	r := strings.Split(repoURL, "/")
	repo := r[len(r)-1]
	GH_ORG := r[len(r)-2]

	req, _ := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/%s", op_url, assigneeRef),
		strings.NewReader(""),
	)

	admin, _ := jsonparser.GetString(data, "work_package", "_embedded", "author", githubUserField)
	token, err := get_token(admin)
	if Check(err, "error") {
		return http.StatusNotFound
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

	resp, err := http.DefaultClient.Do(req)
	Check(err, "fatal")
	respbody, err := io.ReadAll(resp.Body)
	Check(err, "error")
	user, err := jsonparser.GetString(respbody, githubUserField)
	Check(err, "error")

	body := map[string]string{
		"permission": "push",
	}
	jsonBody, _ := json.Marshal(body)

	req_perm, _ := http.NewRequest(
		"PUT",
		fmt.Sprintf("https://api.github.com/repos/%s/%s/collaborators/%s", GH_ORG, repo, user),
		bytes.NewBuffer(jsonBody),
	)

	req_perm.Header.Set("Authorization", fmt.Sprintf("token %s", token))

	resp_perm, err := http.DefaultClient.Do(req_perm)
	Check(err, "fatal")

	log.Info(fmt.Sprintf("%s have got write permissions in %s repository", user, repo))
	return resp_perm.StatusCode
}
