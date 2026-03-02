package tgapi

import (
	"context"
	"fmt"
	"sync"
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

	closeCh   chan struct{} // канал для получения сигнала о завершения всех действий
	closeOnce sync.Once     // чтобы не вызвать панику

	cancel context.CancelFunc // закончить сессию

}

type passReq struct {
	password string
	resp     chan error
}

func (ls *LiveSession) run(ctx context.Context) {

	dispatcher := tg.NewUpdateDispatcher()

	// Подписка на получение сообщений
	dispatcher.OnNewMessage(func(ctx context.Context, e tg.Entities, update *tg.UpdateNewMessage) error {

		msg, ok := update.Message.(*tg.Message)
		if !ok {
			return nil
		}

		bm := BroadcastMessage{
			MessageID: int64(msg.ID),
			Text:      msg.Message,
			From:      peerToName(e, msg.FromID),
			Timestamp: time.Unix(int64(msg.Date), 0),
		}

		if open := ls.hub.Broadcast(bm); !open {
			return nil
		}

		return nil
	})

	cli := telegram.NewClient(ls.appID, ls.appHash, telegram.Options{
		UpdateHandler:  dispatcher,
		SessionStorage: ls.storage,
	})

	_ = cli.Run(ctx, func(rctx context.Context) error {

		err := ls.stepGetQr(rctx, cli, dispatcher)
		if err != nil {
			select {
			case ls.stateCh <- StateNeedPassword:
			default:
			}
			err := ls.stepGetPasswordFor2FA(rctx, cli)
			if err != nil {
				return err
			}
		} else {
			select {
			case ls.stateCh <- StateAuthSuccessful:
			default:
			}
		}

		api := cli.API()
		for {
			select {
			case <-rctx.Done():
				return rctx.Err()
			case <-ls.closeCh:
				return errs.ErrSessionClosed
			case task := <-ls.cmdCh:
				task.run(rctx, api)
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
		select {
		case ls.qrCh <- token.URL():
		default:
		}
		return nil
	}
	_, err := cli.QR().Auth(ctx, accept, show)
	return err
}

func (ls *LiveSession) stepGetPasswordFor2FA(ctx context.Context, cli *telegram.Client) error {
	for {

		select {
		case <-ls.closeCh:
			return errs.ErrSessionClosed
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-ls.passCh:
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
	case <-ls.closeCh:
		return -1, errs.ErrSessionClosed
	}

	select {
	case resp := <-respCh:
		return resp.msgID, resp.err
	case <-ctx.Done():
		return -1, ctx.Err()
	case <-ls.closeCh:
		return -1, errs.ErrSessionClosed
	}
}

func (ls *LiveSession) SubscribeMessages(ctx context.Context) (<-chan BroadcastMessage, error) {
	listenerID := uuid.New().String()

	hub := ls.hub

	ch, unsub, success := hub.Subscribe(listenerID)
	if !success {
		return nil, errs.ErrSessionBroadcastClosed
	}

	go func() {
		<-ctx.Done()
		unsub()
	}()

	return ch, nil
}
func (ls *LiveSession) Close() {
	ls.closeOnce.Do(func() { close(ls.closeCh) })

	if ls.cancel != nil {
		ls.cancel()
	}

	ls.hub.Close()
}
