package editor

import (
	"context"
	"slices"

	"github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"

	"github.com/abunjevac/devtabs/internal/config"
)

// sidebarRow holds the labels of a tab entry so they can follow edits.
type sidebarRow struct {
	row     *gtk.ListBoxRow
	name    *gtk.Label
	command *gtk.Label
}

func (e *editor) buildSidebar(ctx context.Context) *gtk.ListBox {
	list := gtk.NewListBox()

	list.AddCSSClass("navigation-sidebar")
	list.SetSelectionMode(gtk.SelectionBrowse)

	list.SetHeaderFunc(func(row, _ *gtk.ListBoxRow) {
		if row.Index() != 1 {
			row.SetHeader(nil)

			return
		}

		label := gtk.NewLabel("Tabs")

		label.AddCSSClass("heading")
		label.AddCSSClass("dim-label")
		label.SetXAlign(0)
		label.SetMarginStart(12)
		label.SetMarginTop(12)
		label.SetMarginBottom(6)

		row.SetHeader(label)
	})

	list.ConnectRowSelected(func(row *gtk.ListBoxRow) {
		if row == nil {
			return
		}

		i := row.Index()

		if i == 0 {
			e.showPage(ctx, nil)
		} else if i-1 < len(e.tabs) {
			e.showPage(ctx, e.tabs[i-1])
		}

		e.split.SetShowContent(true)
	})

	return list
}

// rebuildSidebar recreates all rows and selects the given tab, or General when nil.
func (e *editor) rebuildSidebar(selected *config.TabConfig) {
	e.sidebar.RemoveAll()

	clear(e.rows)

	e.sidebar.Append(generalRow())

	for i, t := range e.tabs {
		e.sidebar.Append(e.tabRow(i, t))
	}

	index := 0

	if i := slices.Index(e.tabs, selected); i >= 0 {
		index = i + 1
	}

	e.sidebar.SelectRow(e.sidebar.RowAtIndex(index))
}

func generalRow() *gtk.ListBoxRow {
	icon := gtk.NewImageFromIconName("preferences-system-symbolic")
	label := gtk.NewLabel("General")

	label.SetXAlign(0)

	box := gtk.NewBox(gtk.OrientationHorizontal, 12)

	box.SetMarginTop(6)
	box.SetMarginBottom(6)
	box.Append(icon)
	box.Append(label)

	row := gtk.NewListBoxRow()

	row.SetChild(box)

	return row
}

func (e *editor) tabRow(index int, t *config.TabConfig) *gtk.ListBoxRow {
	name := gtk.NewLabel("")

	name.SetXAlign(0)
	name.SetEllipsize(pango.EllipsizeEnd)

	command := gtk.NewLabel("")

	command.SetXAlign(0)
	command.SetEllipsize(pango.EllipsizeEnd)
	command.AddCSSClass("caption")
	command.AddCSSClass("dim-label")

	labels := gtk.NewBox(gtk.OrientationVertical, 2)

	labels.SetHExpand(true)
	labels.Append(name)
	labels.Append(command)

	// Adwaita and Breeze name the drag handle differently
	handle := gtk.NewImageFromGIcon(gio.NewThemedIconFromNames([]string{"list-drag-handle-symbolic", "drag-handle-symbolic"}))

	handle.AddCSSClass("dim-label")
	handle.SetTooltipText("Drag to reorder")

	box := gtk.NewBox(gtk.OrientationHorizontal, 12)

	box.SetMarginTop(6)
	box.SetMarginBottom(6)
	box.Append(labels)
	box.Append(handle)

	row := gtk.NewListBoxRow()

	row.SetChild(box)

	e.addDragAndDrop(row, index)

	e.rows[t] = &sidebarRow{row: row, name: name, command: command}

	e.refreshSidebarRow(t)

	return row
}

func (e *editor) addDragAndDrop(row *gtk.ListBoxRow, index int) {
	source := gtk.NewDragSource()

	source.SetActions(gdk.ActionMove)

	source.ConnectPrepare(func(_, _ float64) *gdk.ContentProvider {
		return gdk.NewContentProviderForValue(glib.NewValue(index))
	})

	source.ConnectDragBegin(func(_ gdk.Dragger) {
		source.SetIcon(gtk.NewWidgetPaintable(row), 0, 0)
	})

	// the drag carries the source tab index
	target := gtk.NewDropTarget(glib.NewValue(0).Type(), gdk.ActionMove)

	target.ConnectDrop(func(value *glib.Value, _, _ float64) bool {
		from, ok := dragIndex(value)
		if !ok {
			return false
		}

		e.moveTab(from, index)

		return true
	})

	row.AddController(source)
	row.AddController(target)
}

func (e *editor) refreshSidebarRow(t *config.TabConfig) {
	r, ok := e.rows[t]
	if !ok {
		return
	}

	if t.Name == "" {
		r.name.SetText("Unnamed")
		r.name.AddCSSClass("dim-label")
	} else {
		r.name.SetText(t.Name)
		r.name.RemoveCSSClass("dim-label")
	}

	r.command.SetText(t.Command)
	r.command.SetVisible(t.Command != "")
}

// dragIndex reads the tab index carried by a sidebar drag.
// gotk4 stores a Go int as G_TYPE_INT64, which reads back as int64.
func dragIndex(value *glib.Value) (int, bool) {
	i, ok := value.GoValue().(int64)

	return int(i), ok
}
