package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"aliyun_oss_dir_sync-tools/internal/config"
	"aliyun_oss_dir_sync-tools/internal/manager"
	"aliyun_oss_dir_sync-tools/internal/platform"
	"aliyun_oss_dir_sync-tools/internal/server"
	"aliyun_oss_dir_sync-tools/ui"
)

func main() {
	// 启动阶段立即隐藏控制台窗口（双重保障，杜绝黑框弹出）
	platform.HideConsoleWindow()

	portFlag := flag.Int("port", 0, "本地服务监听端口，默认 0 表示随机分配高位端口")
	headlessFlag := flag.Bool("headless", false, "是否以纯后台/无窗口模式运行")
	configPathFlag := flag.String("config", "config.json", "配置文件存储路径")
	flag.Parse()

	// 1. 初始化配置管理器
	cfgManager, err := config.NewConfigManager(*configPathFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化配置管理器失败: %v\n", err)
		os.Exit(1)
	}

	// 2. 初始化双 Bucket 可视化工作区服务
	workspace := manager.NewWorkspaceService(cfgManager)

	// 3. 初始化嵌入式静态前端资源与 HTTP 路由
	staticFS, err := ui.GetFS()
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载静态资源失败: %v\n", err)
		os.Exit(1)
	}

	apiHandler := server.NewAPIHandler(cfgManager, workspace)
	handler := server.SetupRouter(apiHandler, staticFS)

	// 4. 绑定本地网络端口
	listenAddr := fmt.Sprintf("127.0.0.1:%d", *portFlag)
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "绑定端口失败 [%s]: %v\n", listenAddr, err)
		os.Exit(1)
	}
	defer listener.Close()

	actualPort := listener.Addr().(*net.TCPAddr).Port
	appURL := fmt.Sprintf("http://127.0.0.1:%d", actualPort)

	httpServer := &http.Server{
		Handler:      handler,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	// 启动 HTTP 服务
	go func() {
		if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "HTTP 服务异常: %v\n", err)
		}
	}()

	fmt.Printf("====================================================\n")
	fmt.Printf("  阿里云 OSS 双 Bucket 可视化文件管理器已就绪\n")
	fmt.Printf("  本地工作台地址: %s\n", appURL)
	fmt.Printf("====================================================\n")

	// 5. 桌面窗口拉起
	shutdownChan := make(chan struct{})
	if !*headlessFlag {
		go func() {
			time.Sleep(100 * time.Millisecond)
			cmd, err := platform.OpenDesktopWindow(appURL)
			if err != nil {
				fmt.Printf("拉起独立桌面窗口失败，请手动在浏览器中打开: %s (%v)\n", appURL, err)
				return
			}
			// 当用户关闭桌面独立窗口时，通知主进程优雅退出
			if cmd != nil {
				_ = cmd.Wait()
				close(shutdownChan)
			}
		}()
	}

	// 6. 优雅退出监听
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sigChan:
		fmt.Println("\n接收到退出信号，正在关闭服务...")
	case <-shutdownChan:
		fmt.Println("\n桌面窗口已关闭，正在停止后台服务...")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(ctx)

	fmt.Println("服务已完全退出。")
}
