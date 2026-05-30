package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func captureOutput(f func()) string {
	r, w, _ := os.Pipe()
	stdout := os.Stdout
	os.Stdout = w

	f()
	w.Close()
	os.Stdout = stdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

//these tests are really just checking that the formatStr variable is set correctly
func TestDebugPrintBool(t *testing.T) {
	got := captureOutput(func() {
		debugPrint("bool test", true)
	})
	if !strings.Contains(got, "true") {
		t.Errorf("expected output to contain 'true', got %q", got)
	}
}

func TestDebugPrintInt(t *testing.T) {
	got := captureOutput(func() {
		debugPrint("int test", 42)
	})
	if !strings.Contains(got, "42") {
		t.Errorf("expected output to contain '42', got %q", got)
	}
}

func TestDebugPrintString(t *testing.T) {
	got := captureOutput(func() {
		debugPrint("string test", "hello")
	})
	if !strings.Contains(got, "hello") {
		t.Errorf("expected output to contain 'hello', got %q", got)
	}
}
