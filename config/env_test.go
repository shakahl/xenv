package config_test

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/ionrock/xenv/config"
)

func TestDataHandler(t *testing.T) {
	cfgs := []config.XeConfig{}
	e := config.NewEnvironment(".", cfgs)

	e.DataHandler(config.XeConfig{Env: map[string]string{"FOO": "foo"}})

	if _, ok := e.Config.Get("FOO"); !ok {
		t.Error("error mapping EnvScript to data")
	}
}

func scriptCmd(name string, s ...string) []string {
	cmd := []string{os.Args[0], fmt.Sprintf("-test.run=%s", name)}
	if len(s) == 0 {
		return cmd
	}
	cmd = append(cmd, "--")
	cmd = append(cmd, s...)
	return cmd
}

func scriptHelper(name string, env []string) *exec.Cmd {
	parts := scriptCmd(name)
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	cmd.Env = append(cmd.Env, env...)
	return cmd
}

func TestScriptEchoBar(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	fmt.Fprintf(os.Stdout, os.Getenv("BAR"))
	os.Exit(0)
}

func TestSetEnv(t *testing.T) {
	e := config.NewEnvironment(".", []config.XeConfig{})

	// Set bar to a value we'll use in our script to ensure that our
	// config is linearly applied throughout the processing.
	e.SetEnv("BAR", "hello")

	// Set our process envvar to skip the test
	e.SetEnv("GO_WANT_HELPER_PROCESS", "1")

	// Get our command using our test executable
	val := fmt.Sprintf("`%s`", strings.Join(scriptCmd("TestScriptEchoBar"), " "))

	// Set the result to FOO
	err := e.SetEnv("FOO", val)

	if err != nil {
		t.Fatalf("error running script: %s", err)
	}

	val, ok := e.Config.Get("FOO")
	if !ok {
		t.Errorf("error setting env var FOO from script")
	}

	if val != "hello" {
		t.Errorf("error with script result: %s != hello", val)
	}
}
