package functions

import (
	"bytes"
	"encoding/json"
	"errors"
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

func openproject_PR_msg(data []byte, msg string) {
	title, errTitle := jsonparser.GetString(data, "pull_request", "title")
	Check(errTitle, "error")

	id := searchID(title)

	openproject_msg(msg, id)
}

func openproject_msg(msg string, id int) {
	jsonStr := []byte(fmt.Sprintf(`{"comment":{"raw":"%s"}}`, msg))
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/api/v3/work_packages/%d/activities", op_url, id),
		bytes.NewBuffer(jsonStr),
	)
	Check(err, "fatal")

	f, err := os.Open(".config/config.json")
	Check(err, "error")
	config, _ := io.ReadAll(f)
	token, _ := jsonparser.GetString(config, "openproject-token")

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	Check(err, "fatal")
	log.Info(resp.Status)
}

func openproject_change_status(data []byte, status_id int) {

	title, errTitle := jsonparser.GetString(data, "pull_request", "title")
	Check(errTitle, "error")

	id := searchID(title)
	lockV := get_lockVersion(id)
	body := map[string]interface{}{
		"lockVersion": lockV,
		"_links": map[string]interface{}{
			"status": map[string]string{
				"href": fmt.Sprintf("/api/v3/statuses/%d", status_id),
			},
		},
	}
	bodyJson, _ := json.Marshal(body)

	req, err := http.NewRequest(
		"PATCH",
		fmt.Sprintf("%s/api/v3/work_packages/%d", op_url, id),
		bytes.NewBuffer(bodyJson),
	)
	Check(err, "fatal")

	f, err := os.Open(".config/config.json")
	Check(err, "error")
	config, _ := io.ReadAll(f)
	token, _ := jsonparser.GetString(config, "openproject-token")

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	Check(err, "fatal")
	log.Info(resp.Status)

}

// Creating function searchID to find the integer between brackets
func searchID(s string) int {
	i := strings.Index(s, "[")
	if i >= 0 {
		j := strings.Index(s, "]")
		if j >= 0 {
			x, err := strconv.Atoi(s[i+1 : j])
			Check(err, "warn")
			return x
		}
	}
	Check(errors.New("no index found"), "error")
	return -1
}

func get_lockVersion(wp_id int) int {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/api/v3/work_packages/%d", op_url, wp_id),
		bytes.NewBuffer([]byte("")),
	)
	Check(err, "fatal")

	f, err := os.Open(".config/config.json")
	Check(err, "error")
	config, _ := io.ReadAll(f)
	token, _ := jsonparser.GetString(config, "openproject-token")

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	Check(err, "fatal")
	body, _ := io.ReadAll(resp.Body)
	lockV, err := jsonparser.GetInt(body, "lockVersion")
	Check(err, "error")

	return int(lockV)
}

func CheckCustomFields() {

	// Se ejecuta al logearte en OP y al hacer refresh

	customFields_workpackages()
	customFields_user()

	var config *gabs.Container
	config_path := ".config/config.json"
	config, err := gabs.ParseJSONFile(config_path)
	Check(err, "Error")

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

func customFields_workpackages() {
	filter := url.QueryEscape(`[{"id":{"operator":"=","values":["1-1"]}}]`)
	url := op_url + `/api/v3/work_packages/schemas?filters=` + filter
	req, err := http.NewRequest(
		"GET",
		url,
		bytes.NewBuffer([]byte("")),
	)
	Check(err, "fatal")

	f, err := os.Open(".config/config.json")
	Check(err, "error")
	config, _ := io.ReadAll(f)
	token, _ := jsonparser.GetString(config, "openproject-token")

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	Check(err, "fatal")

	body, _ := io.ReadAll(resp.Body)

	elements, _, _, _ := jsonparser.Get(body, "_embedded", "elements")
	elements = elements[1:(len(elements) - 1)]

	searchKeys := make(map[string]interface{})
	json.Unmarshal(elements, &searchKeys)

	for key, value := range searchKeys {
		if strings.HasPrefix(key, "customField") {
			customfield := key
			v := reflect.ValueOf(value)
			for _, k := range v.MapKeys() {
				strct := v.MapIndex(k)
				if k.Interface() == "name" {
					subject := strings.ToLower(fmt.Sprintf("%v", strct.Interface()))
					if strings.Contains(subject, "repo") {
						write_config_customFields("repoField", customfield, "work_packages")
					} else if strings.Contains(subject, "source") {
						write_config_customFields("sourceBranchField", customfield, "work_packages")
					} else if strings.Contains(subject, "target") {
						write_config_customFields("targetBranchField", customfield, "work_packages")
					}
				}
			}
		}
	}
}

func customFields_user() {
	url := op_url + `/api/v3/users/schema`
	req, err := http.NewRequest(
		"GET",
		url,
		bytes.NewBuffer([]byte("")),
	)
	Check(err, "fatal")

	f, err := os.Open(".config/config.json")
	Check(err, "error")
	config, _ := io.ReadAll(f)
	token, _ := jsonparser.GetString(config, "openproject-token")

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	Check(err, "fatal")

	body, _ := io.ReadAll(resp.Body)

	searchKeys := make(map[string]interface{})
	json.Unmarshal(body, &searchKeys)

	for key, value := range searchKeys {
		if strings.HasPrefix(key, "customField") {
			customfield := key
			v := reflect.ValueOf(value)
			for _, k := range v.MapKeys() {
				strct := v.MapIndex(k)
				if k.Interface() == "name" {
					subject := strings.ToLower(fmt.Sprintf("%v", strct.Interface()))
					if strings.Contains(subject, "github") {
						write_config_customFields("githubUserField", customfield, "users")
					}
				}
			}
		}
	}
}

func write_config_customFields(key string, value string, path string) {
	var config *gabs.Container
	config_path := ".config/config.json"

	if _, err := os.Stat(config_path); err == nil {
		config, err = gabs.ParseJSONFile(config_path)
		Check(err, "Error")
	} else {
		config = gabs.New()
	}
	config.Set(value, "customFields", path, key)

	f, err := os.Create(config_path)
	Check(err, "Error")
	defer f.Close()
	f.Write(config.BytesIndent("", "\t"))
}
