package tgapi

import (
	"context"
	"errors"
	"fmt"

	gotdauth "github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"
	"github.com/ummuys/pacttelegramservice/services/telegram/internal/errs"
)

func parseGotdError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, context.Canceled) {
		return errs.ErrSessionClosed
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return errs.ErrQrTimeout
	}

	if errors.Is(err, gotdauth.ErrPasswordAuthNeeded) || tgerr.Is(err, "SESSION_PASSWORD_NEEDED") {
		return errs.Err2FA
	}

	if errors.Is(err, gotdauth.ErrPasswordInvalid) ||
		tg.IsPasswordHashInvalid(err) ||
		tgerr.Is(err, "PASSWORD_HASH_INVALID") {
		return errs.ErrInvalidPassword
	}

	if _, ok := tgerr.AsFloodWait(err); ok {
		return errs.ErrFloodWait
	}

	if tgerr.Is(err, "PHONE_NUMBER_FLOOD") || tgerr.Is(err, "PHONE_CODE_FLOOD") {
		return errs.ErrPhoneNumberFlood
	}

	if tgerr.Is(err, "PHONE_CODE_INVALID") {
		return errs.ErrPhoneCodeInvalid
	}
	if tgerr.Is(err, "PHONE_CODE_EXPIRED") {
		return errs.ErrPhoneCodeExpired
	}

	if tgerr.Is(err, "PHONE_NUMBER_INVALID") {
		return errs.ErrPhoneNumberInvalid
	}

	if tgerr.Is(err, "API_ID_INVALID") {
		return errs.ErrAppCredentialsInvalid
	}
	if tgerr.Is(err, "UPDATE_APP_TO_LOGIN") {
		return errs.ErrAppUpdateRequired
	}

	if gotdauth.IsUnauthorized(err) || tgerr.IsCode(err, 401) {
		return errs.ErrUnauthorized
	}
	if tgerr.Is(err, "AUTH_RESTART") {
		return errs.ErrAuthRestart
	}

	if tgerr.Is(err, "USER_DEACTIVATED") || tgerr.Is(err, "USER_DEACTIVATED_BAN") {
		return errs.ErrAccountDeactivated
	}
	if tgerr.Is(err, "PHONE_NUMBER_BANNED") {
		return errs.ErrAccountBanned
	}

	var su *gotdauth.SignUpRequired
	if errors.As(err, &su) {
		return errs.ErrSignUpRequired
	}

	return err
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
