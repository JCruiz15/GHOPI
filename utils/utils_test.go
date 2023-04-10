package utils_test

import (
	"GHOPI/utils"
	"errors"
	"testing"
)

func TestCheck(t *testing.T) {
	errInfo := errors.New("error Info")
	errWarning := errors.New("error Warning")
	errError := errors.New("error Error")
	// errFatal := errors.New("error Fatal")

	resInfo := utils.Check(errInfo, "info", "")
	if resInfo != true {
		t.Error("Check info not worked correctly")
	}
	resWarning := utils.Check(errWarning, "warning", "")
	if resWarning != true {
		t.Error("Check warning not worked correctly")
	}
	resError := utils.Check(errError, "error", "")
	if resError != true {
		t.Error("Check error not worked correctly")
	}
	// resFatal := utils.Check(errFatal, "fatal", "")
	// if resFatal != true {
	// 	t.Error("Check fatal not worked correctly")
	// }
}

func TestGetOPuri(t *testing.T) {
	output := utils.GetOPuri()
	if !(output != "") {
		t.Error("GetOPuri returned an empty string")
	}
}

func TestCheckConnectionGithub(t *testing.T) {
	output := utils.CheckConnectionGithub()
	if !output {
		t.Log("CheckConnectionGithub returned a FALSE value")
	} else {
		t.Log("CheckConnectionGithub returned a TRUE value")
	}
}

func TestCheckConnectionOpenProject(t *testing.T) {
	output := utils.CheckConnectionOpenProject()
	if !output {
		t.Log("CheckConnectionOpenProject returned a FALSE value")
	} else {
		t.Log("CheckConnectionOpenProject returned a TRUE value")
	}
}
