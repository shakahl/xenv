package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/ghodss/yaml"
)

// FlatEnv provides a flattend list of key/values based on a hierarchy
// that might be found in a YAML or JSON file.
type FlatEnv struct {
	// Path to the YAML / JSON file.
	Path string

	// Env maintains the map[string]string of the flattened data.
	Env map[string]string
}

func (env *FlatEnv) key(parts []string) (string, error) {
	if len(parts) == 0 {
		return "", errors.New("no prefix for key")
	}
	return strings.Join(parts, "_"), nil
}

func (env *FlatEnv) addString(prefix []string, value string) error {
	key, err := env.key(prefix)
	if err != nil || key == "" {
		return err
	}

	if _, ok := env.Env[key]; ok {
		value = fmt.Sprintf("%s %s", env.Env[key], value)
	}
	env.Env[key] = value
	return nil
}

func (env *FlatEnv) addBool(prefix []string, value bool) error {
	return env.addString(prefix, fmt.Sprintf("%t", value))
}

func (env *FlatEnv) addFloat64(prefix []string, value float64) error {
	return env.addString(prefix, strconv.FormatFloat(value, 'f', -1, 64))
}

// Load takes an interface and applies the data to the Env. The prefix
// allows a list of fields to be combined into a prefix for the keys
// found.
func (env *FlatEnv) Load(v interface{}, prefix []string) error {
	iterMap := func(x map[string]interface{}, prefix []string) {
		for k, v := range x {
			env.Load(v, append(prefix, k))
		}
	}

	iterSlice := func(x []interface{}, prefix []string) {
		for _, v := range x {
			env.Load(v, prefix)
		}
	}

	switch vv := v.(type) {
	case string:
		if err := env.addString(prefix, v.(string)); err != nil {
			return err
		}
	case bool:
		if err := env.addBool(prefix, v.(bool)); err != nil {
			return err
		}
	case float64:
		if err := env.addFloat64(prefix, v.(float64)); err != nil {
			return err
		}
	case map[string]interface{}:
		iterMap(vv, prefix)
	case []interface{}:
		iterSlice(vv, prefix)
	default:
		return fmt.Errorf("Unknown type: %#v", vv)
	}

	return nil
}

// Decode reads the FlatEnv's path to create an interface{} for loading.
func (env *FlatEnv) Decode() (interface{}, error) {
	b, err := ioutil.ReadFile(env.Path)
	if err != nil {
		return nil, err
	}

	var f interface{}

	err = yaml.Unmarshal(b, &f)
	if err != nil {
		return nil, err
	}

	return f, nil
}

// NewFlatEnv creates a new FlatEnv, loads the provided file, flattens
// the result and returns it as a map[string]string.
func NewFlatEnv(path string) (map[string]string, error) {
	env := &FlatEnv{
		Path: path,
		Env:  make(map[string]string),
	}

	f, err := env.Decode()
	if err != nil {
		return nil, err
	}

	err = env.Load(f, []string{})
	if err != nil {
		return nil, err
	}
	return env.Env, nil
}
