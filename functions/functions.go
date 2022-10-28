package functions

import (
	"errors"
	"fmt"
	"io"
	"os"
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
		github_writePermission(data)
	case "work_package:updated":
		status, errStatus := jsonparser.GetString(data, "work_package", "_embedded", "status", "name")
		Check(errStatus, "warning")
		fmt.Println(status)
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
