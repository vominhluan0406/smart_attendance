package event

import (
	"encoding/json"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

// Bus wraps a NATS connection for pub/sub.
type Bus struct {
	conn *nats.Conn
}

// Connect creates a new event bus connected to NATS.
// Returns nil (not error) if NATS URL is empty — allows running without broker.
func Connect(natsURL string) *Bus {
	if natsURL == "" {
		log.Printf("[event] NATS_URL not set, running without event bus")
		return nil
	}

	opts := []nats.Option{
		nats.Name("smart-attendance"),
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(-1), // Reconnect forever
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			log.Printf("[event] NATS disconnected: %v", err)
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			log.Printf("[event] NATS reconnected")
		}),
	}

	conn, err := nats.Connect(natsURL, opts...)
	if err != nil {
		log.Printf("[event] WARNING: NATS connection failed: %v (running without event bus)", err)
		return nil
	}

	log.Printf("[event] connected to NATS: %s", natsURL)
	return &Bus{conn: conn}
}

// Publish sends an event to a subject. No-op if bus is nil.
func (b *Bus) Publish(subject string, payload interface{}) {
	if b == nil || b.conn == nil {
		return
	}

	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[event] ERROR marshal %s: %v", subject, err)
		return
	}

	if err := b.conn.Publish(subject, data); err != nil {
		log.Printf("[event] ERROR publish %s: %v", subject, err)
		return
	}

	log.Printf("[event] published: %s (%d bytes)", subject, len(data))
}

// Subscribe listens for events on a subject. handler receives raw JSON bytes.
// Returns the subscription (caller can Unsubscribe). No-op if bus is nil.
func (b *Bus) Subscribe(subject string, handler func(data []byte)) {
	if b == nil || b.conn == nil {
		return
	}

	_, err := b.conn.Subscribe(subject, func(msg *nats.Msg) {
		handler(msg.Data)
	})
	if err != nil {
		log.Printf("[event] ERROR subscribe %s: %v", subject, err)
		return
	}

	log.Printf("[event] subscribed: %s", subject)
}

// SubscribeJSON is a typed helper — decodes JSON into target type before calling handler.
func SubscribeJSON[T any](b *Bus, subject string, handler func(event T)) {
	if b == nil {
		return
	}
	b.Subscribe(subject, func(data []byte) {
		var ev T
		if err := json.Unmarshal(data, &ev); err != nil {
			log.Printf("[event] ERROR unmarshal %s: %v", subject, err)
			return
		}
		handler(ev)
	})
}

// Close disconnects from NATS.
func (b *Bus) Close() {
	if b != nil && b.conn != nil {
		b.conn.Drain()
	}
}
