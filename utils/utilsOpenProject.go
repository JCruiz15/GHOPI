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
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/buger/jsonparser"
	log "github.com/sirupsen/logrus"
)

/*
Function openprojectPRmsg uses data from GitHub POSTs to obtain the ID given to search for the Open Project work package related to the Pull request.

It uses openprojectMsg to write the message given in 'msg' variable into the Open Project work package.

It uses searchID function to obtain the ID of the work package from the Pull Request title.
*/
func openprojectPRmsg(data []byte, msg string) {
	title, errTitle := jsonparser.GetString(data, "pull_request", "title")
	Check(errTitle, "error", "Pull request title was not found on Github data")
	id := searchID(title)

	openprojectMsg(msg, id)
}

/*
Function openprojectMsg uses the Open Project API to write a message given into the work package with id: 'id'.
*/
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
	Check(err, "error", "Error 500. Config file could not be opened writing a message in Open Project. Config file may not exist")
	defer f.Close()
	config, _ := io.ReadAll(f)
	token, _ := jsonparser.GetString(config, "openproject-token")

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	Check(err, "error", fmt.Sprintf("Open Project API call to send message failed (%s)", fmt.Sprintf("%s/api/v3/work_packages/%d/activities", OP_url, id)))
	if resp.StatusCode != 201 {
		log.Error("Pull request message could not be sent correctly. Check if the custom fields are correctly inserted.")
	} else {
		log.Info(fmt.Sprintf("Pull request message sent to package %d", id))
	}
}

/*
Function openprojectChangeStatus uses the Open Project API to change the status of a work package depending on the 'status_id' given by the utils.OpenProjectOptions function.

It uses searchID function to obtain the ID of the work package from the Pull Request title.

It uses the getLockVersion to obtain the latest version of the work package.
*/
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
	Check(err, "error", "Error 500. Config file could not be opened when changing a Task status. Config file may not exist")
	defer f.Close()
	config, _ := io.ReadAll(f)
	token, _ := jsonparser.GetString(config, "openproject-token")

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	_, err = http.DefaultClient.Do(req)
	Check(err, "error", fmt.Sprintf("Open Project API call to change work package '%d' status failed (%s)", id, fmt.Sprintf("%s/api/v3/work_packages/%d", OP_url, id)))
}

/*
Function searchID looks for a number into brackets from a string given.
*/
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
	Check(fmt.Errorf("no index found in '%s' it must be between brackets ('[]')", s), "error", "")
	return -1
}

/*
Function getLockVersion uses the Open Project API to obtain the latest version of the work package given by 'wp_id'.
*/
func getLockVersion(wp_id int) int {
	OP_url = GetOPuri()
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/api/v3/work_packages/%d", OP_url, wp_id),
		bytes.NewBuffer([]byte("")),
	)
	Check(err, "error", fmt.Sprintf("Open Project API request creation to get lock version of work package '%d' failed", wp_id))

	f, err := os.Open(".config/config.json")
	Check(err, "error", "Error 500. Config file could not be opened getting a Task Lock version. Config file may not exist")
	config, _ := io.ReadAll(f)
	token, _ := jsonparser.GetString(config, "openproject-token")

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	Check(err, "error", fmt.Sprintf("Open Project API call to get lock version of work package '%d' failed (%s)", wp_id, fmt.Sprintf("%s/api/v3/work_packages/%d", OP_url, wp_id)))
	body, _ := io.ReadAll(resp.Body)
	lockV, err := jsonparser.GetInt(body, "lockVersion")
	Check(err, "error", "The lockVersion was not found in response body from Open Project API call to get lock version")

	return int(lockV)
}

/*
Function CheckCustomFields has the purpose of checking if the custom fields needed for the correct functioning of the app.

It uses customFieldsWorkpackages and customFieldsUser functions to check the custom fields from work packages and users, respectively.
*/
func CheckCustomFields(channel ...chan []string) {

	// It is executed when you log into Open Project and do a syunchronization

	customFieldsWorkpackages()
	customFieldsUser()

	var config *gabs.Container
	config_path := ".config/config.json"
	config, err := gabs.ParseJSONFile(config_path)
	Check(err, "Error", "Error 500. Config file could not be read")

	if len(channel) < 1 {
		if !config.Exists("customFields", "users", "githubUserField") {
			log.Error("Github user custom field is not created or could not be found. Its name must contain 'github' to be found correctly.")
		} else if !config.Exists("customFields", "work_packages", "repoField") {
			log.Error("Repository custom field is not created or could not be found. Its name must contain 'repo' to be found correctly.")
		} else if !config.Exists("customFields", "work_packages", "sourceBranchField") {
			log.Error("Source branch custom field is not created or could not be found. Its name must contain 'source' to be found correctly.")
		} else if !config.Exists("customFields", "work_packages", "targetBranchField") {
			log.Error("Target branch custom field is not created or could not be found. Its name must contain 'target' to be found correctly.")
		}
	} else if len(channel) == 1 {
		missing_fields := []string{}
		if !config.Exists("customFields", "users", "githubUserField") {
			log.Error("Github user custom field is not created or could not be found. Its name must contain 'github' to be found correctly.")
			missing_fields = append(missing_fields, "githubUserField")
		} else if !config.Exists("customFields", "work_packages", "repoField") {
			log.Error("Repository custom field is not created or could not be found. Its name must contain 'repo' to be found correctly.")
			missing_fields = append(missing_fields, "repoField")
		} else if !config.Exists("customFields", "work_packages", "sourceBranchField") {
			log.Error("Source branch custom field is not created or could not be found. Its name must contain 'source' to be found correctly.")
			missing_fields = append(missing_fields, "sourceBranchField")
		} else if !config.Exists("customFields", "work_packages", "targetBranchField") {
			log.Error("Target branch custom field is not created or could not be found. Its name must contain 'target' to be found correctly.")
			missing_fields = append(missing_fields, "targetBranchField")
		}
		channel[0] <- missing_fields
	}
}

/*
Function customFieldsWorkpackages checks the existence and stores the value of 'repoField', 'sourceBranchField' and 'targetBranchField' custom fields.
*/
func customFieldsWorkpackages() {
	// Obtain Open PRoject token
	f, err := os.Open(".config/config.json")
	Check(err, "error", "Error 500. Config file could not be opened checking work packages custom fields. Config file may not exist")
	defer f.Close()
	config, _ := io.ReadAll(f)
	token, err := jsonparser.GetString(config, "openproject-token")
	Check(err, "error", "Open Project token not found when checking Work packages custom fields. Log in Open Project")

	OP_url = GetOPuri()
	lastProjectID := GetLastProjectID(OP_url, token)
	filter := url.QueryEscape(fmt.Sprintf(`[{"id":{"operator":"=","values":["%d-1"]}}]`, lastProjectID))
	url := OP_url + `/api/v3/work_packages/schemas?filters=` + filter
	req, err := http.NewRequest(
		"GET",
		url,
		bytes.NewBuffer([]byte("")),
	)
	Check(err, "error", "Open Project API request creation to get work packages custom fields failed")

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	Check(err, "error", fmt.Sprintf("Open Project API call to get work packages custom fields failed (%s)", url))

	body, _ := io.ReadAll(resp.Body)

	elements, _, _, err := jsonparser.Get(body, "_embedded", "elements")
	if err != nil {
		Check(err, "error", "Unauthenticated. Log again in Open Project to synchronize correctly")
		return
	}
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

/*
Function customFieldsUser checks the existence and stores the value of 'githubUserField' custom field.
*/
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
	Check(err, "error", "Error 500. Config file could not be opened checking users custom fields. Config file may not exist")
	config, _ := io.ReadAll(f)
	token, _ := jsonparser.GetString(config, "openproject-token")

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	Check(err, "error", fmt.Sprintf("Open Project API call to get custom user fields failed (%s)", url))

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
						writeConfigCustomFields("githubUserField", customfield, "users")
					}
				}
			}
		}
	}
}

/*
Function writeConfigCustomFields is used to store into '.config/config.json' the values of custom fields.
*/
func writeConfigCustomFields(key string, value string, path string) {
	var config *gabs.Container
	config_path := ".config/config.json"

	if _, err := os.Stat(config_path); err == nil {
		config, err = gabs.ParseJSONFile(config_path)
		Check(err, "Error", "Error 500. Config file could not be read")
	} else {
		config = gabs.New()
	}
	config.Set(value, "customFields", path, key)

	f, err := os.Create(config_path)
	Check(err, "Error", "Config file could not be created. Check permissions of editing of the app")
	defer f.Close()
	f.Write(config.BytesIndent("", "\t"))
}

/*
Function GetLastProjectID is used to obtain the latest project of an Open Project instance.
This is done to be able to check correctly the connection and custom fields of Open Project.

The function returns an integer, indicating the latest project ID.
*/
func GetLastProjectID(op_url string, token string) int {
	url := fmt.Sprintf("%s/api/v3/projects?select=[total,elements/name,elements/id]", op_url)
	req, err := http.NewRequest(
		"GET",
		url,
		bytes.NewBuffer([]byte("")),
	)
	if err != nil {
		Check(err, "error", "Error when creating last project URL")
		return 1
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		Check(err, "error", fmt.Sprintf("Open Project API call to get custom user fields failed (%s)", url))
		return 1
	}

	body, _ := io.ReadAll(resp.Body)

	elements, _, _, err := jsonparser.Get(body, "_embedded", "elements")
	if err != nil {
		Check(err, "error", "Unauthenticated. Log again in Open Project to synchronize correctly")
		return 1
	}
	// elements = elements[1:(len(elements) - 1)]

	var searchKeys []map[string]interface{}
	json.Unmarshal(elements, &searchKeys)
	if len(searchKeys) < 1 {
		log.Error("Error: No projects found in Open Project")
		return 1
	}

	return int(searchKeys[0]["id"].(float64))
}

func CheckExpirationDate() bool {
	var config *gabs.Container
	if _, err := os.Stat(Config_path); err == nil {
		config, err = gabs.ParseJSONFile(Config_path)
		Check(err, "Error", "Error 500. Config file could not be read")
	} else {
		config = gabs.New()
	}
	expiration, err := jsonparser.GetString(config.Bytes(), "openproject-expiration")
	if Check(err, "warning", "ERROR 404: Expiration date not found in config file.") {
		expiration = "2006-01-02T15:04:05Z"
	}
	now := time.Now()
	expDate, err := time.Parse("2006-01-02T15:04:05Z", expiration)
	if Check(err, "error", "ERROR 501: Error parsing the expiration date") {
		return false
	}
	return now.After(expDate)
}
