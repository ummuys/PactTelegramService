package tgapi

import (
	"context"
	"errors"
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
	stateCh chan SessionState // канал для получения состояния клиента по авторизации
	errCh   chan error        // канал для получения ошибок по авторизации
	cmdCh   chan commands     // канал для получения команд после авторизации
	passCh  chan passReq      // канал для получения пароля

	closeCh   chan struct{} // канал для получения сигнала о завершения всех действий (нужен для того, если уже какое)
	closeOnce sync.Once     // чтобы не вызвать панику

	cancel context.CancelFunc // закончить сессию (тут храниться cancel функция того ctx, который передается в функцию run)
}

type passReq struct {
	password string
	resp     chan error
}

func (ls *LiveSession) run(ctx context.Context, authCtx context.Context) {
	defer func() {
		ls.logger.Info().
			Str("session_id", ls.sessionID).
			Msg("live session closing")
	}()

	ls.logger.Info().
		Str("session_id", ls.sessionID).
		Msg("live session start")

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

		select {
		case <-ls.closeCh:
			return nil
		default:
		}

		ls.hub.Broadcast(bm)

		return nil
	})

	cli := telegram.NewClient(ls.appID, ls.appHash, telegram.Options{
		UpdateHandler:  dispatcher,
		SessionStorage: ls.storage,
	})
	api := cli.API()

	go func() {
		<-ctx.Done()
		timer := time.NewTimer(15 * time.Second)
		defer timer.Stop()

		select {
		case <-ls.closeCh:
			return
		case <-timer.C:
			ls.close()
		}
	}()

	err := cli.Run(ctx, func(rctx context.Context) error {
		if err := ls.stepGetQr(authCtx, cli, dispatcher); err != nil {
			gerr := parseGotdError(err)

			ls.logger.Warn().
				Str("session_id", ls.sessionID).
				Err(err).
				Str("mapped_error", gerr.Error()).
				Msg("auth via qr failed")

			if errors.Is(gerr, errs.Err2FA) {

				ls.logger.Info().
					Str("session_id", ls.sessionID).
					Msg("2FA required")

				select {
				case ls.stateCh <- StateNeedPassword:
				default:
				}

				if err := ls.stepGetPasswordFor2FA(rctx, cli); err != nil {
					gerr2 := parseGotdError(err)
					ls.logger.Warn().
						Str("session_id", ls.sessionID).
						Err(err).
						Str("mapped_error", gerr2.Error()).
						Msg("2FA password step failed")

					return gerr2
				}

				ls.logger.Info().
					Str("session_id", ls.sessionID).
					Msg("auth successful after 2FA")

			} else {
				select {
				case ls.errCh <- gerr:
				default:
				}
				return gerr
			}
		} else {
			ls.logger.Info().
				Str("session_id", ls.sessionID).
				Msg("auth successful via qr")

			select {
			case ls.stateCh <- StateAuthSuccessful:
			default:
			}
		}

		for {
			select {
			case <-rctx.Done():
				ls.logger.Info().
					Str("session_id", ls.sessionID).
					Err(rctx.Err()).
					Msg("run context done")
				return rctx.Err()
			case <-ls.closeCh:
				ls.logger.Info().
					Str("session_id", ls.sessionID).
					Msg("session closed by closeCh")
				return errs.ErrSessionClosed
			case task, ok := <-ls.cmdCh:
				if !ok {
					ls.logger.Info().
						Str("session_id", ls.sessionID).
						Msg("cmd channel closed")
					return errs.ErrSessionClosed
				}

				task.run(rctx, api)
			}
		}
	})

	if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, errs.ErrSessionClosed) {
		ls.logger.Error().
			Str("session_id", ls.sessionID).
			Err(err).
			Msg("telegram client run exited with error")
	} else {
		ls.logger.Info().
			Str("session_id", ls.sessionID).
			Msg("telegram client run exited")
	}
}

func (ls *LiveSession) stepGetQr(ctx context.Context, cli *telegram.Client, dispatcher tg.UpdateDispatcher) error {
	tries := 0
	attempt := 10
	accept := qrlogin.OnLoginToken(dispatcher)

	authStepCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		select {
		case <-ls.closeCh:
			cancel()
		case <-authStepCtx.Done():
		}
	}()

	errCh := make(chan error, 1)

	show := func(ctx context.Context, token qrlogin.Token) error {
		if tries >= attempt {
			return errs.ErrQrTimeout
		}
		tries++
		select {
		case <-ls.closeCh:
			return errs.ErrSessionClosed
		case ls.qrCh <- token.URL():
		default:
		}
		return nil
	}

	go func() {
		_, err := cli.QR().Auth(authStepCtx, accept, show)
		errCh <- err
	}()

	select {
	case aerr := <-errCh:
		return aerr
	case <-ls.closeCh:
		return errs.ErrSessionClosed
	case <-authStepCtx.Done():
		return authStepCtx.Err()
	}
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
				select {
				case msg.resp <- parseGotdError(err):
				default:
				}
				continue
			}

			select {
			case msg.resp <- nil:
			default:
			}
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

func (ls *LiveSession) SubscribeMessages(ctx context.Context) <-chan BroadcastMessage {
	listenerID := uuid.New().String()

	hub := ls.hub

	ch := hub.Subscribe(listenerID)

	go func() {
		<-ctx.Done()
		hub.Unsubscribe(listenerID)
	}()

	return ch
}

func (ls *LiveSession) close() {
	d := &disconnect{errCh: make(chan error, 1)}
	ls.cmdCh <- d
	err := <-d.errCh
	if err != nil {
		ls.logger.Warn().
			Str("session_id", ls.sessionID).
			Msg("session disconnected with err")
	}

	ls.closeOnce.Do(func() { close(ls.closeCh) })

	if ls.cancel != nil {
		ls.cancel()
	}

	ls.hub.Close()

	select {
	case <-ls.hub.Done():
	case <-time.After(time.Second * 5):
		ls.logger.Warn().
			Str("session_id", ls.sessionID).
			Msg("hub did not stop within timeout")
	}
}
