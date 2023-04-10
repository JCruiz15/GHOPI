package utils

import (
	"io"
	"os"

	"github.com/buger/jsonparser"
)

type permission string
type CustomFields struct {
	RepoField,
	SourceBranchField,
	TargetBranchField,
	GithubUserField string
}

const (
	WRITE       permission = "push"
	READ        permission = "pull"
	Config_path string     = ".config/config.json"
)

var OP_url string = "http://localhost:8080"

func GetCustomFields() CustomFields {
	f, errFile := os.Open(Config_path)
	Check(errFile, "error", "Config file could not be opened when looking for the custom fields")
	defer f.Close()
	config, _ := io.ReadAll(f)

	RepoField, _ := jsonparser.GetString(config, "customFields", "work_packages", "repoField")
	SourceBranchField, _ := jsonparser.GetString(config, "customFields", "work_packages", "sourceBranchField")
	TargetBranchField, _ := jsonparser.GetString(config, "customFields", "work_packages", "targetBranchField")
	GithubUserField, _ := jsonparser.GetString(config, "customFields", "users", "githubUserField")

	var fields = CustomFields{
		RepoField,
		SourceBranchField,
		TargetBranchField,
		GithubUserField,
	}

	return fields

}
