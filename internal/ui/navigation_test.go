package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdjacentTabIndex(t *testing.T) {
	tests := []struct {
		name  string
		index int
		delta int
		count int
		wrap  bool
		want  int
	}{
		{name: "moves right", index: 1, delta: 1, count: 3, want: 2},
		{name: "moves left", index: 1, delta: -1, count: 3, want: 0},
		{name: "stays at first tab without wrapping", index: 0, delta: -1, count: 3, want: 0},
		{name: "stays at last tab without wrapping", index: 2, delta: 1, count: 3, want: 2},
		{name: "wraps left from first tab", index: 0, delta: -1, count: 3, wrap: true, want: 2},
		{name: "wraps right from last tab", index: 2, delta: 1, count: 3, wrap: true, want: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, adjacentTabIndex(tc.index, tc.delta, tc.count, tc.wrap))
		})
	}
}
