package chat

import (
	"context"
	"io"
	"testing"

	chatpb "github.com/grigory222/go-chat-proto/gen/go/proto"
)

// stubStream minimal implementation for hub tests
type stubStream struct {
	chatpb.ChatService_JoinChatServer
}

func (s *stubStream) Context() context.Context               { return context.Background() }
func (s *stubStream) Recv() (*chatpb.JoinChatRequest, error) { return nil, io.EOF }
func (s *stubStream) Send(*chatpb.Message) error             { return nil }

func TestHubRegisterUnregister(t *testing.T) {
	h := NewHub(testLogger())
	msgCh, doneCh := h.Register(1, 10, &stubStream{})
	if msgCh == nil || doneCh == nil {
		t.Fatalf("expected channels")
	}
	h.Unregister(1, 10) // should close doneCh
}

func TestHubBroadcast(t *testing.T) {
	h := NewHub(testLogger())
	_, _ = h.Register(1, 1, &stubStream{})
	ch2, _ := h.Register(1, 2, &stubStream{})

	msg := &chatpb.Message{ChatId: 1, Text: "Hello"}
	h.Broadcast(msg, 1)

	select {
	case <-ch2:
		// received
	default:
		t.Fatalf("expected user2 to receive message")
	}

	// Fill channel for user2 to capacity
	for i := 0; i < 11; i++ {
		h.Broadcast(msg, 1)
	} // Should not block even when channel is full
}
