package chat

import (
	"testing"

	chatpb "github.com/grigory222/go-chat-proto/gen/go/proto"
)

type mockSubscriber struct {
	id       int64
	received []*chatpb.Message
	closed   bool
}

func (m *mockSubscriber) Notify(msg *chatpb.Message) {
	m.received = append(m.received, msg)
}

func (m *mockSubscriber) ID() int64 {
	return m.id
}

func (m *mockSubscriber) Close() {
	m.closed = true
}

func TestPublisherRegisterUnregister(t *testing.T) {
	p := NewPublisher(testLogger())
	sub := &mockSubscriber{id: 10}

	p.Register(1, sub)

	if len(p.subscribers[1]) != 1 {
		t.Fatalf("expected 1 subscriber, got %d", len(p.subscribers[1]))
	}

	p.Unregister(1, 10)

	if !sub.closed {
		t.Fatal("expected subscriber to be closed")
	}

	if _, ok := p.subscribers[1]; ok {
		t.Fatal("expected chat to be removed from subscribers map")
	}
}

func TestPublisherBroadcast(t *testing.T) {
	p := NewPublisher(testLogger())
	sub1 := &mockSubscriber{id: 1}
	sub2 := &mockSubscriber{id: 2}

	p.Register(1, sub1)
	p.Register(1, sub2)

	msg := &chatpb.Message{ChatId: 1, Text: "Hello"}
	p.Broadcast(msg, 1)

	if len(sub1.received) != 0 {
		t.Fatal("sender should not receive own message")
	}

	if len(sub2.received) != 1 {
		t.Fatalf("expected user2 to receive 1 message, got %d", len(sub2.received))
	}

	if sub2.received[0].Text != "Hello" {
		t.Fatalf("expected 'Hello', got '%s'", sub2.received[0].Text)
	}
}
