package tgapi

import (
	"context"
	"sync"
	"time"
)

type Session interface {
	SendMessage(ctx context.Context, peer, text string) (int64, error)
	SubscribeMessages(ctx context.Context) <-chan BroadcastMessage
	Close() error
}

// ------------- BROADCAST -------------

type BroadcastMessage struct {
	MessageID int64
	Text      string
	From      string
	Timestamp time.Time
}

type broadcastHub struct {
	mu   sync.RWMutex
	subs map[string]chan BroadcastMessage // айди слушателя возвращает канал слушателя
}

// -------- hub methods --------
func (h *broadcastHub) Subscribe(listernerID string) (<-chan BroadcastMessage, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()

	c := make(chan BroadcastMessage, 100)
	h.subs[listernerID] = c

	unsub := func() {
		h.mu.Lock()
		ch, ok := h.subs[listernerID]
		if ok {
			delete(h.subs, listernerID)
			close(ch)
		}
		h.mu.Unlock()
	}

	return c, unsub
}

func (h *broadcastHub) Broadcast(msg BroadcastMessage) {
	h.mu.RLock()
	chans := make([]chan BroadcastMessage, 0, len(h.subs))
	for _, ch := range h.subs {
		chans = append(chans, ch)
	}
	h.mu.RUnlock()

	// рассылка без лока
	for _, ch := range chans {
		select {
		case ch <- msg:
		default:

		}
	}
}
