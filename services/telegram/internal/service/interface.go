package service

import (
	"context"

	"github.com/ummuys/pacttelegramservice/services/telegram/internal/tgapi"
)

type TelegramService interface {
	CreateSession() tgapi.SessionInfoCh
	SubmitPassword(sessionID, password string) error
	SendMessage(ctx context.Context, sessionID, peer, txt string) (int64, error)
	SubscribeMessages(ctx context.Context, sessionID string) (<-chan tgapi.BroadcastMessage, error)
}
