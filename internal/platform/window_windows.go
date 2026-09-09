//go:build windows

package platform

import (
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	procShowWindow               = user32.NewProc("ShowWindow")
	procSwitchToThisWindow       = user32.NewProc("SwitchToThisWindow")
	procGetWindowTextW           = user32.NewProc("GetWindowTextW")
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procAttachThreadInput        = user32.NewProc("AttachThreadInput")
	procBringWindowToTop         = user32.NewProc("BringWindowToTop")
	procSetWindowPos             = user32.NewProc("SetWindowPos")
	procKeybdEvent               = user32.NewProc("keybd_event")
	procGetSystemMetrics         = user32.NewProc("GetSystemMetrics")
	procGetWindowRect            = user32.NewProc("GetWindowRect")

	procGetCurrentThreadId = kernel32.NewProc("GetCurrentThreadId")
	procGetConsoleWindow   = kernel32.NewProc("GetConsoleWindow")
)

// HideConsoleWindow 隐藏当前进程可能依附的控制台黑框 (SW_HIDE = 0)
func HideConsoleWindow() {
	hwnd, _, _ := procGetConsoleWindow.Call()
	if hwnd != 0 {
		procShowWindow.Call(hwnd, 0) // SW_HIDE = 0
	}
}

type winRect struct {
	Left, Top, Right, Bottom int32
}

// GetScreenSize 获取当前主屏幕分辨率宽度与高度
func GetScreenSize() (width, height int) {
	w, _, _ := procGetSystemMetrics.Call(0) // SM_CXSCREEN
	h, _, _ := procGetSystemMetrics.Call(1) // SM_CYSCREEN
	if w <= 0 || h <= 0 {
		return 1920, 1080
	}
	return int(w), int(h)
}

// CalculateCenterPosition 计算指定宽高窗口居中放置的左上角 (x, y) 坐标
func CalculateCenterPosition(winWidth, winHeight int) (x, y int) {
	sw, sh := GetScreenSize()
	x = (sw - winWidth) / 2
	y = (sh - winHeight) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	return x, y
}

// ForceWindowToFront 强行突破 Windows 前台锁限制，将指定 HWND 窗口瞬时拉到屏幕最前面并赋予输入焦点
func ForceWindowToFront(hwnd uintptr) {
	const (
		SWP_NOSIZE      = 0x0001
		SWP_NOMOVE      = 0x0002
		SWP_SHOWWINDOW  = 0x0040
		SW_RESTORE      = 9
		VK_MENU         = 0x12
		KEYEVENTF_KEYUP = 0x0002
	)
	hwndTopmost := ^uintptr(0)    // HWND_TOPMOST (-1)
	hwndNotTopmost := ^uintptr(1) // HWND_NOTOPMOST (-2)

	// 1. 瞬时赋予 HWND_TOPMOST，直接从物理 Z-Order 上将窗口推到桌面最前方
	procShowWindow.Call(hwnd, SW_RESTORE)
	procSetWindowPos.Call(hwnd, hwndTopmost, 0, 0, 0, 0, SWP_NOMOVE|SWP_NOSIZE|SWP_SHOWWINDOW)

	// 2. 模拟 Alt 键和线程输入关联，突破 Windows 11 前台锁保护
	foreWnd, _, _ := procGetForegroundWindow.Call()
	var forePid uint32
	foreThread, _, _ := procGetWindowThreadProcessId.Call(foreWnd, uintptr(unsafe.Pointer(&forePid)))
	targetThread, _, _ := procGetWindowThreadProcessId.Call(hwnd, 0)
	curThread, _, _ := procGetCurrentThreadId.Call()

	// 按下 Alt 键欺骗系统获得焦点转移特权
	procKeybdEvent.Call(VK_MENU, 0, 0, 0)

	if foreThread != 0 && curThread != 0 {
		procAttachThreadInput.Call(curThread, foreThread, 1)
	}
	if targetThread != 0 && curThread != 0 {
		procAttachThreadInput.Call(curThread, targetThread, 1)
	}

	procBringWindowToTop.Call(hwnd)
	procSetForegroundWindow.Call(hwnd)
	procSwitchToThisWindow.Call(hwnd, 1)

	if foreThread != 0 && curThread != 0 {
		procAttachThreadInput.Call(curThread, foreThread, 0)
	}
	if targetThread != 0 && curThread != 0 {
		procAttachThreadInput.Call(curThread, targetThread, 0)
	}

	procKeybdEvent.Call(VK_MENU, 0, KEYEVENTF_KEYUP, 0)

	// 3. 350ms 后降级为普通非置顶 (HWND_NOTOPMOST)，确保窗口已在前台但不会遮挡用户后续操作
	time.AfterFunc(350*time.Millisecond, func() {
		procSetWindowPos.Call(hwnd, hwndNotTopmost, 0, 0, 0, 0, SWP_NOMOVE|SWP_NOSIZE|SWP_SHOWWINDOW)
	})
}

// CenterWindow 将窗口物理移动到主屏幕居中位置，优先保持窗口自身实际大小（避免高 DPI 缩放下被压小）
func CenterWindow(hwnd uintptr, targetW, targetH int) {
	sw, sh := GetScreenSize()

	var rect winRect
	ret, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))

	const (
		SWP_NOSIZE     = 0x0001
		SWP_NOZORDER   = 0x0004
		SWP_SHOWWINDOW = 0x0040
		SW_RESTORE     = 9
	)

	procShowWindow.Call(hwnd, SW_RESTORE)

	if ret != 0 {
		w := int(rect.Right - rect.Left)
		h := int(rect.Bottom - rect.Top)
		// 如果窗口已经具备正常尺寸，保持当前尺寸居中（SWP_NOSIZE），杜绝高 DPI 下窗口被强行压小
		if w > 300 && h > 200 {
			posX := (sw - w) / 2
			posY := (sh - h) / 2
			if posX < 0 {
				posX = 0
			}
			if posY < 0 {
				posY = 0
			}
			procSetWindowPos.Call(hwnd, 0, uintptr(posX), uintptr(posY), 0, 0, SWP_NOSIZE|SWP_NOZORDER|SWP_SHOWWINDOW)
			return
		}
	}

	// 降级兜底：如果尚未取得窗口实际尺寸，使用默认目标尺寸
	posX := (sw - targetW) / 2
	posY := (sh - targetH) / 2
	if posX < 0 {
		posX = 0
	}
	if posY < 0 {
		posY = 0
	}
	procSetWindowPos.Call(hwnd, 0, uintptr(posX), uintptr(posY), uintptr(targetW), uintptr(targetH), SWP_NOZORDER|SWP_SHOWWINDOW)
}

// CenterAndBringWindowToFront 寻找指定 PID 或包含指定关键字的窗口，将其物理居中并赋予焦点
func CenterAndBringWindowToFront(pid int, titleKeyword string, targetW, targetH int) {
	go func() {
		// 在 4.5 秒内每隔 150ms 进行轮询检测，直到窗口创建并完成居中弹出
		matchedCount := 0
		for i := 0; i < 30; i++ {
			time.Sleep(150 * time.Millisecond)

			cb := syscall.NewCallback(func(hwnd syscall.Handle, lParam uintptr) uintptr {
				vis, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
				if vis == 0 {
					return 1
				}

				// 读取标题
				buf := make([]uint16, 256)
				ret, _, _ := procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), 256)
				title := ""
				if ret > 0 {
					title = syscall.UTF16ToString(buf)
				}

				var wPid uint32
				procGetWindowThreadProcessId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&wPid)))

				matched := false
				if pid > 0 && int(wPid) == pid {
					matched = true
				}

				if !matched && titleKeyword != "" && strings.Contains(title, titleKeyword) {
					matched = true
				}

				if matched {
					CenterWindow(uintptr(hwnd), targetW, targetH)
					ForceWindowToFront(uintptr(hwnd))
					matchedCount++
					if matchedCount >= 2 {
						return 0 // 成功居中弹出 2 次后退出
					}
				}
				return 1
			})

			procEnumWindows.Call(cb, 0)
			if matchedCount >= 2 {
				break
			}
		}
	}()
}

// BringWindowToFront 寻找指定 PID 或包含指定关键字的窗口并将其强行弹出至最前并居中
func BringWindowToFront(pid int, titleKeyword string) {
	CenterAndBringWindowToFront(pid, titleKeyword, 960, 720)
}
