package service

import (
	"github.com/rs/zerolog"
	"github.com/ummuys/pacttelegramservice/services/telegram/internal/repository"
	"github.com/ummuys/pacttelegramservice/services/telegram/internal/tgapi"
)

type TelegramService interface {
	CreateSession() tgapi.SessionInfoCh
	SubmitPassword(sessionID, password string) error
	SendMessage()
	SubscribeMessages()
}

type telegramService struct {
	sessionManager tgapi.SessionManager
	repos          repository.SessionRepository
	logger         zerolog.Logger
}

func (ts *telegramService) CreateSession() tgapi.SessionInfoCh {
	return ts.CreateSession()
}

func (ts *telegramService) SubmitPassword(sessionID, password string) error {
	return ts.sessionManager.SubmitPassword(sessionID, password)
}
