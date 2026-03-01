package config

import (
	"fmt"
	"os"
	"strconv"
)

func parseStr(path string) (string, error) {
	env := os.Getenv(path)
	if env == "" {
		return "", fmt.Errorf("%s is empty", path)
	}

	return env, nil
}

func parseInt(path string, canBeZero bool) (int, error) {
	env := os.Getenv(path)
	intEnv, err := strconv.Atoi(env)
	if err != nil {
		return 0, fmt.Errorf("%s is empty", path)
	}
	if !canBeZero && intEnv == 0 {
		return 0, fmt.Errorf("%s is invalid", path)
	}

	return intEnv, nil
}
