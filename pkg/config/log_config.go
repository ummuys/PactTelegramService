package config

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
)

func ParseLogLevel(nameConf string) (zerolog.Level, error) {

	lvl, err := validateLevel(nameConf)
	if err != nil {
		return 0, err
	}

	return lvl, nil
}

func validateLevel(path string) (zerolog.Level, error) {
	env := os.Getenv(path)
	if env == "" {
		return 0, fmt.Errorf("%s is empty", path)
	}

	level, err := zerolog.ParseLevel(env)
	if err != nil {
		return 0, err
	}

	return level, nil
}
