package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dimkarp93/install-libs/shellcomplete"
)

func TestCompletionDelegatesAfterDash(t *testing.T) {
	got := completionSpec.Complete([]string{"--secrets=a:B", "--", "git", "ch"})

	assert.Equal(t, []string{shellcomplete.DirCommand, "2"}, got)
}

func TestCompletionDelegatesAfterColonSplitWords(t *testing.T) {
	words := []string{"--secrets", "=", "a", ":", "B", "--", "gi"}

	got := completionSpec.Complete(words)

	assert.Equal(t, []string{shellcomplete.DirCommand, "6"}, got)
}

func TestCompletionKeepsSubcommands(t *testing.T) {
	got := completionSpec.Complete([]string{"conf"})

	assert.Equal(t, []string{"config"}, got)
}
