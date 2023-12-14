package util

import (
	"fmt"
	"os"
)

func GetEnvConfigs(names []string) (map[string]string, error) {

	res := make(map[string]string)

	for _, name := range names {
		config, err := GetEnvConfig(name)
		if err != nil {
			return nil, fmt.Errorf("Aborting parsing of configs from env: %w", err)
		} else {
			res[name] = config
		}
	}

	return res, nil
}

func GetEnvConfig(name string) (string, error) {

	value := os.Getenv(name)
	if value == "" {
		return "", fmt.Errorf("No environment variable with name %v", name)
	}

	return value, nil
}
