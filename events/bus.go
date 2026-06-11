package events

import (
	"sync"
)

// Bus est un bus d'événements pub/sub simple.
type Bus struct {
	mu         sync.RWMutex
	listeners  map[string][]chan string
}

// NewBus crée un nouveau bus d'événements.
func NewBus() *Bus {
	return &Bus{
		listeners: make(map[string][]chan string),
	}
}

// Publish publie un événement à tous les abonnés.
func (b *Bus) Publish(event string, data string) {
	b.mu.RLock()
	channels := b.listeners[event]
	b.mu.RUnlock()

	for _, ch := range channels {
		select {
		case ch <- data:
		default:
		}
	}
}

// Subscribe s'abonne à un événement.
func (b *Bus) Subscribe(event string) chan string {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan string, 10)
	b.listeners[event] = append(b.listeners[event], ch)
	return ch
}

// Unsubscribe se désabonne d'un événement.
func (b *Bus) Unsubscribe(event string, ch chan string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	channels := b.listeners[event]
	for i, c := range channels {
		if c == ch {
			b.listeners[event] = append(channels[:i], channels[i+1:]...)
			close(ch)
			return
		}
	}
}

// Destroy ferme tous les canaux et nettoie le bus.
func (b *Bus) Destroy() {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, channels := range b.listeners {
		for _, ch := range channels {
			close(ch)
		}
	}
	b.listeners = make(map[string][]chan string)
}
