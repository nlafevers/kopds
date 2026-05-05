package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestPasswordFromArgsReadsStdin(t *testing.T) {
	password, err := passwordFromArgs([]string{"--password-stdin"}, strings.NewReader("secret\n"), &bytes.Buffer{})
	if err != nil {
		t.Fatalf("passwordFromArgs failed: %v", err)
	}
	if password != "secret" {
		t.Fatalf("expected trimmed password, got %q", password)
	}
}

func TestPasswordFromArgsRejectsPositionalPassword(t *testing.T) {
	_, err := passwordFromArgs([]string{"secret"}, strings.NewReader(""), &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected positional password to be rejected")
	}
}

func TestPasswordFromArgsRejectsEmptyStdinPassword(t *testing.T) {
	_, err := passwordFromArgs([]string{"--password-stdin"}, strings.NewReader("\n"), &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected empty stdin password to be rejected")
	}
}
