package ui

func adjacentTabIndex(index, delta, count int, wrap bool) int {
	if count <= 0 {
		return -1
	}

	next := index + delta

	if next >= 0 && next < count {
		return next
	}

	if !wrap {
		return index
	}

	if next < 0 {
		return count - 1
	}

	return 0
}
