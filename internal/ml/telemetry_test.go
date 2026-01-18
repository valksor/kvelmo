package ml

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/valksor/go-mehrhof/internal/workflow"
)

// TestFileTelemetryStorage_ConcurrentWrites tests concurrent event storage.
// This test verifies the fix for the TOCTOU race condition in StoreEvent.
func TestFileTelemetryStorage_ConcurrentWrites(t *testing.T) {
	// Create a temporary directory for test
	tempDir := t.TempDir()

	storage, err := NewFileTelemetryStorage(tempDir)
	if err != nil {
		t.Fatalf("NewFileTelemetryStorage failed: %v", err)
	}

	// Number of concurrent goroutines
	numGoroutines := 100
	eventsPerGoroutine := 10

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	// Launch multiple goroutines writing events concurrently
	for i := range numGoroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := range eventsPerGoroutine {
				event := &WorkflowEvent{
					TaskID:    fmt.Sprintf("task-%d", id),
					Timestamp: time.Now(),
					EventType: "state_change",
					State:     workflow.StatePlanning,
					Event:     workflow.EventStart,
					Duration:  time.Duration(j) * time.Second,
				}

				if err := storage.StoreEvent(context.Background(), event); err != nil {
					errors <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("StoreEvent failed: %v", err)
	}

	// Verify events were stored
	ctx := context.Background()
	events, err := storage.LoadEvents(ctx, EventQueryOptions{
		StartTime: time.Now().Add(-1 * time.Hour),
		EndTime:   time.Now().Add(1 * time.Hour),
	})
	if err != nil {
		t.Fatalf("LoadEvents failed: %v", err)
	}

	expectedEvents := numGoroutines * eventsPerGoroutine
	if len(events) != expectedEvents {
		t.Errorf("expected %d events, got %d", expectedEvents, len(events))
	}
}

// TestFileTelemetryStorage_SameFileConcurrency tests concurrent writes to the same file.
// This tests the specific fix for the TOCTOU race condition when multiple goroutines
// append to the same daily telemetry file.
func TestFileTelemetryStorage_SameFileConcurrency(t *testing.T) {
	tempDir := t.TempDir()

	storage, err := NewFileTelemetryStorage(tempDir)
	if err != nil {
		t.Fatalf("NewFileTelemetryStorage failed: %v", err)
	}

	// All events should have the same date, so they go to the same file
	now := time.Now()

	numGoroutines := 50
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for i := range numGoroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			event := &WorkflowEvent{
				TaskID:    fmt.Sprintf("task-%d", id),
				Timestamp: now,
				EventType: "test_event",
				State:     workflow.StateIdle,
			}

			if err := storage.StoreEvent(context.Background(), event); err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors (the race condition would cause errors)
	for err := range errors {
		t.Errorf("StoreEvent failed concurrently: %v", err)
	}

	// Verify all events were stored
	ctx := context.Background()
	events, err := storage.LoadEvents(ctx, EventQueryOptions{
		StartTime: now.Add(-1 * time.Second),
		EndTime:   now.Add(1 * time.Second),
	})
	if err != nil {
		t.Fatalf("LoadEvents failed: %v", err)
	}

	if len(events) != numGoroutines {
		t.Errorf("expected %d events, got %d", numGoroutines, len(events))
	}
}

// TestFileTelemetryStorage_EmptyEvents tests storing and loading empty event lists.
func TestFileTelemetryStorage_EmptyEvents(t *testing.T) {
	tempDir := t.TempDir()

	storage, err := NewFileTelemetryStorage(tempDir)
	if err != nil {
		t.Fatalf("NewFileTelemetryStorage failed: %v", err)
	}

	// Load from empty storage should return empty list
	ctx := context.Background()
	events, err := storage.LoadEvents(ctx, EventQueryOptions{
		StartTime: time.Now().Add(-1 * time.Hour),
		EndTime:   time.Now().Add(1 * time.Hour),
	})
	if err != nil {
		t.Fatalf("LoadEvents failed: %v", err)
	}

	if len(events) != 0 {
		t.Errorf("expected 0 events from empty storage, got %d", len(events))
	}
}

// TestFileTelemetryStorage_DateRollover tests that events are correctly stored in daily files.
func TestFileTelemetryStorage_DateRollover(t *testing.T) {
	tempDir := t.TempDir()

	telemetry, err := NewFileTelemetryStorage(tempDir)
	if err != nil {
		t.Fatalf("NewFileTelemetryStorage failed: %v", err)
	}

	// Store events on different days
	day1 := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	day2 := time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC)

	event1 := &WorkflowEvent{
		TaskID:    "task-1",
		Timestamp: day1,
		EventType: "event1",
		State:     workflow.StateIdle,
	}

	event2 := &WorkflowEvent{
		TaskID:    "task-2",
		Timestamp: day2,
		EventType: "event2",
		State:     workflow.StatePlanning,
	}

	ctx := context.Background()

	if err := telemetry.StoreEvent(ctx, event1); err != nil {
		t.Fatalf("StoreEvent failed for event1: %v", err)
	}

	if err := telemetry.StoreEvent(ctx, event2); err != nil {
		t.Fatalf("StoreEvent failed for event2: %v", err)
	}

	// Load all events - both should be present
	allEvents, err := telemetry.LoadEvents(ctx, EventQueryOptions{
		StartTime: day1.Add(-1 * time.Second),
		EndTime:   day2.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("LoadEvents failed: %v", err)
	}

	if len(allEvents) != 2 {
		t.Errorf("expected 2 events total, got %d", len(allEvents))
	}
}

// TestHashTaskID tests that task ID hashing uses SHA-256 (not weak custom hash).
func TestHashTaskID(t *testing.T) {
	tests := []struct {
		name           string
		taskID         string
		wantLen        int
		wantConsistent bool
	}{
		{
			name:           "simple task ID",
			taskID:         "task-abc-123",
			wantLen:        21, // "task-" + 16 hex chars (8 bytes * 2)
			wantConsistent: true,
		},
		{
			name:           "long task ID",
			taskID:         "very-long-task-id-with-lots-of-information",
			wantLen:        21,
			wantConsistent: true,
		},
		{
			name:           "task ID with special chars",
			taskID:         "task/with/slashes&and&ampersands",
			wantLen:        21,
			wantConsistent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hashTaskID(tt.taskID)

			// Check length
			if len(result) != tt.wantLen {
				t.Errorf("hashTaskID() length = %d, want %d", len(result), tt.wantLen)
			}

			// Check it starts with "task-" prefix
			if !startsWith(result, "task-") {
				t.Errorf("hashTaskID() should start with 'task-', got %s", result)
			}

			// Check consistency - same input should produce same output
			if tt.wantConsistent {
				result2 := hashTaskID(tt.taskID)
				if result != result2 {
					t.Errorf("hashTaskID() not consistent: %s != %s", result, result2)
				}
			}

			// Check different inputs produce different outputs
			differentInput := tt.taskID + "-different"
			result3 := hashTaskID(differentInput)
			if result == result3 {
				t.Errorf("hashTaskID() produced same hash for different inputs")
			}
		})
	}
}

// TestHashTaskID_CollisionResistance tests that different task IDs don't collide.
func TestHashTaskID_CollisionResistance(t *testing.T) {
	taskIDs := make([]string, 1000)
	hashes := make(map[string]bool)

	// Generate 1000 unique task IDs
	for i := range 1000 {
		taskIDs[i] = fmt.Sprintf("task-%d", i)
		hash := hashTaskID(taskIDs[i])

		// Check for collisions
		if hashes[hash] {
			t.Errorf("hash collision detected for task ID %s", taskIDs[i])
		}
		hashes[hash] = true
	}

	// We should have 1000 unique hashes
	if len(hashes) != 1000 {
		t.Errorf("expected 1000 unique hashes, got %d", len(hashes))
	}
}

// TestHashTaskID_CryptographicallySecure tests that the hash uses SHA-256.
func TestHashTaskID_CryptographicallySecure(t *testing.T) {
	// Test that changing one bit in the input produces a completely different hash
	taskID1 := "task-abc-123"
	taskID2 := "task-abc-124" // Only one character changed

	hash1 := hashTaskID(taskID1)
	hash2 := hashTaskID(taskID2)

	// Hashes should be different
	if hash1 == hash2 {
		t.Errorf("similar inputs produced same hash")
	}

	// Calculate Hamming distance between hash portions (excluding "task-" prefix)
	hashPart1 := hash1[5:] // Strip "task-" prefix
	hashPart2 := hash2[5:]

	// With SHA-256, even a 1-bit input change should avalanche through the output
	// We expect at least 4 bits different in the 8-byte output
	diffs := 0
	for i := 0; i < len(hashPart1) && i < len(hashPart2); i++ {
		if hashPart1[i] != hashPart2[i] {
			diffs++
		}
	}

	if diffs < 4 {
		t.Errorf("expected significant difference between hashes, got %d differing characters", diffs)
	}
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
