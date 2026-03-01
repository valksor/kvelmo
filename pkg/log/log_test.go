package log

import (
	"bytes"
	"strings"
	"testing"
)

func TestLevelString(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
	}

	for _, tt := range tests {
		if got := tt.level.String(); got != tt.want {
			t.Errorf("Level(%d).String() = %s, want %s", tt.level, got, tt.want)
		}
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  Level
	}{
		{"debug", LevelDebug},
		{"DEBUG", LevelDebug},
		{"info", LevelInfo},
		{"INFO", LevelInfo},
		{"warn", LevelWarn},
		{"WARN", LevelWarn},
		{"warning", LevelWarn},
		{"error", LevelError},
		{"ERROR", LevelError},
		{"unknown", LevelInfo}, // default
	}

	for _, tt := range tests {
		if got := ParseLevel(tt.input); got != tt.want {
			t.Errorf("ParseLevel(%s) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestLoggerOutput(t *testing.T) {
	var buf bytes.Buffer
	logger := New()
	logger.SetOutput(&buf)
	logger.SetColorize(false)

	logger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "INFO") {
		t.Errorf("output should contain INFO: %s", output)
	}
	if !strings.Contains(output, "test message") {
		t.Errorf("output should contain message: %s", output)
	}
}

func TestLoggerLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := New()
	logger.SetOutput(&buf)
	logger.SetLevel(LevelWarn)
	logger.SetColorize(false)

	logger.Info("info message")
	logger.Warn("warn message")

	output := buf.String()
	if strings.Contains(output, "info message") {
		t.Error("info message should be filtered")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("warn message should appear")
	}
}

func TestLoggerPrefix(t *testing.T) {
	var buf bytes.Buffer
	logger := New()
	logger.SetOutput(&buf)
	logger.SetColorize(false)

	prefixed := logger.WithPrefix("socket")
	prefixed.Info("test")

	output := buf.String()
	if !strings.Contains(output, "[socket]") {
		t.Errorf("output should contain prefix: %s", output)
	}
}

func TestLoggerFields(t *testing.T) {
	var buf bytes.Buffer
	logger := New()
	logger.SetOutput(&buf)
	logger.SetColorize(false)

	withFields := logger.WithField("key", "value")
	withFields.Info("test")

	output := buf.String()
	if !strings.Contains(output, "key=value") {
		t.Errorf("output should contain field: %s", output)
	}
}

func TestLoggerWithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := New()
	logger.SetOutput(&buf)
	logger.SetColorize(false)

	withFields := logger.WithFields(map[string]any{
		"key1": "value1",
		"key2": 42,
	})
	withFields.Info("test")

	output := buf.String()
	if !strings.Contains(output, "key1=value1") {
		t.Errorf("output should contain key1: %s", output)
	}
	if !strings.Contains(output, "key2=42") {
		t.Errorf("output should contain key2: %s", output)
	}
}

func TestLoggerFormatArgs(t *testing.T) {
	var buf bytes.Buffer
	logger := New()
	logger.SetOutput(&buf)
	logger.SetColorize(false)

	logger.Info("count: %d, name: %s", 5, "test")

	output := buf.String()
	if !strings.Contains(output, "count: 5") {
		t.Errorf("output should contain formatted count: %s", output)
	}
	if !strings.Contains(output, "name: test") {
		t.Errorf("output should contain formatted name: %s", output)
	}
}

func TestDefaultLogger(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	SetLevel(LevelDebug)

	Debug("debug msg")
	Info("info msg")
	Warn("warn msg")
	Error("error msg")

	output := buf.String()
	for _, msg := range []string{"debug msg", "info msg", "warn msg", "error msg"} {
		if !strings.Contains(output, msg) {
			t.Errorf("output should contain %s: %s", msg, output)
		}
	}
}

func TestWithPrefixDefault(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)

	logger := WithPrefix("test")
	logger.SetColorize(false)
	logger.Info("message")

	output := buf.String()
	if !strings.Contains(output, "[test]") {
		t.Errorf("output should contain prefix: %s", output)
	}
}

func TestWithFieldDefault(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)

	logger := WithField("key", "val")
	logger.SetColorize(false)
	logger.Info("message")

	output := buf.String()
	if !strings.Contains(output, "key=val") {
		t.Errorf("output should contain field: %s", output)
	}
}
