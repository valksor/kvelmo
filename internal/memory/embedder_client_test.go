package memory

import (
	"sync"
	"testing"
)

func TestEmbedderClient_SetEventPublisher(t *testing.T) {
	t.Parallel()

	client := NewEmbedderClient(EmbedderClientOptions{})

	var called bool
	var receivedMsg string
	var mu sync.Mutex

	client.SetEventPublisher(func(message string) {
		mu.Lock()
		defer mu.Unlock()
		called = true
		receivedMsg = message
	})

	// Trigger an event
	client.publishEvent("test message")

	mu.Lock()
	defer mu.Unlock()
	if !called {
		t.Error("event publisher was not called")
	}
	if receivedMsg != "test message" {
		t.Errorf("received message = %q, want %q", receivedMsg, "test message")
	}
}

func TestEmbedderClient_publishEvent_NilPublisher(t *testing.T) {
	t.Parallel()

	client := NewEmbedderClient(EmbedderClientOptions{})

	// Should not panic when publisher is nil
	client.publishEvent("test message")
}

func TestEmbedderClient_SetEventPublisher_Overwrite(t *testing.T) {
	t.Parallel()

	client := NewEmbedderClient(EmbedderClientOptions{})

	var firstCalled, secondCalled bool
	var mu sync.Mutex

	client.SetEventPublisher(func(_ string) {
		mu.Lock()
		defer mu.Unlock()
		firstCalled = true
	})

	client.SetEventPublisher(func(_ string) {
		mu.Lock()
		defer mu.Unlock()
		secondCalled = true
	})

	client.publishEvent("test")

	mu.Lock()
	defer mu.Unlock()
	if firstCalled {
		t.Error("first publisher should not have been called after overwrite")
	}
	if !secondCalled {
		t.Error("second publisher should have been called")
	}
}
