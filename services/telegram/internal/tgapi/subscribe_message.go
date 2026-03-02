package tgapi

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/gotd/td/tg"
	"github.com/ummuys/pacttelegramservice/services/telegram/internal/errs"
)

func (c *tgCli) SubscribeMessages(sessionID string, ctx context.Context) (<-chan BroadcastMessage, error) {

	data, have := c.repos.Get(sessionID)
	if !have {
		return nil, errs.ErrSessionNotFound
	}

	// 1) hub
	hub, ok := c.broadcastManager.GetHub(sessionID)
	if !ok {
		hub = c.broadcastManager.CreateHub(sessionID)
	}

	// 2) подписчик
	listenerID := uuid.New().String()
	subCh, unsubscribe, isOpen := hub.Subscribe(listenerID)
	if !isOpen {
		return nil, errs.ErrSessionNotFound
	}

	// 3) cli + dispatcher
	dispatcher := tg.NewUpdateDispatcher()
	cli := c.newClientWithStorage(data, dispatcher, ctx)

	dispatcher.OnNewMessage(func(_ context.Context, _ tg.Entities, u *tg.UpdateNewMessage) error {
		msg, ok := u.Message.(*tg.Message)
		if !ok {
			return nil
		}
		if msg.Out || msg.Message == "" {
			return nil
		}

		hub.Broadcast(BroadcastMessage{
			MessageID: int64(msg.ID),
			Text:      msg.Message,
			From:      peerToString(msg.FromID),
			Timestamp: time.Unix(int64(msg.Date), 0),
		})
		return nil
	})

	// 5) Run
	go func() {
		defer unsubscribe()

		_ = cli.Run(ctx, func(rctx context.Context) error {
			<-rctx.Done()
			return rctx.Err()
		})
	}()

	return subCh, nil
}
