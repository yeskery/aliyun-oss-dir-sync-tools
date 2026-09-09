package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// OpenDesktopWindow 根据不同操作系统拉起独立无边框应用窗口（若无可用独立窗口则降级打开系统默认浏览器）
func OpenDesktopWindow(url string) (*exec.Cmd, error) {
	switch runtime.GOOS {
	case "windows":
		return openWindowsApp(url)
	case "darwin":
		return openDarwinApp(url)
	default:
		return openLinuxApp(url)
	}
}

// FindBrowserPath 返回当前系统上可用的 Chrome / Edge 浏览器可执行文件路径
func FindBrowserPath() string {
	switch runtime.GOOS {
	case "windows":
		candidates := []string{
			`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
			`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
			filepath.Join(os.Getenv("LOCALAPPDATA"), `Microsoft\Edge\Application\msedge.exe`),
			`C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
			filepath.Join(os.Getenv("LOCALAPPDATA"), `Google\Chrome\Application\chrome.exe`),
		}
		for _, path := range candidates {
			if path != "" {
				if _, err := os.Stat(path); err == nil {
					return path
				}
			}
		}
	case "darwin":
		candidates := []string{
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		}
		for _, path := range candidates {
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	default:
		for _, name := range []string{"microsoft-edge", "google-chrome", "chromium", "chromium-browser"} {
			if path, err := exec.LookPath(name); err == nil {
				return path
			}
		}
	}
	return ""
}

// PrepareCleanProfile 初始化一个纯净且彻底禁用微软账户登录/同步弹窗的浏览器 Profile
func PrepareCleanProfile(profileDir string) {
	defaultDir := filepath.Join(profileDir, "Default")
	_ = os.MkdirAll(defaultDir, 0755)

	// 1. 注入 Default/Preferences 禁用 signin 与 sync 并清空历史缓存窗口位置
	prefFile := filepath.Join(defaultDir, "Preferences")
	prefContent := `{
  "browser": {
    "app_window_placement": {},
    "window_placement": {}
  },
  "signin": {"allowed": false},
  "sync": {
    "suppress_start": true,
    "has_setup_completed": true
  },
  "edge": {
    "fre": {"has_user_seen_fre": true},
    "sync": {
      "has_setup_completed": true,
      "user_choice_made": true
    },
    "profile": {
      "show_sync_confirmation": false
    },
    "profile_sync": {
      "suppress_fre": true
    }
  },
  "profile": {
    "exit_type": "Normal",
    "password_manager_enabled": false
  }
}`
	_ = os.WriteFile(prefFile, []byte(prefContent), 0644)

	// 2. 注入 Local State 标记已跳过首次欢迎流与个人配置同步向导
	localStateFile := filepath.Join(profileDir, "Local State")
	localStateContent := `{
  "browser": {
    "has_seen_welcome_page": true,
    "app_window_placement": {},
    "window_placement": {}
  },
  "fre": {"has_user_seen_fre": true},
  "edge": {
    "fre": {"has_user_seen_fre": true}
  }
}`
	_ = os.WriteFile(localStateFile, []byte(localStateContent), 0644)
}

// openWindowsApp Windows 下优先使用 Edge / Chrome App 独立窗口模式
func openWindowsApp(url string) (*exec.Cmd, error) {
	// 常见 Edge / Chrome 安装路径
	candidates := []string{
		`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
		`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
		filepath.Join(os.Getenv("LOCALAPPDATA"), `Microsoft\Edge\Application\msedge.exe`),
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		filepath.Join(os.Getenv("LOCALAPPDATA"), `Google\Chrome\Application\chrome.exe`),
	}

	tempDir := filepath.Join(os.TempDir(), "aliyun_oss_sync_app_profile")
	PrepareCleanProfile(tempDir)
	posX, posY := CalculateCenterPosition(1240, 840)

	for _, path := range candidates {
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			cmd := exec.Command(path,
				fmt.Sprintf("--app=%s", url),
				fmt.Sprintf("--user-data-dir=%s", tempDir),
				"--no-first-run",
				"--no-default-browser-check",
				"--disable-sync",
				"--disable-features=msImplicitSignin,msSignInPromptOnLaunch,msEdgeShowFeatureWelcomePage,msFreDialogFlow,msHub,msSyncAndServicesDialog,msFirstRunDefault,msEdgeIdentityProfileSwitcher,msSingleSignOn",
				fmt.Sprintf("--window-position=%d,%d", posX, posY),
				"--window-size=1240,840",
			)
			if err := cmd.Start(); err == nil {
				CenterAndBringWindowToFront(cmd.Process.Pid, "双 Bucket", 1240, 840)
				return cmd, nil
			}
		}
	}

	// 降级：调用系统默认关联打开
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	return cmd, cmd.Start()
}

// openDarwinApp macOS 下优先检测 Chrome --app 模式，否则调用 open
func openDarwinApp(url string) (*exec.Cmd, error) {
	chromePath := "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
	if _, err := os.Stat(chromePath); err == nil {
		tempDir := filepath.Join(os.TempDir(), "aliyun_oss_sync_app_profile")
		PrepareCleanProfile(tempDir)
		posX, posY := CalculateCenterPosition(1240, 840)
		cmd := exec.Command(chromePath,
			fmt.Sprintf("--app=%s", url),
			fmt.Sprintf("--user-data-dir=%s", tempDir),
			"--no-first-run",
			"--no-default-browser-check",
			"--disable-sync",
			fmt.Sprintf("--window-position=%d,%d", posX, posY),
			"--window-size=1240,840",
		)
		if err := cmd.Start(); err == nil {
			return cmd, nil
		}
	}

	cmd := exec.Command("open", url)
	return cmd, cmd.Start()
}

// openLinuxApp Linux 下优先检测 Chromium/Chrome，否则调用 xdg-open
func openLinuxApp(url string) (*exec.Cmd, error) {
	for _, name := range []string{"google-chrome", "chromium", "chromium-browser"} {
		if path, err := exec.LookPath(name); err == nil {
			tempDir := filepath.Join(os.TempDir(), "aliyun_oss_sync_app_profile")
			PrepareCleanProfile(tempDir)
			posX, posY := CalculateCenterPosition(1240, 840)
			cmd := exec.Command(path,
				fmt.Sprintf("--app=%s", url),
				fmt.Sprintf("--user-data-dir=%s", tempDir),
				"--no-first-run",
				"--no-default-browser-check",
				"--disable-sync",
				fmt.Sprintf("--window-position=%d,%d", posX, posY),
				"--window-size=1240,840",
			)
			if err := cmd.Start(); err == nil {
				return cmd, nil
			}
		}
	}

	cmd := exec.Command("xdg-open", url)
	return cmd, cmd.Start()
}
