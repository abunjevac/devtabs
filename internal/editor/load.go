package editor

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/abunjevac/devtabs/internal/config"
)

// document is the editable state read from a config file.
type document struct {
	general     config.Config
	tabs        []*config.TabConfig
	hadComments bool
	// explains why the form does not reflect a valid file, empty when it does
	notice string
}

// loadDocument reads path for editing. It never fails: a missing or unparsable
// file yields a fresh document with a notice, so the user can always start over.
func loadDocument(path, notice string) document {
	data, err := os.ReadFile(path)

	switch {
	case errors.Is(err, fs.ErrNotExist):
		return newDocument(fmt.Sprintf("%s does not exist yet. Add your tabs and save to create it.", filepath.Base(path)))
	case err != nil:
		return newDocument(fmt.Sprintf("Could not read the config: %v. Saving will overwrite it.", err))
	}

	cfg, err := config.Decode(string(data))
	if err != nil {
		return newDocument(fmt.Sprintf("Could not parse the config: %v. Saving will overwrite it.", err))
	}

	doc := document{
		general:     *cfg,
		hadComments: config.HasComments(string(data)),
		notice:      notice,
	}

	doc.general.Tabs = nil

	for i := range cfg.Tabs {
		doc.tabs = append(doc.tabs, &cfg.Tabs[i])
	}

	if len(doc.tabs) == 0 {
		doc.tabs = []*config.TabConfig{{Name: newTabName(nil)}}
	}

	return doc
}

func newDocument(notice string) document {
	return document{
		tabs:   []*config.TabConfig{{Name: newTabName(nil)}},
		notice: notice,
	}
}

// config assembles the document into a config ready to validate and save.
func (d *document) config() *config.Config {
	cfg := d.general

	cfg.Tabs = make([]config.TabConfig, 0, len(d.tabs))

	for _, t := range d.tabs {
		cfg.Tabs = append(cfg.Tabs, *t)
	}

	return &cfg
}

// newTabName returns "New tab", or "New tab N" when that name is taken.
func newTabName(tabs []*config.TabConfig) string {
	taken := make(map[string]struct{}, len(tabs))

	for _, t := range tabs {
		taken[t.Name] = struct{}{}
	}

	name := "New tab"

	for n := 2; ; n++ {
		if _, ok := taken[name]; !ok {
			return name
		}

		name = fmt.Sprintf("New tab %d", n)
	}
}
