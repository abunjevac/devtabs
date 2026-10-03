package editor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abunjevac/devtabs/internal/config"
)

func writeFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "devtabs.yaml")

	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	return path
}

func TestLoadDocumentMissingFile(t *testing.T) {
	doc := loadDocument(filepath.Join(t.TempDir(), "devtabs.yaml"), "ignored")

	assert.Contains(t, doc.notice, "does not exist")
	require.Len(t, doc.tabs, 1)
	assert.Equal(t, "New tab", doc.tabs[0].Name)
}

func TestLoadDocumentParseError(t *testing.T) {
	doc := loadDocument(writeFile(t, "tabs: [\n"), "ignored")

	assert.Contains(t, doc.notice, "Could not parse")
	assert.Len(t, doc.tabs, 1)
}

func TestLoadDocumentKeepsRawValuesAndNotice(t *testing.T) {
	path := writeFile(t, "# hi\nfont: Mono\ntabs:\n  - name: a\n  - name: b\n    command: y\n")

	doc := loadDocument(path, "startup failed")

	assert.Equal(t, "startup failed", doc.notice)
	assert.True(t, doc.hadComments)
	assert.Equal(t, "Mono", doc.general.Font)
	assert.Nil(t, doc.general.Tabs)
	require.Len(t, doc.tabs, 2)
	assert.Empty(t, doc.tabs[0].Shell)
}

func TestDocumentConfigRoundTrip(t *testing.T) {
	doc := loadDocument(writeFile(t, "title: X\ntabs:\n  - name: a\n    command: x\n"), "")

	doc.tabs = append(doc.tabs, &config.TabConfig{Name: "b", Command: "y"})

	cfg := doc.config()

	require.NoError(t, config.Validate(cfg))
	assert.Equal(t, "X", cfg.Title)
	assert.Len(t, cfg.Tabs, 2)
}

func TestNewTabName(t *testing.T) {
	tabs := []*config.TabConfig{{Name: "New tab"}, {Name: "New tab 2"}}

	assert.Equal(t, "New tab", newTabName(nil))
	assert.Equal(t, "New tab 3", newTabName(tabs))
}

func TestParseList(t *testing.T) {
	assert.Equal(t, []string{"-l", "-i"}, parseList(" -l, -i ,, "))
	assert.Nil(t, parseList("  "))
}
