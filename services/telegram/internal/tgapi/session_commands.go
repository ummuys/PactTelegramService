package tgapi

import (
	"context"

	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/tg"
)

type commands interface {
	run(rctx context.Context, api *tg.Client)
}

//

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
	_, err := s.Resolve(sm.peer).Text(rctx, sm.text)

	resp := sendMsgResp{msgID: 0, err: err}

	select {
	case sm.ch <- resp:
	case <-rctx.Done():
	}
}
