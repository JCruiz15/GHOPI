package functions

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
)

// TODO - Change keyword:change

func Refresh(lastRefresh time.Time, channel chan string) {

	go CheckCustomFields()

	var repo_list []string
	op_url = Get_OP_uri()

	// ====== Obtain the list of work_packages since the last refresh ======

	f, err := os.Open(".config/config.json")
	Check(err, "error")
	config, _ := io.ReadAll(f)
	gh_token, err := jsonparser.GetString(config, "github-token")
	Check(err, "error")
	op_token, err := jsonparser.GetString(config, "openproject-token")
	Check(err, "error")

	page_size := 1000
	req_url := op_url + // host of openproject
		`/api/v3/work_packages?filters=%5B%7B%22updatedAt%22:%7B%22operator%22:%22%3C%3Ed%22,%22values%22:%5B%22` + // traduction = /api/v3/work_packages?filters=[{"updatedAt":{"operator":"<>d","values":["
		lastRefresh.Format("2006-01-02T15:04:05Z") + // Last date refreshed
		`%22,%20%22` + // traduction = ", "
		time.Now().Format("2006-01-02T15:04:05Z") + // Current date in your pc
		`%22%5D%7D%7D%5D&pageSize=` + // traduction = "]}}]&pageSize=
		strconv.Itoa(page_size) // the number of packages shown, by default is set to the max (1000)

	req, err := http.NewRequest("GET", req_url, strings.NewReader(""))
	Check(err, "fatal")

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", op_token))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode >= 400 && resp.StatusCode <= 499 {
		channel <- fmt.Sprintf("Error %d: Could not obtain Open Project work packages correctly. Try to log in again in Open Project or refresh again in a few minutes", resp.StatusCode)
		return
	}

	body, _ := io.ReadAll(resp.Body)

	var package_list []interface{}
	ids_body, _, _, _ := jsonparser.Get(body, "_embedded", "elements")
	json.Unmarshal(ids_body, &package_list)

	if len(package_list) == 0 {
		channel <- "There are no new changes to update"
		return
	}

	// ====== For each work_package give permissions and remove them as needed ======

	for i := 0; i < len(package_list); i++ {
		pack, _ := json.Marshal(package_list[i])
		repoURL, err := jsonparser.GetString(pack, RepoField)

		if err != nil && strings.Contains(err.Error(), "null") { // If repo field is empty exit the for loop
			subject, _ := jsonparser.GetString(pack, SourceBranchField)
			id, _ := jsonparser.GetInt(pack, "id")
			log.Warn(fmt.Sprintf(
				"Work package %s[id: %d] has no repo declared",
				subject,
				id,
			))
		} else {
			r := strings.Split(repoURL, "/")
			repoName := r[len(r)-1]
			org := r[len(r)-2]
			if !slices.Contains(repo_list, repoName) { // If repository not updated yet, all its collaborators must get read permissions by default
				// Get all collaborators of a repository
				collabs, err := getAllCollabs(repoURL, gh_token)
				Check(err, "error")
				for i := 0; i < len(collabs); i++ {
					collab, _ := json.Marshal(collabs[i])
					user, _ := jsonparser.GetString(collab, "login")
					body := map[string]string{
						"permission": "pull",
					}
					jsonBody, _ := json.Marshal(body)
					req_pull, _ := http.NewRequest(
						"PUT",
						fmt.Sprintf("https://api.github.com/repos/%s/%s/collaborators/%s", org, repoName, user),
						bytes.NewBuffer(jsonBody),
					)
					req_pull.Header.Set("Authorization", fmt.Sprintf("Bearer %s", gh_token))
					_, err := http.DefaultClient.Do(req_pull)
					Check(err, "fatal")
				}
				// Add repository to repo_list, to avoid repetitions
				repo_list = append(repo_list, repoName)

			}
			subj, _ := jsonparser.GetString(pack, SourceBranchField)
			target, _ := jsonparser.GetString(pack, TargetBranchField)

			// If task branch does not exist, create a new one
			if !branchExists(repoURL, subj, gh_token) {
				go createbranch(gh_token, repoName, org, subj, target)
			}

			// Get assignee and give write permision
			assignee_ref, err := jsonparser.GetString(pack, "_links", "assignee", "href")
			if err == nil {
				user, err := getGHuser_from_assigneehref(assignee_ref, op_token)
				Check(err, "error")
				givePermission(org, repoName, user, WRITE, gh_token)
				time.Sleep(500 * time.Millisecond)
			}
		}
	}

	date_path := ".config/lastRefresh.txt"
	new := time.Now()
	os.WriteFile(date_path, []byte(new.Format("2006-01-02T15:04:05Z")), fs.FileMode(os.O_TRUNC))
	msg := fmt.Sprintf("All changes since %s have been updated", lastRefresh.Format("Mon, 2 Jan 2006 [15:04]"))
	log.Info(msg)
	channel <- msg
}

func getAllCollabs(repository string, token string) ([]interface{}, error) {
	if !strings.Contains(repository, "github") {
		e := errors.New("repository manager not supported, only github may be used")
		return nil, e
	}
	r := strings.Split(repository, "/")
	repoName := r[len(r)-1]
	GH_ORG := r[len(r)-2]

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("https://api.github.com/repos/%s/%s/collaborators", GH_ORG, repoName),
		strings.NewReader(""),
	)
	Check(err, "error")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	Check(err, "error")

	var output []interface{}
	body, err := io.ReadAll(resp.Body)
	Check(err, "error")
	json.Unmarshal(body, &output)

	return output, nil

}

func branchExists(repository string, subject string, token string) bool {
	r := strings.Split(repository, "/")
	repoName := r[len(r)-1]
	GH_ORG := r[len(r)-2]
	req, _ := http.NewRequest(
		"GET",
		fmt.Sprintf("https://api.github.com/repos/%s/%s/branches/%s", GH_ORG, repoName, subject),
		strings.NewReader(""),
	)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	res, _ := http.DefaultClient.Do(req)
	if res.StatusCode == http.StatusNotFound {
		return false
	} else {
		return true
	}
}

func getGHuser_from_assigneehref(assigneehref string, token string) (string, error) {
	op_url = Get_OP_uri()
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s%s", op_url, assigneehref),
		strings.NewReader(""),
	)
	Check(err, "error")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := http.DefaultClient.Do(req)
	Check(err, "fatal")
	body, _ := io.ReadAll(resp.Body)
	return jsonparser.GetString(body, GithubUserField)
}
