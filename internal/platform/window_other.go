//go:build !windows

package platform

// BringWindowToFront 非 Windows 系统下的空实现
func BringWindowToFront(pid int, titleKeyword string) {
	// no-op
}

// CenterAndBringWindowToFront 非 Windows 系统下的空实现
func CenterAndBringWindowToFront(pid int, titleKeyword string, targetW, targetH int) {
	// no-op
}

// CalculateCenterPosition 非 Windows 系统默认居中估算
func CalculateCenterPosition(winWidth, winHeight int) (x, y int) {
	return (1920 - winWidth) / 2, (1080 - winHeight) / 2
}

// HideConsoleWindow 非 Windows 系统下的空实现
func HideConsoleWindow() {
	// no-op
}
