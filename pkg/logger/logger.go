package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/ummuys/pacttelegramservice/pkg/config"
)

func InitLogger(compName string, confName string) (zerolog.Logger, error) {
	zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }
	zerolog.DurationFieldUnit = time.Millisecond
	zerolog.DurationFieldInteger = true
	const consoleTimeFormat = "02 Jan 06 15:04 MST"

	cw := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: consoleTimeFormat,
	}

	baseLog := zerolog.New(cw).With().Timestamp().Logger()

	lvl, err := config.ParseLogLevel(confName)
	if err != nil {
		return zerolog.Logger{}, err
	}

	log := baseLog.With().Str("service", compName).Logger().Level(lvl)

	return log, nil
}
