package logging

import (
	"bytes"
	"strings"
	"testing"
)

func TestFormatWithSection(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, DEBUG)
	logger.Info("wifi", "configuring SSID %q", "mynetwork")

	output := buf.String()
	if !strings.Contains(output, "bootconf:") {
		t.Error("output missing 'bootconf:' prefix")
	}
	if !strings.Contains(output, "INFO") {
		t.Error("output missing 'INFO' level")
	}
	if !strings.Contains(output, "section=wifi") {
		t.Error("output missing 'section=wifi' tag")
	}
	if !strings.Contains(output, `configuring SSID "mynetwork"`) {
		t.Error("output missing formatted message")
	}
}

func TestLogLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, WARN)

	logger.Debug("test", "debug message")
	logger.Info("test", "info message")
	logger.Warn("test", "warn message")

	output := buf.String()
	if strings.Contains(output, "debug message") {
		t.Error("DEBUG should be suppressed at WARN level")
	}
	if strings.Contains(output, "info message") {
		t.Error("INFO should be suppressed at WARN level")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("WARN should pass at WARN level")
	}
}

func TestMultipleSections(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, INFO)

	logger.Info("wifi", "wifi message")
	logger.Info("ssh", "ssh message")
	logger.Info("system", "system message")

	output := buf.String()
	if !strings.Contains(output, "section=wifi") {
		t.Error("output missing section=wifi")
	}
	if !strings.Contains(output, "section=ssh") {
		t.Error("output missing section=ssh")
	}
	if !strings.Contains(output, "section=system") {
		t.Error("output missing section=system")
	}
}

func TestLogLevelString(t *testing.T) {
	cases := []struct {
		level LogLevel
		want  string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
		{FATAL, "FATAL"},
	}
	for _, testCase := range cases {
		if got := testCase.level.String(); got != testCase.want {
			t.Errorf("LogLevel(%d).String() = %q, want %q", testCase.level, got, testCase.want)
		}
	}
}

func TestSetLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, INFO)

	logger.Debug("test", "before")
	logger.SetLevel(DEBUG)
	logger.Debug("test", "after")

	output := buf.String()
	if strings.Contains(output, "before") {
		t.Error("DEBUG should be suppressed before SetLevel")
	}
	if !strings.Contains(output, "after") {
		t.Error("DEBUG should pass after SetLevel to DEBUG")
	}
}
