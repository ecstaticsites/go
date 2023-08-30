package util

import (
	"fmt"
	"os"
)

func GetEnvConfig(name string) (string, error) {
	value := os.Getenv(name)
	if (value == "") {
		return "", fmt.Errorf("No environment variable with name %v", name)
	}
	return value, nil
}
