package tgapi

import (
	"sync"
	"time"
)

type BroadcastMessage struct {
	MessageID int64
	Text      string
	From      string
	Timestamp time.Time
}

type broadcastHub struct {
	mu     sync.RWMutex
	subs   map[string]chan BroadcastMessage // айди слушателя возвращает канал слушателя
	closed bool                             // нужно для того, чтобы во время удаления не добавлялись бы новые
}

// -------- hub methods --------
func (h *broadcastHub) Subscribe(listernerID string) (<-chan BroadcastMessage, func(), bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return nil, nil, false
	}

	c := make(chan BroadcastMessage, 100)
	h.subs[listernerID] = c

	unsub := func() {
		h.mu.Lock()
		delete(h.subs, listernerID)
		h.mu.Unlock()
	}

	return c, unsub, true
}

func (h *broadcastHub) Broadcast(msg BroadcastMessage) bool {
	h.mu.RLock()
	if h.closed {
		h.mu.Unlock()
		return false
	}

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
	return true
}

func (h *broadcastHub) Close() {
	h.mu.Lock()
	h.closed = true
	h.subs = make(map[string]chan BroadcastMessage)
	h.mu.Unlock()
}
