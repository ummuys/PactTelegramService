package tgapi

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth/qrlogin"
	"github.com/gotd/td/tg"
	"github.com/rs/zerolog"
	"github.com/ummuys/pacttelegramservice/services/telegram/internal/errs"
)

type LiveSession struct {
	sessionID string

	appID   int
	appHash string

	logger zerolog.Logger

	storage *session.StorageMemory // хранение инфы о сессии после авторизации (для бд)

	hub *broadcastHub // комната для прослушивания сообщений

	qrCh    chan string       // канал для получения qrCode
	stateCh chan SessionState // канал для получения состояния клиента
	errCh   chan error        // канал для получения ошибок
	cmdCh   chan commands     // канал для получения команд после авторизации
	passCh  chan passReq      // канал для получения пароля

	cancel context.CancelFunc // закончить сессию

}

type passReq struct {
	password string
	resp     chan error
}

func (ls *LiveSession) run(ctx context.Context) {

	dispatcher := tg.NewUpdateDispatcher()
	cli := telegram.NewClient(ls.appID, ls.appHash, telegram.Options{
		UpdateHandler:  dispatcher,
		SessionStorage: ls.storage,
	})

	_ = cli.Run(ctx, func(rctx context.Context) error {

		err := ls.stepGetQr(ctx, cli, dispatcher)
		if err != nil {
			ls.stateCh <- StateNeedPassword
			err := ls.stepGetPasswordFor2FA(ctx, cli)
			if err != nil {
				return err
			}
		} else {
			ls.stateCh <- StateAuthSuccessful
		}

		dispatcher.OnNewMessage(func(ctx context.Context, e tg.Entities, update *tg.UpdateNewMessage) error {

			msg, ok := update.Message.(*tg.Message)
			if !ok {
				return nil
			}

			bm := BroadcastMessage{
				MessageID: int64(msg.ID),
				Text:      msg.Message,
				From:      "", // можно резолвить из e.Users/e.Chats по FromID
				Timestamp: time.Unix(int64(msg.Date), 0),
			}

			ls.hub.Broadcast(bm)
			return nil
		})

		api := cli.API()
		for {
			select {
			case <-ctx.Done():
				return nil
			case task := <-ls.cmdCh:
				task.run(ctx, api)
			}
		}

	})

}

func (ls *LiveSession) stepGetQr(ctx context.Context, cli *telegram.Client, dispatcher tg.UpdateDispatcher) error {
	tries := 0
	attempt := 10
	accept := qrlogin.OnLoginToken(dispatcher)
	show := func(ctx context.Context, token qrlogin.Token) error {
		if tries >= attempt {
			return errs.ErrQrTimeout
		}
		tries++
		ls.qrCh <- token.URL()
		return nil
	}
	_, err := cli.QR().Auth(ctx, accept, show)
	return err
}

func (ls *LiveSession) stepGetPasswordFor2FA(ctx context.Context, cli *telegram.Client) error {
	for {

		msg, ok := <-ls.passCh
		if !ok {
			return fmt.Errorf("passCh closed")
		}

		_, err := cli.Auth().Password(ctx, msg.password)
		if err != nil {
			msg.resp <- errs.ErrInvalidPassword
			continue
		}

		msg.resp <- nil
		return nil

	}
}

func (ls *LiveSession) SendMessage(ctx context.Context, peer, text string) (int64, error) {

	respCh := make(chan sendMsgResp, 1)
	cmd := sendMessage{
		peer: peer,
		text: text,
		ch:   respCh,
	}

	select {
	case ls.cmdCh <- cmd:
	case <-ctx.Done():
		return -1, ctx.Err()
	}

	select {
	case resp := <-respCh:
		fmt.Println(resp.err)
		return resp.msgID, resp.err
	case <-ctx.Done():
		return -1, ctx.Err()
	}
}

func (ls *LiveSession) SubscribeMessages(ctx context.Context) <-chan BroadcastMessage {
	listenerID := uuid.New().String()

	hub := ls.hub

	ch, unsub := hub.Subscribe(listenerID)

	go func() {
		<-ctx.Done()
		unsub()
	}()

	return ch
}
func (ls *LiveSession) Close() error { return nil }
