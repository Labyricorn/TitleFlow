// Package hub implements a simple SSE (Server-Sent Events) publish/subscribe hub.
// It is used to push animation commands from the dock to connected runtime pages.
package hub

import (
	"fmt"
	"sync"
)

// Hub manages a set of SSE subscriber channels and broadcasts events to all of them.
type Hub struct {
	mu          sync.Mutex
	subscribers map[chan string]struct{}
}

// New returns a ready-to-use Hub.
func New() *Hub {
	return &Hub{
		subscribers: make(map[chan string]struct{}),
	}
}

// Subscribe registers a new subscriber and returns its receive channel.
// The channel is buffered to avoid blocking the publisher on slow clients.
func (h *Hub) Subscribe() chan string {
	ch := make(chan string, 4)
	h.mu.Lock()
	h.subscribers[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

// Unsubscribe removes a subscriber and closes its channel.
func (h *Hub) Unsubscribe(ch chan string) {
	h.mu.Lock()
	delete(h.subscribers, ch)
	h.mu.Unlock()
	close(ch)
}

// Publish sends an SSE-formatted message to all current subscribers.
// eventName is the SSE event name; jsonData is the JSON payload for the data field.
// Slow subscribers that cannot receive within the channel buffer are dropped silently.
func (h *Hub) Publish(eventName, jsonData string) {
	msg := fmt.Sprintf("event: %s\ndata: %s\n\n", eventName, jsonData)
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subscribers {
		select {
		case ch <- msg:
		default:
			// Subscriber is too slow; skip rather than block.
		}
	}
}
