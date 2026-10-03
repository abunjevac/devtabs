package editor

import (
	"testing"

	"github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/stretchr/testify/assert"
)

func TestDragIndexRoundTrip(t *testing.T) {
	i, ok := dragIndex(glib.NewValue(2))

	assert.True(t, ok)
	assert.Equal(t, 2, i)
}
