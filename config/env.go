package config

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/ionrock/xenv/manager"
)

func findLongestServiceName(cfgs []*XeConfig) int {
	size := 0

	for _, cfg := range cfgs {
		if cfg.Service == nil {
			continue
		}

		if len(cfg.Service.Name) > size {
			size = len(cfg.Service.Name)
		}
	}

	return size
}

// Environment maintains the executable environment state.
type Environment struct {
	// Services provides a simple process manager to start/stop
	// processes along with the primary command.
	Services *manager.Manager

	// Tasks can run before and after a command.
	Tasks map[string]*exec.Cmd

	// Config provides the environment for the command.
	Config *Config

	// ConfigDir is the directory where the config file is in order to
	// provide a base for tasks / services.
	ConfigDir string

	DataOnly bool
	post     []*XeConfig
}

// NewEnvironment creates a new *Environment rooted at the provided
// directory.
func NewEnvironment(cfgDir string) *Environment {
	return &Environment{
		Services:  manager.New(),
		Tasks:     make(map[string]*exec.Cmd),
		Config:    &Config{make(map[string]string)},
		ConfigDir: cfgDir,
	}
}

// Pre runs the defined steps before the specified command.
func (e *Environment) Pre(cfgs []*XeConfig) error {
	handler := e.ConfigHandler
	if e.DataOnly {
		handler = e.DataHandler
	}

	for _, cfg := range cfgs {
		if err := handler(cfg); err != nil {
			return err
		}
	}

	return nil
}

// Post runs the defined steps after the process exits, no matter the
// exit status of the command.
func (e *Environment) Post() error {
	fmt.Println(fmt.Sprintf("post value: %#v", e.post))
	err := e.StopServices()
	if err != nil {
		return err
	}

	for _, cfg := range e.post {
		// We don't worry about using a data handler here.
		if err := e.ConfigHandler(cfg); err != nil {
			return err
		}
	}

	return nil
}

// StartService starts a process with the process manager in the environment.
func (e *Environment) StartService(name, command, dir string) error {
	return e.Services.Start(name, command, dir, e.Config.ToEnv())
}

// SetEnv sets an environment value.
func (e *Environment) SetEnv(k, v string) error {
	v = os.Expand(v, e.Config.GetConfig)
	val, err := CompileValue(v, e.ConfigDir, e.Config.ToEnv())
	if err != nil {
		fmt.Printf("error getting value for env: %q %q\n", v, err)
		return err
	}
	e.Config.Set(k, val)
	return nil
}

// SetEnvFromScript will run a script that outputs YAML or JSON,
// flatten the output and add it to the environment's configuration.
func (e *Environment) SetEnvFromScript(cmd, dir string) error {
	s := Script{
		Cmd: cmd,
		Dir: dir,
		Env: e.Config.ToEnv(),
	}

	env, err := s.Load()
	if err != nil {
		return err
	}

	for k, v := range env {
		// We expand the value if it has any vars defined. This will
		// also remove expansions that don't exist leaving things with an
		// empty string.
		val := os.Expand(v, e.Config.GetConfig)
		e.SetEnv(k, val)
	}

	return nil
}

// RunTask runs a task in the environment. The output is sent to
// stdout and is prefixed by the name of the task.
func (e *Environment) RunTask(name, command, dir string) error {
	if dir == "" {
		dir = e.ConfigDir
	}

	t := &Task{
		Name: name,
		Cmd:  command,
		Dir:  dir,
		Env:  e.Config.ToEnv(),
	}

	return t.Run()
}

// DataHandler only responds to items that update the data. This is
// used for debugging configurations.
func (e *Environment) DataHandler(cfg *XeConfig) error {
	switch {
	case cfg.Env != nil:
		for k, v := range cfg.Env {
			err := e.SetEnv(k, v)
			return err
		}

	case cfg.EnvScript != "":
		err := e.SetEnvFromScript(cfg.EnvScript, e.ConfigDir)
		if err != nil {
			return err
		}
	}

	return nil
}

// ConfigHandler calls the respective handler actionss based on the
// passed in XeConfig. It is assumed the XeConfig will only have 1 field in its struct filled in.
func (e *Environment) ConfigHandler(cfg *XeConfig) error {
	switch {
	case cfg.Service != nil:
		if cfg.Service.Dir == "" {
			cfg.Service.Dir = e.ConfigDir
		}

		err := e.StartService(cfg.Service.Name, cfg.Service.Cmd, cfg.Service.Dir)
		if err != nil {
			return err
		}

	case cfg.Env != nil:
		for k, v := range cfg.Env {
			err := e.SetEnv(k, v)
			return err
		}

	case cfg.EnvScript != "":
		err := e.SetEnvFromScript(cfg.EnvScript, e.ConfigDir)
		if err != nil {
			return err
		}

	case cfg.Task != nil:
		err := e.RunTask(cfg.Task.Name, cfg.Task.Cmd, cfg.Task.Dir)
		if err != nil {
			return err
		}

	case cfg.Post != nil:
		if e.post == nil {
			e.post = make([]*XeConfig, 0)
		}

		for _, cfg := range cfg.Post {
			e.post = append(e.post, cfg)
		}
	}

	return nil
}

// StopServices stops the services managed by the process manager.
func (e *Environment) StopServices() error {
	for name, _ := range e.Services.Processes {
		err := e.Services.Stop(name)
		if err != nil {
			return err
		}
	}

	return nil
}
