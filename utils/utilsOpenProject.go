package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/Jeffail/gabs/v2"
	"github.com/buger/jsonparser"
	log "github.com/sirupsen/logrus"
)

func openprojectPRmsg(data []byte, msg string) {
	title, errTitle := jsonparser.GetString(data, "pull_request", "title")
	Check(errTitle, "error", "Pull request title was not found on Github data")
	id := searchID(title)

	openprojectMsg(msg, id)
}

func openprojectMsg(msg string, id int) {
	jsonStr := []byte(fmt.Sprintf(`{"comment":{"raw":"%s"}}`, msg))
	OP_url = GetOPuri()
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/api/v3/work_packages/%d/activities", OP_url, id),
		bytes.NewBuffer(jsonStr),
	)
	Check(err, "error", "Open Project API request creation to send message failed")

	f, err := os.Open(".config/config.json")
	Check(err, "error", "Error 500. Config file could not be opened. Config file may not exists")
	defer f.Close()
	config, _ := io.ReadAll(f)
	token, _ := jsonparser.GetString(config, "openproject-token")

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	Check(err, "fatal", fmt.Sprintf("Open Project API call to send message failed (%s)", fmt.Sprintf("%s/api/v3/work_packages/%d/activities", OP_url, id)))
	if resp.StatusCode != 200 {
		log.Error("Pull request message could not be sent correctly. Check if the custom fields are correctly inserted.")
	} else {
		log.Info(fmt.Sprintf("Pull request message sent to package %d", id))
	}
}

func openprojectChangeStatus(data []byte, status_id int) {

	title, errTitle := jsonparser.GetString(data, "pull_request", "title")
	Check(errTitle, "error", "Pull request title was not found on Github data")

	id := searchID(title)
	lockV := getLockVersion(id)
	body := map[string]interface{}{
		"lockVersion": lockV,
		"_links": map[string]interface{}{
			"status": map[string]string{
				"href": fmt.Sprintf("/api/v3/statuses/%d", status_id),
			},
		},
	}
	bodyJson, _ := json.Marshal(body)
	OP_url = GetOPuri()

	req, err := http.NewRequest(
		"PATCH",
		fmt.Sprintf("%s/api/v3/work_packages/%d", OP_url, id),
		bytes.NewBuffer(bodyJson),
	)
	Check(err, "error", "Open Project API request creation to change status failed")

	f, err := os.Open(".config/config.json")
	Check(err, "error", "Error 500. Config file could not be opened. Config file may not exists")
	defer f.Close()
	config, _ := io.ReadAll(f)
	token, _ := jsonparser.GetString(config, "openproject-token")

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	Check(err, "fatal", fmt.Sprintf("Open Project API call to change work package '%d' status failed (%s)", id, fmt.Sprintf("%s/api/v3/work_packages/%d", OP_url, id)))
	log.Info(resp.Status)

}

// Creating function searchID to find the integer between brackets
func searchID(s string) int {
	i := strings.Index(s, "[")
	if i >= 0 {
		j := strings.Index(s, "]")
		if j >= 0 {
			x, err := strconv.Atoi(s[i+1 : j])
			Check(err, "warn", fmt.Sprintf("The index found for work package '%s' could not be converted into int", s))
			return x
		}
	}
	Check(fmt.Errorf("no index found for work package with title '%s'", s), "error", "")
	return -1
}

func getLockVersion(wp_id int) int {
	OP_url = GetOPuri()
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/api/v3/work_packages/%d", OP_url, wp_id),
		bytes.NewBuffer([]byte("")),
	)
	Check(err, "error", fmt.Sprintf("Open Project API request creation to get lock version of work package '%d' failed", wp_id))

	f, err := os.Open(".config/config.json")
	Check(err, "error", "Error 500. Config file could not be opened. Config file may not exists")
	config, _ := io.ReadAll(f)
	token, _ := jsonparser.GetString(config, "openproject-token")

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	Check(err, "fatal", fmt.Sprintf("Open Project API call to get lock version of work package '%d' failed (%s)", wp_id, fmt.Sprintf("%s/api/v3/work_packages/%d", OP_url, wp_id)))
	body, _ := io.ReadAll(resp.Body)
	lockV, err := jsonparser.GetInt(body, "lockVersion")
	Check(err, "error", "lockVersion was not found in response body from Open Project API call to get lock version")

	return int(lockV)
}

func CheckCustomFields() {

	// Se ejecuta al logearte en OP y al hacer refresh

	customFieldsWorkpackages()
	customFieldsUser()

	var config *gabs.Container
	config_path := ".config/config.json"
	config, err := gabs.ParseJSONFile(config_path)
	Check(err, "Error", "Error 500. Config file could not be read")

	if !config.Exists("customFields", "users", "githubUserField") {
		log.Error("Github user custom field is not created or could not be found. Its name must contain 'github' to be found correctly.")
	} else if !config.Exists("customFields", "work_packages", "repoField") {
		log.Error("Repository custom field is not created or could not be found. Its name must contain 'repo' to be found correctly.")
	} else if !config.Exists("customFields", "work_packages", "sourceBranchField") {
		log.Error("Source branch custom field is not created or could not be found. Its name must contain 'source' to be found correctly.")
	} else if !config.Exists("customFields", "work_packages", "targetBranchField") {
		log.Error("Target branch custom field is not created or could not be found. Its name must contain 'target' to be found correctly.")
	}
}

func customFieldsWorkpackages() {
	OP_url = GetOPuri()
	filter := url.QueryEscape(`[{"id":{"operator":"=","values":["1-1"]}}]`)
	url := OP_url + `/api/v3/work_packages/schemas?filters=` + filter
	req, err := http.NewRequest(
		"GET",
		url,
		bytes.NewBuffer([]byte("")),
	)
	Check(err, "error", "Open Project API request creation to get work packages custom fields failed")

	f, err := os.Open(".config/config.json")
	Check(err, "error", "Error 500. Config file could not be opened. Config file may not exists")
	defer f.Close()
	config, _ := io.ReadAll(f)
	token, _ := jsonparser.GetString(config, "openproject-token")

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	Check(err, "fatal", fmt.Sprintf("Open Project API call to get work packages custom fields failed (%s)", url))

	body, _ := io.ReadAll(resp.Body)

	elements, _, _, err := jsonparser.Get(body, "_embedded", "elements")
	if err != nil {
		Check(err, "error", "Unauthenticated. Log again in Open Project to synchronize correctly")
		return
	}
	elements = elements[1:(len(elements) - 1)]

	searchKeys := make(map[string]interface{})
	json.Unmarshal(elements, &searchKeys) // TODO - errcheck

	for key, value := range searchKeys {
		if strings.HasPrefix(key, "customField") {
			customfield := key
			v := reflect.ValueOf(value)
			for _, k := range v.MapKeys() {
				strct := v.MapIndex(k)
				if k.Interface() == "name" {
					subject := strings.ToLower(fmt.Sprintf("%v", strct.Interface()))
					if strings.Contains(subject, "repo") {
						writeConfigCustomFields("repoField", customfield, "work_packages")
					} else if strings.Contains(subject, "source") {
						writeConfigCustomFields("sourceBranchField", customfield, "work_packages")
					} else if strings.Contains(subject, "target") {
						writeConfigCustomFields("targetBranchField", customfield, "work_packages")
					}
				}
			}
		}
	}
}

func customFieldsUser() {
	OP_url = GetOPuri()
	url := OP_url + `/api/v3/users/schema`
	req, err := http.NewRequest(
		"GET",
		url,
		bytes.NewBuffer([]byte("")),
	)
	Check(err, "error", "Open Project API request creation to get custom user fields failed")

	f, err := os.Open(".config/config.json")
	Check(err, "error", "Error 500. Config file could not be opened. Config file may not exists")
	config, _ := io.ReadAll(f)
	token, _ := jsonparser.GetString(config, "openproject-token")

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	Check(err, "fatal", fmt.Sprintf("Open Project API call to get custom user fields failed (%s)", url))

	body, _ := io.ReadAll(resp.Body)

	searchKeys := make(map[string]interface{})
	json.Unmarshal(body, &searchKeys) // TODO - errcheck

	for key, value := range searchKeys {
		if strings.HasPrefix(key, "customField") {
			customfield := key
			v := reflect.ValueOf(value)
			for _, k := range v.MapKeys() {
				strct := v.MapIndex(k)
				if k.Interface() == "name" {
					subject := strings.ToLower(fmt.Sprintf("%v", strct.Interface()))
					if strings.Contains(subject, "github") {
						writeConfigCustomFields("githubUserField", customfield, "users")
					}
				}
			}
		}
	}
}

func writeConfigCustomFields(key string, value string, path string) {
	var config *gabs.Container
	config_path := ".config/config.json"

	if _, err := os.Stat(config_path); err == nil {
		config, err = gabs.ParseJSONFile(config_path)
		Check(err, "Error", "Error 500. Config file could not be read")
	} else {
		config = gabs.New()
	}
	config.Set(value, "customFields", path, key) // TODO - errcheck

	f, err := os.Create(config_path)
	Check(err, "Error", "Config file could not be created. Check permissions of editing of the app")
	defer f.Close()                       // TODO - errcheck
	f.Write(config.BytesIndent("", "\t")) // TODO - errcheck
}
