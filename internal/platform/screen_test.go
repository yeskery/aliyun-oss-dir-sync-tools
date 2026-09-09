package platform

import (
	"testing"
)

func TestGetScreenSize(t *testing.T) {
	w, h := GetScreenSize()
	t.Logf("GetScreenSize returned: %d x %d", w, h)
	x, y := CalculateCenterPosition(1240, 840)
	t.Logf("CalculateCenterPosition(1240, 840) = (%d, %d)", x, y)
}
