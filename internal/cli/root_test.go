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

func TestNewRootCommandIncludesReserveCommand(t *testing.T) {
	command := NewRootCommand()
	reserveCommand, _, err := command.Find([]string{"reserve"})
	if err != nil {
		t.Fatalf("finding reserve command: %v", err)
	}
	if reserveCommand == nil || reserveCommand.Use != "reserve" {
		t.Fatalf("unexpected reserve command: %+v", reserveCommand)
	}
}
