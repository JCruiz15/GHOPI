package utils

import (
	"io"
	"os"

	"github.com/buger/jsonparser"
)

/*Variable to define a string type that contains the desired permissions for Github*/
type permission string

/*Variable to define a struct type that contains the names of the Open Project custom fields.*/
type CustomFields struct {
	RepoField,
	SourceBranchField,
	TargetBranchField,
	GithubUserField string
}

/*
Constants:

	WRITE: Of the 'permission' type that we previously created and contains the value "push".
	READ: Of the 'permission' type that we previously created and contains the value "pull".
	Config_path: Of the string type, contains the relative address of the configuration file.
*/
const (
	WRITE       permission = "push"
	READ        permission = "pull"
	Config_path string     = ".config/config.json"
)

/*Variable OP_url that contains the URL address of Open Project, by default it is "http://localhost:8080".*/
var OP_url string = "http://localhost:8080"

/*Function GetCustomFields() that obtains the values of the custom fields from the configuration file and returns them through the CustomFields struct.*/
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
