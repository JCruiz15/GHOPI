package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/buger/jsonparser"
	log "github.com/sirupsen/logrus"
)

/*
githubCreateBranch uses the information from Open Project tasks to create a new branch on GitHub when needed.

It recieves data from Open Project, from which it takes the desired name of the branch, the target branch name and the GitHub repository assigned. Then it calls createBranch to do the call to GitHub API.
*/
func githubCreateBranch(data []byte) int {
	// Get repository
	id, err := jsonparser.GetString(data, "work_package", "id")
	Check(err, "warning", "Id not found on Open Project post. It may not be a work package post.")
	repo, err := jsonparser.GetString(data, "work_package", GetCustomFields().RepoField)
	Check(err, "warning", fmt.Sprintf("Repository was not found on Open Project post. Work package id: '%s'", id))
	r := strings.Split(repo, "/")
	repoName := r[len(r)-1]
	GH_ORG := r[len(r)-2]

	// Get name of task
	branchName, err2 := jsonparser.GetString(data, "work_package", GetCustomFields().SourceBranchField)
	Check(err2, "warning", fmt.Sprintf("Github new branch name was not found in Open Project post. Work package id: '%s'. Check if custom fields are correct", id))
	targetBranch, err2 := jsonparser.GetString(data, "work_package", GetCustomFields().TargetBranchField)
	Check(err2, "warning", fmt.Sprintf("Github target branch name was not found in Open Project post. Work package id: '%s'. Check if custom fields are correct", id))

	f, err := os.Open(".config/config.json")
	Check(err, "error", "Error 500. Config file could not be opened. Config file may not exists")
	defer f.Close()
	config, _ := io.ReadAll(f)
	token, err := jsonparser.GetString(config, "github-token")

	if Check(err, "error", "Github token was not found in config file") {
		return http.StatusNotFound
	}

	return createBranch(token, repoName, GH_ORG, branchName, targetBranch)
}

/*
createBranch uses the information gotten in githubCreateBranch to call the GitHub API and create a new branch with name 'source' and target branch 'target'.

It uses the function getLastcommit to obtain the sha string from the target branch, then it calls the GitHub API to create the desired branch. 
*/
func createBranch(token string, repoName string, orgName string, source string, target string) int {
	// Get Last commit

	sha := getLastcommit(token, repoName, orgName, target)
	source = strings.Replace(source, " ", "-", -1)

	// Post new branch
	body := map[string]string{
		"ref": fmt.Sprintf("refs/heads/%s", source),
		"sha": sha,
	}
	requestJSON, _ := json.Marshal(body)
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("https://api.github.com/repos/%s/%s/git/refs", orgName, repoName),
		bytes.NewBuffer(requestJSON),
	)
	Check(err, "error", "Error when creating the Github request for posting a new branch")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := http.DefaultClient.Do(req)
	Check(err, "fatal", fmt.Sprintf("Github API call to create a new branch '%s' failed (%s)", source, fmt.Sprintf("https://api.github.com/repos/%s/%s/git/refs", orgName, repoName)))

	return resp.StatusCode
}

/*
getLastCommit uses the GitHub API and returns a string with the sha code from the GitHub branch 'target'.
*/
func getLastcommit(token string, repoName string, GH_ORG string, target string) string {
	req, _ := http.NewRequest(
		"GET",
		fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s", GH_ORG, repoName, target),
		strings.NewReader(""),
	)

	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

	resp, err := http.DefaultClient.Do(req)
	Check(err, "fatal", fmt.Sprintf("Github did not respond to API call to obtain target branch '%s' sha (%s)", target, fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s", GH_ORG, repoName, target)))

	respbody, err := io.ReadAll(resp.Body)
	Check(err, "error", "Error when reading the response body from Github")
	sha, err_notFound := jsonparser.GetString(respbody, "sha")
	Check(err_notFound, "warn", "Sha not found in response body to obtain target branch")

	return sha
}

// func github_createPR(data []byte) int {
// 	// Get repository
// 	repo, err := jsonparser.GetString(data, "work_package", RepoField)
// 	Check(err, "warning")
// 	r := strings.Split(repo, "/")
// 	repoName := r[len(r)-1]
// 	GH_ORG := r[len(r)-1]

// 	// Get name of task
// 	subject, err2 := jsonparser.GetString(data, "work_package", "subject")
// 	Check(err2, "warning")

// 	// Get id of task
// 	id, err3 := jsonparser.GetInt(data, "work_package", "id")
// 	Check(err3, "warning")

// 	// Get description of task
// 	desc, err4 := jsonparser.GetString(data, "work_package", "description", "raw")
// 	Check(err4, "warning")

// 	f, err := os.Open(".config/config.json")
// 	Check(err, "error")
// 	config, _ := io.ReadAll(f)
// 	token, err := jsonparser.GetString(config, "github-token")
// 	if Check(err, "error") {
// 		return http.StatusNotFound
// 	}

// 	// Get branch
// 	branch := get_branch(data, token, repoName, subject, GH_ORG)

// 	// Body for request

// 	bodyMap := map[string]string{
// 		"title": fmt.Sprintf("%s[%d]", subject, id),
// 		"body":  desc,
// 		"head":  branch,
// 		"base":  "main",
// 	}
// 	requestJSON, _ := json.Marshal(bodyMap)

// 	req, err := http.NewRequest(
// 		"POST",
// 		fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls", GH_ORG, repoName),
// 		bytes.NewBuffer(requestJSON),
// 	)
// 	Check(err, "fatal")

// 	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

// 	resp, err := http.DefaultClient.Do(req)
// 	Check(err, "fatal")

// 	return resp.StatusCode

// }

// func get_branch(data []byte, token string, repoName string, subject string, GH_ORG string) string {
// 	req, _ := http.NewRequest(
// 		"GET",
// 		fmt.Sprintf("https://api.github.com/repos/%s/%s/branches/%s", GH_ORG, repoName, subject),
// 		strings.NewReader(""),
// 	)

// 	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

// 	resp, err := http.DefaultClient.Do(req)
// 	Check(err, "fatal")
// 	respbody, err := io.ReadAll(resp.Body)
// 	Check(err, "error")
// 	var branch string
// 	branch, err_notFound := jsonparser.GetString(respbody, "name")
// 	if err_notFound != nil {
// 		github_createBranch(data)
// 		branch = subject
// 	}

// 	return branch
// }

/*
githubReadPermission receives data from an Open Project POST and obtains the information needed from it to give READ permission to the user assigned to a task.

It uses the givePermission function to call the GitHub API.
*/
func githubReadPermission(data []byte) int {
	/*Obtain the assigne reference from Open Project POST*/
	assigneeRef, err := jsonparser.GetString(data, "work_package", "_links", "assignee", "href")
	if Check(err, "error", "Assignee on Open Project information was not found") {
		task, _ := jsonparser.GetString(data, "work_package", "subject")
		id, _ := jsonparser.GetInt(data, "work_package", "id")
		log.Error(fmt.Sprintf("Task %s(%d) have no assignee. Attach a new collaborator to assign new permissions.This error provoked the last one", task, id))
		return http.StatusNotFound
	}
	/*Obtain the work package ID from Open Project POST*/
	id, err := jsonparser.GetString(data, "work_package", "id")
	Check(err, "error", "ID was not found on work package")
	repoURL, err := jsonparser.GetString(data, "work_package", GetCustomFields().RepoField)
	Check(err, "warning", fmt.Sprintf("Repository was not found on work package with id '%s'", id))
	r := strings.Split(repoURL, "/")
	repo := r[len(r)-1]
	GH_ORG := r[len(r)-2]
	OP_url = GetOPuri()

	/*Obtain the github user from the assignee information in Open Project*/
	req, _ := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/%s", OP_url, assigneeRef),
		strings.NewReader(""),
	)

	f, err := os.Open(".config/config.json")
	Check(err, "error", "Error 500. Config file could not be opened. Config file may not exists")
	defer f.Close()
	config, _ := io.ReadAll(f)
	token, err := jsonparser.GetString(config, "github-token")
	if Check(err, "error", "Github token key was not found in config file. Check if you are correctly logged in") {
		return http.StatusNotFound
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

	resp, err := http.DefaultClient.Do(req)
	Check(err, "fatal", fmt.Sprintf("Open project API call to obtain assignee reference failed (%s)", fmt.Sprintf("%s/%s", OP_url, assigneeRef)))
	respbody, err := io.ReadAll(resp.Body)
	Check(err, "error", "Post body could not be read")
	user, err := jsonparser.GetString(respbody, GetCustomFields().GithubUserField)
	Check(err, "error", "Github user was not found in custom fields of Open Project post. Check if it is included in config and in your Open Project correctly.")

	return givePermission(GH_ORG, repo, user, READ, token)
}

/*
githubReadPermission receives data from an Open Project POST and obtains the information needed from it to give WRITE permission to the user assigned to a task.

It uses the givePermission function to call the GitHub API.
*/
func githubWritePermission(data []byte) int {
	/*Obtain the assigne reference from Open Project POST*/
	assigneeRef, err := jsonparser.GetString(data, "work_package", "_links", "assignee", "href")
	if Check(err, "error", "") {
		task, _ := jsonparser.GetString(data, "work_package", "subject")
		id, _ := jsonparser.GetInt(data, "work_package", "id")
		log.Error(fmt.Sprintf("Task %s(%d) have no assignee. Attach a new collaborator to assign new permissions.This error provoked the last one", task, id))
		return http.StatusNotFound
	}
	/*Obtain the work package ID from Open Project POST*/
	id, err := jsonparser.GetString(data, "work_package", "id")
	Check(err, "error", "ID was not found on work package")
	repoURL, err := jsonparser.GetString(data, "work_package", GetCustomFields().RepoField)
	Check(err, "warning", fmt.Sprintf("Repository was not found on work package with id '%s'", id))
	r := strings.Split(repoURL, "/")
	repo := r[len(r)-1]
	GH_ORG := r[len(r)-2]
	OP_url = GetOPuri()

	/*Obtain the github user from the assignee information in Open Project*/
	req, _ := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/%s", OP_url, assigneeRef),
		strings.NewReader(""),
	)

	f, err := os.Open(".config/config.json")
	Check(err, "error", "Error 500. Config file could not be opened. Config file may not exists")
	defer f.Close()
	config, _ := io.ReadAll(f)
	token, err := jsonparser.GetString(config, "github-token")
	if Check(err, "error", "Github token key was not found in config file. Check if you are correctly logged in") {
		return http.StatusNotFound
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

	resp, err := http.DefaultClient.Do(req)
	Check(err, "fatal", fmt.Sprintf("Open project API call to obtain assignee reference failed (%s)", fmt.Sprintf("%s/%s", OP_url, assigneeRef)))
	respbody, err := io.ReadAll(resp.Body)
	Check(err, "error", "Post body could not be read")
	user, err := jsonparser.GetString(respbody, GetCustomFields().GithubUserField)
	Check(err, "error", "Github user was not found in custom fields of Open Project post. Check if it is included in config and in your Open Project correctly.")

	return givePermission(GH_ORG, repo, user, WRITE, token)

}

/*
givePermission is the general function that uses the GitHub API to give permission to the 'user' given as a string. 

The type of permission is given by the 'permission' variable which uses the enum slope created in globalVariables.go

It returns the status code of the response and logs the output.
*/
func givePermission(organization string, repo string, user string, slope permission, token string) int {
	body := map[string]string{
		"permission": string(slope),
	}
	jsonBody, _ := json.Marshal(body)

	req_perm, _ := http.NewRequest(
		"PUT",
		fmt.Sprintf("https://api.github.com/repos/%s/%s/collaborators/%s", organization, repo, user),
		bytes.NewBuffer(jsonBody),
	)

	req_perm.Header.Set("Authorization", fmt.Sprintf("token %s", token))

	resp_perm, err := http.DefaultClient.Do(req_perm)
	Check(err, "fatal", fmt.Sprintf("Github API call to give permission to user '%s' in repository '%s' failed (%s)", user, repo, fmt.Sprintf("https://api.github.com/repos/%s/%s/collaborators/%s", organization, repo, user)))

	if resp_perm.StatusCode >= 200 && resp_perm.StatusCode <= 299 {
		log.Info(fmt.Sprintf("%s have got write permissions in %s repository", user, repo))
	} else {
		log.Error(fmt.Sprintf("Error %d: Could not give %s permissions", resp_perm.StatusCode, slope))
	}
	return resp_perm.StatusCode
}
