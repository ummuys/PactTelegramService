package tgapi

import (
	"context"
	"time"

	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/message/unpack"
	"github.com/gotd/td/tg"
)

type commands interface {
	run(rctx context.Context, api *tg.Client)
}

// send

type sendMessage struct {
	peer string
	text string
	ch   chan sendMsgResp
}

type sendMsgResp struct {
	msgID int64
	err   error
}

func (sm sendMessage) run(rctx context.Context, api *tg.Client) {
	s := message.NewSender(api)

	msgID, err := unpack.MessageID(s.Resolve(sm.peer).Text(rctx, sm.text))

	resp := sendMsgResp{msgID: int64(msgID), err: err}

	select {
	case sm.ch <- resp:
	case <-rctx.Done():
	}
}

// disconnect
type disconnect struct {
	errCh chan error
}

func (d *disconnect) run(rctx context.Context, api *tg.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	_, err := api.AuthLogOut(ctx)
	d.errCh <- err
}
