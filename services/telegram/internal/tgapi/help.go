package tgapi

import (
	"errors"
	"fmt"

	gotdauth "github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"
)

// TO WORK
func isPasswordNeeded(err error) bool {
	return tgerr.Is(err, "SESSION_PASSWORD_NEEDED") ||
		(tgerr.IsCode(err, 400, 401) && tgerr.Is(err, "SESSION_PASSWORD_NEEDED"))
}

func isPasswordIncorrect(err error) bool {
	return errors.Is(err, gotdauth.ErrPasswordInvalid) ||
		tg.IsPasswordHashInvalid(err) ||
		tgerr.Is(err, "PASSWORD_HASH_INVALID")
}

func peerToString(p tg.PeerClass) string {
	if p == nil {
		return "unknown"
	}
	switch v := p.(type) {
	case *tg.PeerUser:
		return fmt.Sprintf("user:%d", v.UserID)
	case *tg.PeerChat:
		return fmt.Sprintf("chat:%d", v.ChatID)
	case *tg.PeerChannel:
		return fmt.Sprintf("channel:%d", v.ChannelID)
	default:
		return "unknown"
	}
}
