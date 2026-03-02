package service

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/ummuys/pacttelegramservice/services/telegram/internal/errs"
	"github.com/ummuys/pacttelegramservice/services/telegram/internal/repository"
	"github.com/ummuys/pacttelegramservice/services/telegram/internal/tgapi"
)

type telegramService struct {
	sessionManager tgapi.SessionManager
	repos          repository.SessionRepository
	logger         zerolog.Logger
}

func NewTelegramService(sm tgapi.SessionManager, repos repository.SessionRepository, baseLogger zerolog.Logger) TelegramService {
	logger := baseLogger.With().Str("component", "tg_svc").Logger()
	return &telegramService{
		sessionManager: sm,
		repos:          repos,
		logger:         logger,
	}
}

func (ts *telegramService) CreateSession() tgapi.SessionInfoCh {
	return ts.sessionManager.CreateSession()
}

func (ts *telegramService) SubmitPassword(sessionID, password string) error {
	return ts.sessionManager.SubmitPassword(sessionID, password)
}

func (ts *telegramService) SendMessage(ctx context.Context, sessionID, peer, txt string) (int64, error) {

	session := ts.sessionManager.Get(sessionID)
	if session == nil {
		return -1, errs.ErrSessionNotFound
	}

	msgID, err := session.SendMessage(ctx, peer, txt)
	if err != nil {
		return -1, err
	}

	return msgID, nil
}

func (ts *telegramService) SubscribeMessages(ctx context.Context, sessionID string) (<-chan tgapi.BroadcastMessage, error) {

	session := ts.sessionManager.Get(sessionID)
	if session == nil {
		return nil, errs.ErrSessionNotFound
	}

	return session.SubscribeMessages(ctx)
}
