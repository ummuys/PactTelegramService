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

	ErrFloodWait             = errors.New("too many requests, try again later")
	ErrPhoneCodeInvalid      = errors.New("invalid phone code")
	ErrPhoneCodeExpired      = errors.New("phone code expired")
	ErrPhoneNumberInvalid    = errors.New("invalid phone number")
	ErrPhoneNumberFlood      = errors.New("too many attempts, try again later")
	ErrUnauthorized          = errors.New("unauthorized")
	ErrSignUpRequired        = errors.New("sign up required")
	ErrAccountBanned         = errors.New("account is banned")
	ErrAccountDeactivated    = errors.New("account is deactivated")
	ErrAuthRestart           = errors.New("auth restart required")
	ErrAppUpdateRequired     = errors.New("update app to login")
	ErrAppCredentialsInvalid = errors.New("invalid app credentials (api_id/api_hash)")
)
