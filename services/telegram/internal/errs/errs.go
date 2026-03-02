package errs

import "errors"

var (
	ErrInternalServer         = errors.New("somefing wrong with server, try again later")
	ErrQrTimeout              = errors.New("qr timeout")
	ErrInvalidPassword        = errors.New("invalid password")
	ErrUnknownClientEvent     = errors.New("unknown client event")
	Err2FA                    = errors.New("detected 2FA, needed a password")
	ErrSessionNotFound        = errors.New("session not found")
	ErrSessionBroadcastClosed = errors.New("session broadcast is closed")
	ErrSessionClosed          = errors.New("session is closed")
)
