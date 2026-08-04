package testutils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-openapi/testify/v2/require"
)

// StripANSI removes the SGR escape sequences lipgloss emits, so assertions can look at the text a user reads rather
// than the styling around it.
func StripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}

	return b.String()
}

func KeyRune(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

// WriteTempGo puts a Go file on disk and returns its path.
func WriteTempGo(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "x.go")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}
