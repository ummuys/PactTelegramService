package config

import (
	"errors"
	"strings"
)

type TelegramServiceConfig struct {
	Network string
	Port    string
	AppID   int
	AppHash string
}

func ParseTelegramServiceConfig() (TelegramServiceConfig, error) {

	var errs []string
	add := func(err error) {
		if err != nil {
			errs = append(errs, err.Error())
		}
	}

	network, err := parseStr("TELEGRAM_SERVICE_NETWORK")
	add(err)

	port, err := parseStr("TELEGRAM_SERVICE_PORT")
	add(err)

	id, err := parseInt("TELEGRAM_SERVICE_APP_ID", false)
	add(err)

	hash, err := parseStr("TELEGRAM_SERVICE_APP_HASH")
	add(err)

	if len(errs) > 0 {
		return TelegramServiceConfig{}, errors.New(strings.Join(errs, ", "))
	}

	return TelegramServiceConfig{Network: network, Port: ":" + port, AppID: id, AppHash: hash}, nil
}
