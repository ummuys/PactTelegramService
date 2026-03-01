package telegramapi

import (
	"context"
	"sync"

	"github.com/ummuys/pacttelegramservice/services/telegram/internal/errs"
)

type authManager struct {
	mu sync.Mutex
	m  map[string]authSession
}

type authSession struct {
	passMsgCh chan passMsg
	ctx       context.Context
	cancel    context.CancelFunc
}

type passMsg struct {
	password string
	errCh    chan error
}

func newAuthManager() *authManager {
	return &authManager{
		m: make(map[string]authSession),
	}
}

func (am *authManager) newAuthSession(sessionID string, ctx context.Context, cancel context.CancelFunc) {

	as := authSession{
		ctx:       ctx,
		passMsgCh: make(chan passMsg, 1),
		cancel:    cancel,
	}

	am.mu.Lock()
	am.m[sessionID] = as
	am.mu.Unlock()
}

func (am *authManager) sendPassword(sessionID string, password string) error {

	am.mu.Lock()
	authSession, ok := am.m[sessionID]
	am.mu.Unlock()

	if !ok {
		return errs.ErrSessionNotFound
	}

	ch := make(chan error, 1)

	msg := passMsg{
		password: password,
		errCh:    ch,
	}

	actx := authSession.ctx

	select {
	case <-actx.Done():
		am.closeAuthSession(sessionID)
		return actx.Err()
	case authSession.passMsgCh <- msg:
	}

	select {
	case err := <-ch:
		return err
	case <-actx.Done():
		am.closeAuthSession(sessionID)
		return actx.Err()
	}
}

func (am *authManager) waitPassword(sessionID string) (string, chan error, error) {
	am.mu.Lock()
	authSession, ok := am.m[sessionID]
	am.mu.Unlock()

	if !ok {
		return "", nil, errs.ErrSessionNotFound
	}

	actx := authSession.ctx

	select {
	case msg, ok := <-authSession.passMsgCh:
		if !ok {
			return "", nil, errs.ErrSessionNotFound
		}
		return msg.password, msg.errCh, nil

	case <-actx.Done():
		return "", nil, actx.Err()
	}
}

func (am *authManager) closeAuthSession(sessionID string) {
	am.mu.Lock()
	as, ok := am.m[sessionID]
	if ok {
		delete(am.m, sessionID)
	}
	am.mu.Unlock()

	if ok {
		as.cancel()
	}
}
