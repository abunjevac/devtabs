package editor

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// entryRow is a full-width text row for values that have no default.
func (e *editor) entryRow(field, title, value string, set func(string)) *adw.EntryRow {
	row := adw.NewEntryRow()

	row.SetTitle(title)
	row.SetText(value)

	row.ConnectChanged(func() {
		row.RemoveCSSClass("error")

		set(row.Text())

		e.markDirty()
	})

	e.fields[field] = row

	return row
}

// textRow is a labelled row with a compact entry whose placeholder shows the default.
func (e *editor) textRow(field, title, subtitle, placeholder, value string, set func(string)) *adw.ActionRow {
	return e.parsedRow(field, title, subtitle, placeholder, value, func(s string) error {
		set(s)

		return nil
	})
}

// parsedRow is like textRow, but only commits input that parse accepts.
// Rejected input stays visible, is flagged as an error, and blocks saving.
func (e *editor) parsedRow(field, title, subtitle, placeholder, value string, parse func(string) error) *adw.ActionRow {
	key := e.fieldKey(field)
	entry := gtk.NewEntry()

	entry.SetPlaceholderText(placeholder)
	entry.SetVAlign(gtk.AlignCenter)
	entry.SetWidthChars(22)

	if raw, ok := e.invalid[key]; ok {
		value = raw.text

		entry.AddCSSClass("error")
	}

	entry.SetText(value)

	entry.ConnectChanged(func() {
		text := strings.TrimSpace(entry.Text())

		e.markDirty()

		if err := parse(text); err != nil {
			entry.AddCSSClass("error")

			e.invalid[key] = invalidInput{tab: e.current, title: title, text: entry.Text()}

			return
		}

		entry.RemoveCSSClass("error")

		delete(e.invalid, key)
	})

	row := adw.NewActionRow()

	row.SetTitle(title)
	row.SetSubtitle(subtitle)
	row.AddSuffix(entry)
	row.SetActivatableWidget(entry)

	e.fields[field] = entry

	return row
}

func (e *editor) intRow(field, title, subtitle, placeholder string, value int, set func(int)) *adw.ActionRow {
	return e.parsedRow(field, title, subtitle, placeholder, formatInt(value), func(s string) error {
		if s == "" {
			set(0)

			return nil
		}

		n, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("parse-int: %w", err)
		}

		set(n)

		return nil
	})
}

func (e *editor) switchRow(title, subtitle string, value bool, set func(bool)) *adw.SwitchRow {
	row := adw.NewSwitchRow()

	row.SetTitle(title)
	row.SetSubtitle(subtitle)
	row.SetActive(value)

	row.NotifyProperty("active", func() {
		set(row.Active())

		e.markDirty()
	})

	return row
}

func group(title, description string) *adw.PreferencesGroup {
	g := adw.NewPreferencesGroup()

	g.SetTitle(title)
	g.SetDescription(description)

	return g
}

func formatInt(n int) string {
	if n == 0 {
		return ""
	}

	return strconv.Itoa(n)
}

func formatList(items []string) string {
	return strings.Join(items, ", ")
}

func parseList(s string) []string {
	var out []string

	for item := range strings.SplitSeq(s, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}

	return out
}
