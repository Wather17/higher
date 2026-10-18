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

func TestVersionCommandDisplaysDevelopmentDefaults(t *testing.T) {
	previousVersion, previousCommit, previousBuildDate := Version, Commit, BuildDate
	Version, Commit, BuildDate = "dev", "none", "unknown"
	t.Cleanup(func() {
		Version, Commit, BuildDate = previousVersion, previousCommit, previousBuildDate
	})

	output, err := executeRoot(NewRootCommand(), "version")
	if err != nil {
		t.Fatalf("executing version command: %v", err)
	}
	if output != "Higher dev (commit none, build unknown)\n" {
		t.Fatalf("unexpected version output: %q", output)
	}
}

func TestVersionCommandDisplaysBuildMetadata(t *testing.T) {
	previousVersion, previousCommit, previousBuildDate := Version, Commit, BuildDate
	Version, Commit, BuildDate = "1.2.3", "abc1234", "2026-09-22T12:34:56Z"
	t.Cleanup(func() {
		Version, Commit, BuildDate = previousVersion, previousCommit, previousBuildDate
	})

	output, err := executeRoot(NewRootCommand(), "version")
	if err != nil {
		t.Fatalf("executing version command: %v", err)
	}
	if output != "Higher v1.2.3 (commit abc1234, build 2026-09-22T12:34:56Z)\n" {
		t.Fatalf("unexpected version output: %q", output)
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
