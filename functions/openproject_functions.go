package functions

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/buger/jsonparser"
)

func openproject_PR_msg(data []byte, msg string) {
	title, errTitle := jsonparser.GetString(data, "pull_request", "title")
	Check(errTitle, "error")

	// Creating function searchID to find the integer between brackets
	searchID := func(s string) int {
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

	id := searchID(title)
	// Body for request

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
	fmt.Println(resp.Status)
}

// func openproject_task_msg(data []byte, msg string) {
// 	_, errTitle := jsonparser.GetString(data, "pull_request", "title")
// 	Check(errTitle, "error")
// 	repo, errRepo := jsonparser.GetString(data, "repository", "full_name")
// 	Check(errRepo, "error")

// 	tasks := get_tasks(repo, 200)

// 	fmt.Println(tasks)

// }

// func get_tasks(repo string, pagSize int) map[string]interface{} {
// 	req, err := http.NewRequest(
// 		"GET",
// 		fmt.Sprintf(`%s/api/v3/work_packages?pageSize=%d&filters=[{{"%s":{{"operator":"~", "values":"%s"}}}}]`, op_url, pagSize, "custom_filter1", repo),
// 		bytes.NewBuffer([]byte("")),
// 	)
// 	Check(err, "fatal")
// 	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", "iytm017DYUnuGlHqz92i8EU251QPVou1d03PE6c77uE"))

// 	resp, err := http.DefaultClient.Do(req)
// 	Check(err, "fatal")
// 	var out map[string]interface{}
// 	errMap := json.NewDecoder(resp.Body).Decode(&out)
// 	Check(errMap, "warn")

// 	return out
// }
