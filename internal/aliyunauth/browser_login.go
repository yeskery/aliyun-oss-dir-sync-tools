package aliyunauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"aliyun_oss_dir_sync-tools/internal/platform"
	"github.com/gorilla/websocket"
)

// LoginStatus 登录流程状态
type LoginStatus string

const (
	StatusPending LoginStatus = "pending" // 等待用户在浏览器中完成登录
	StatusSuccess LoginStatus = "success" // 登录成功并已获取 STS 凭据
	StatusFailed  LoginStatus = "failed"  // 凭据置换或解密失败
	StatusClosed  LoginStatus = "closed"  // 登录窗口被用户主动关闭或超时
)

// LoginSessionInfo 返回给前端的状态快照
type LoginSessionInfo struct {
	Target      string          `json:"target"`
	Status      LoginStatus     `json:"status"`
	Message     string          `json:"message"`
	Credentials *STSCredentials `json:"credentials,omitempty"`
}

type cdpTarget struct {
	ID                   string `json:"id"`
	Type                 string `json:"type"`
	URL                  string `json:"url"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

type cdpCookie struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Domain string `json:"domain"`
	Path   string `json:"path"`
}

type cdpResponse struct {
	ID     int `json:"id"`
	Result struct {
		Cookies []cdpCookie `json:"cookies"`
	} `json:"result"`
	Error interface{} `json:"error"`
}

type activeSession struct {
	target      string
	status      LoginStatus
	message     string
	credentials *STSCredentials
	cmd         *exec.Cmd
	cancel      context.CancelFunc
	updatedAt   time.Time
}

// BrowserLoginManager 管理登录流程
type BrowserLoginManager struct {
	mu       sync.Mutex
	sessions map[string]*activeSession
}

var globalLoginManager = &BrowserLoginManager{
	sessions: make(map[string]*activeSession),
}

// GetLoginManager 获取全局单例管理器
func GetLoginManager() *BrowserLoginManager {
	return globalLoginManager
}

// StartLogin 启动对应 Target 的账号登录窗口并开始 CDP 监听
func (m *BrowserLoginManager) StartLogin(target string) (*LoginSessionInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 若已有正在进行的会话，先强行关闭它
	if old, exists := m.sessions[target]; exists {
		if old.cancel != nil {
			old.cancel()
		}
		if old.cmd != nil && old.cmd.Process != nil {
			_ = old.cmd.Process.Kill()
		}
		delete(m.sessions, target)
	}

	browserPath := platform.FindBrowserPath()
	if browserPath == "" {
		return nil, fmt.Errorf("系统未检测到 Google Chrome 或 Microsoft Edge，请确保已安装任一浏览器")
	}

	freePort, err := getFreeTCPPort()
	if err != nil {
		return nil, fmt.Errorf("无法分配调试端口: %w", err)
	}

	// 准备独立的登录用浏览器 Profile 目录并注入纯净配置（彻底禁用微软账户同步与弹窗）
	profileDir := filepath.Join(os.TempDir(), fmt.Sprintf("aliyun_oss_login_%s_%d", target, time.Now().Unix()))
	platform.PrepareCleanProfile(profileDir)

	loginURL := "https://account.aliyun.com/login/login.htm"
	loginPosX, loginPosY := platform.CalculateCenterPosition(960, 720)
	args := []string{
		fmt.Sprintf("--app=%s", loginURL),
		"--new-window",
		fmt.Sprintf("--remote-debugging-port=%d", freePort),
		fmt.Sprintf("--user-data-dir=%s", profileDir),
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-sync",
		"--disable-features=msImplicitSignin,msSignInPromptOnLaunch,msEdgeShowFeatureWelcomePage,msFreDialogFlow,msHub,msSyncAndServicesDialog,msFirstRunDefault,msEdgeIdentityProfileSwitcher,msSingleSignOn,msEdgeWamAccounts",
		fmt.Sprintf("--window-position=%d,%d", loginPosX, loginPosY),
		"--window-size=960,720",
	}

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, browserPath, args...)

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("拉起登录浏览器失败: %w", err)
	}

	// 强行将新弹出的登录窗口激活置顶，彻底避免被系统前台锁阻挡留在任务栏
	if cmd.Process != nil {
		platform.BringWindowToFront(cmd.Process.Pid, "阿里云")
	}

	sess := &activeSession{
		target:    target,
		status:    StatusPending,
		message:   "登录窗口已唤起，请在弹出窗口中完成 RAM 子账号或主账号登录",
		cmd:       cmd,
		cancel:    cancel,
		updatedAt: time.Now(),
	}
	m.sessions[target] = sess

	// 后台运行 CDP 监控与凭据置换协程
	go m.runCDPMonitor(ctx, target, freePort, cmd, profileDir)

	return &LoginSessionInfo{
		Target:  target,
		Status:  StatusPending,
		Message: sess.message,
	}, nil
}

// GetStatus 查询当前 target 的登录状态
func (m *BrowserLoginManager) GetStatus(target string) LoginSessionInfo {
	m.mu.Lock()
	defer m.mu.Unlock()

	sess, exists := m.sessions[target]
	if !exists {
		return LoginSessionInfo{
			Target:  target,
			Status:  StatusClosed,
			Message: "未发起登录",
		}
	}

	return LoginSessionInfo{
		Target:      sess.target,
		Status:      sess.status,
		Message:     sess.message,
		Credentials: sess.credentials,
	}
}

// CancelLogin 主动取消/关闭登录窗口
func (m *BrowserLoginManager) CancelLogin(target string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sess, exists := m.sessions[target]; exists {
		if sess.cancel != nil {
			sess.cancel()
		}
		if sess.cmd != nil && sess.cmd.Process != nil {
			_ = sess.cmd.Process.Kill()
		}
		sess.status = StatusClosed
		sess.message = "用户取消了登录"
	}
}

// runCDPMonitor 负责轮询 CDP 提取 Cookie 并在捕获后完成换取凭据
// runCDPMonitor 连接浏览器根调试通道（Browser target），此通道在任何跨域跳转、页面重定向时均永不断开
func (m *BrowserLoginManager) runCDPMonitor(ctx context.Context, target string, port int, cmd *exec.Cmd, profileDir string) {
	defer func() {
		time.Sleep(2 * time.Second)
		_ = os.RemoveAll(profileDir)
	}()

	cmdDone := make(chan error, 1)
	go func() {
		cmdDone <- cmd.Wait()
	}()

	// 1. 等待浏览器根调试端口就绪并获取 Browser target URL
	var wsURL string
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return
		case <-cmdDone:
			m.updateSessionState(target, StatusClosed, "登录窗口已关闭", nil)
			return
		default:
		}

		u, err := fetchBrowserDebuggerURL(port)
		if err == nil && u != "" {
			wsURL = u
			break
		}
		time.Sleep(250 * time.Millisecond)
	}

	if wsURL == "" {
		m.updateSessionState(target, StatusFailed, "无法建立与浏览器的 DevTools 根通道连接", nil)
		return
	}

	// 2. 连接 Browser WebSocket（在任何页面跳转过程中绝不断开）
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		m.updateSessionState(target, StatusFailed, fmt.Sprintf("连接 DevTools 失败: %v", err), nil)
		return
	}
	defer conn.Close()

	// 3. 启动独立的消息监听协程
	type storageCookiesResp struct {
		ID     int `json:"id"`
		Result struct {
			Cookies []struct {
				Name   string `json:"name"`
				Value  string `json:"value"`
				Domain string `json:"domain"`
			} `json:"cookies"`
		} `json:"result"`
	}

	msgChan := make(chan []byte, 30)
	readErrChan := make(chan error, 1)
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				readErrChan <- err
				return
			}
			msgChan <- msg
		}
	}()

	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	reqSeq := 100
	for {
		select {
		case <-ctx.Done():
			return
		case <-cmdDone:
			m.updateSessionState(target, StatusClosed, "登录窗口已关闭", nil)
			return
		case <-readErrChan:
			return
		case <-ticker.C:
			reqSeq++
			_ = conn.WriteJSON(map[string]interface{}{
				"id":     reqSeq,
				"method": "Storage.getCookies",
			})
		case msg := <-msgChan:
			var resp storageCookiesResp
			if err := json.Unmarshal(msg, &resp); err == nil && resp.ID > 0 {
				cookies := resp.Result.Cookies
				var ticketVal string
				var aliyunidVal string
				var cookieParts []string

				for _, c := range cookies {
					cookieParts = append(cookieParts, fmt.Sprintf("%s=%s", c.Name, c.Value))
					if c.Name == "login_aliyunid_ticket" && strings.TrimSpace(c.Value) != "" {
						ticketVal = strings.TrimSpace(c.Value)
					}
					if (c.Name == "login_aliyunid" || c.Name == "login_aliyunid_pk") && strings.TrimSpace(c.Value) != "" {
						aliyunidVal = strings.TrimSpace(c.Value)
					}
				}

				// 检测是否已产生登录凭据或已完成登录
				if ticketVal != "" || aliyunidVal != "" {
					// ⚡ 立即关闭登录浏览器窗口，秒级反馈
					if cmd.Process != nil {
						_ = cmd.Process.Kill()
					}

					cookieStr := strings.Join(cookieParts, "; ")
					creds, err := ExchangeTicketWithCookies(cookieStr, ticketVal)
					if err == nil && creds != nil && creds.AccessKeyID != "" {
						m.updateSessionState(target, StatusSuccess, "阿里云登录成功，已解密临时访问凭证", creds)
						return
					} else {
						m.updateSessionState(target, StatusFailed, fmt.Sprintf("获取临时凭证失败: %v", err), nil)
						return
					}
				}
			}
		}
	}
}

func (m *BrowserLoginManager) updateSessionState(target string, status LoginStatus, message string, creds *STSCredentials) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sess, exists := m.sessions[target]; exists {
		// 如果已经成功，不再被 closed 覆盖
		if sess.status == StatusSuccess && status == StatusClosed {
			return
		}
		sess.status = status
		sess.message = message
		if creds != nil {
			sess.credentials = creds
		}
		sess.updatedAt = time.Now()
	}
}

func fetchBrowserDebuggerURL(port int) (string, error) {
	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/json/version", port))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var ver struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err := json.Unmarshal(body, &ver); err != nil {
		return "", err
	}
	if ver.WebSocketDebuggerURL == "" {
		return "", fmt.Errorf("webSocketDebuggerUrl 为空")
	}
	return ver.WebSocketDebuggerURL, nil
}

func getFreeTCPPort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
