package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootNoArgsPrintsHelp(t *testing.T) {
	messageFlag = ""
	linkFlag = ""
	imagePath = ""
	imageAlt = ""
	targetsFlag = nil
	dryRun = false
	verbose = false

	root := newRootCommand()
	root.SetIn(strings.NewReader(""))
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetErr(&output)
	root.SetArgs(nil)

	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Usage:") || !strings.Contains(output.String(), "xpost publishes the same update") {
		t.Fatalf("output = %q, want help", output.String())
	}
}

func TestResolveMessageReadsPipedInput(t *testing.T) {
	messageFlag = ""
	linkFlag = ""
	imagePath = ""
	imageAlt = ""
	targetsFlag = nil
	dryRun = false
	verbose = false

	root := newRootCommand()
	root.SetIn(strings.NewReader("hello from stdin\n"))
	message, err := resolveMessage(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if message != "hello from stdin" {
		t.Fatalf("message = %q, want hello from stdin", message)
	}
}
