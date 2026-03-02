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

func peerToName(e tg.Entities, p tg.PeerClass) string {
	if p == nil {
		return "unknown"
	}

	switch v := p.(type) {
	case *tg.PeerUser:
		u := e.Users[v.UserID]
		if u == nil {
			return fmt.Sprintf("user:%d", v.UserID)
		}
		if u.Username != "" {
			return "@" + u.Username
		}
		name := u.FirstName
		if u.LastName != "" {
			if name != "" {
				name += " "
			}
			name += u.LastName
		}
		if name == "" {
			return fmt.Sprintf("user:%d", v.UserID)
		}
		return name

	case *tg.PeerChat:
		ch := e.Chats[v.ChatID]
		if ch != nil {
			return ch.Title
		}
		return fmt.Sprintf("chat:%d", v.ChatID)

	case *tg.PeerChannel:
		ch := e.Chats[v.ChannelID]
		if ch != nil {
			return ch.Title
		}
		return fmt.Sprintf("channel:%d", v.ChannelID)

	default:
		return "unknown"
	}
}
