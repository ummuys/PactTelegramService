package tgapi

import (
	"context"
)

type Session interface {
	SendMessage(ctx context.Context, peer, text string) (int64, error)
	SubscribeMessages(ctx context.Context) <-chan BroadcastMessage
	close()
}
