package logging

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func TestInfoAndWarnWriteToInfoWriter(t *testing.T) {
	var infoBuf bytes.Buffer
	var errBuf bytes.Buffer
	SetWriters(&infoBuf, &errBuf)
	defer ResetWriters()

	Infof("server starting on %s", ":5000")
	Warnln("config fallback enabled")

	infoOut := infoBuf.String()
	if !strings.Contains(infoOut, "[INFO] server starting on :5000") {
		t.Fatalf("expected info output, got %q", infoOut)
	}
	if !strings.Contains(infoOut, "[WARN] config fallback enabled") {
		t.Fatalf("expected warn output in info writer, got %q", infoOut)
	}
	if errBuf.Len() != 0 {
		t.Fatalf("expected empty error writer, got %q", errBuf.String())
	}
}

func TestErrorWritesToErrorWriter(t *testing.T) {
	var infoBuf bytes.Buffer
	var errBuf bytes.Buffer
	SetWriters(&infoBuf, &errBuf)
	defer ResetWriters()

	Errorf("failed to start server: %v", "boom")

	if infoBuf.Len() != 0 {
		t.Fatalf("expected empty info writer, got %q", infoBuf.String())
	}
	if !strings.Contains(errBuf.String(), "[ERROR] failed to start server: boom") {
		t.Fatalf("expected error output, got %q", errBuf.String())
	}
}

func TestConfigureStdlibRoutesWarningsAndInfo(t *testing.T) {
	var infoBuf bytes.Buffer
	var errBuf bytes.Buffer
	SetWriters(&infoBuf, &errBuf)
	defer func() {
		ResetWriters()
		ConfigureStdlib()
	}()

	ConfigureStdlib()
	log.Printf("Server starting on %s", ":5000")
	log.Printf("Warning: failed to load config: %v", "missing")

	if !strings.Contains(infoBuf.String(), "Server starting on :5000") {
		t.Fatalf("expected stdlib info output, got %q", infoBuf.String())
	}
	if !strings.Contains(errBuf.String(), "Warning: failed to load config: missing") {
		t.Fatalf("expected stdlib warning output, got %q", errBuf.String())
	}
}
