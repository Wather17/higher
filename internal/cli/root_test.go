package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewRootCommandShowsHelpWithoutArguments(t *testing.T) {
	command := NewRootCommand()
	output := new(bytes.Buffer)
	command.SetOut(output)
	command.SetErr(output)

	if err := command.Execute(); err != nil {
		t.Fatalf("executing root command: %v", err)
	}

	if !strings.Contains(output.String(), "Usage:") {
		t.Fatalf("expected help output, got %q", output.String())
	}
}

func TestNewRootCommandRejectsUnknownCommand(t *testing.T) {
	command := NewRootCommand()
	command.SetArgs([]string{"unknown"})

	if err := command.Execute(); err == nil {
		t.Fatal("expected unknown command to fail")
	}
}
