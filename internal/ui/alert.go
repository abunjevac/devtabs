package ui

/*
#cgo pkg-config: gtk4
#include <stdlib.h>
#include <gtk/gtk.h>

static GtkAlertDialog *new_alert_dialog(const char *message) {
	return gtk_alert_dialog_new("%s", message);
}
*/
import "C"

import (
	"unsafe"

	"github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func newAlertDialog(message string) *gtk.AlertDialog {
	cMessage := C.CString(message)

	defer freeCString(cMessage)

	dialog := C.new_alert_dialog(cMessage)
	obj := glib.AssumeOwnership(unsafe.Pointer(dialog)).Cast()
	alert, ok := obj.(*gtk.AlertDialog)

	if !ok {
		panic("GtkAlertDialog cast failed")
	}

	return alert
}

func freeCString(str *C.char) {
	C.free(unsafe.Pointer(str))
}
