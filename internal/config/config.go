package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// BucketConfig 单个 Bucket 的连接凭据
type BucketConfig struct {
	Name            string `json:"name"`                        // 标识名称，如 "源端" 或 "目标端"
	Endpoint        string `json:"endpoint"`                    // 如 "oss-cn-hangzhou.aliyuncs.com"
	AccessKeyID     string `json:"access_key_id"`               // 阿里云 AccessKey ID
	AccessKeySecret string `json:"access_key_secret"`           // 阿里云 AccessKey Secret
	SecurityToken   string `json:"security_token,omitempty"`     // 阿里云 STS 临时 Token (账号登录/授权码生成)
	AuthType        string `json:"auth_type,omitempty"`          // 认证方式: "ak", "account", "auth_code"
	BucketName      string `json:"bucket_name"`                 // 目标 Bucket 名称
	CDNDomain       string `json:"cdn_domain,omitempty"`        // 可选 CDN 加速域名 (如 assets-dev.dongvoyage.cn)
}

// AppConfig 应用全局配置
type AppConfig struct {
	BucketA         BucketConfig `json:"bucket_a"`          // 测试环境 Bucket A
	BucketB         BucketConfig `json:"bucket_b"`          // 生产环境 Bucket B
	SyncDirs        []string     `json:"sync_dirs"`         // 需要同步的目录前缀列表（如 ["images/", "static/assets/"]）
	IntervalSeconds int          `json:"interval_seconds"`  // 轮询周期（秒），默认 5 秒
	AutoStart       bool         `json:"auto_start"`        // 是否启动即开启自动同步
	SyncMode        string       `json:"sync_mode"`         // 同步模式: "bidirectional" (双向), "a_to_b", "b_to_a"
}

// ConfigManager 线程安全的配置管理器
type ConfigManager struct {
	filePath string
	config   AppConfig
	mu       sync.RWMutex
}

// NewDefaultConfig 返回一套带合理初始值的默认配置
func NewDefaultConfig() AppConfig {
	return AppConfig{
		BucketA: BucketConfig{
			Name:            "源端 (Bucket A)",
			Endpoint:        "oss-cn-hangzhou.aliyuncs.com",
			AccessKeyID:     "",
			AccessKeySecret: "",
			BucketName:      "",
		},
		BucketB: BucketConfig{
			Name:            "目标端 (Bucket B)",
			Endpoint:        "oss-cn-hangzhou.aliyuncs.com",
			AccessKeyID:     "",
			AccessKeySecret: "",
			BucketName:      "",
		},
		SyncDirs:        []string{"images/", "static/"},
		IntervalSeconds: 5,
		AutoStart:       false,
		SyncMode:        "bidirectional",
	}
}

// ResolveConfigPath 解析实际生效的配置文件存储路径
func ResolveConfigPath(filePath string) string {
	if filePath != "config.json" && filePath != "" {
		return filePath
	}
	// 1. 如果当前工作目录下已存在 config.json，则优先使用它
	if _, err := os.Stat("config.json"); err == nil {
		return "config.json"
	}
	// 2. 检查程序同级目录下是否已有 config.json 或位于 macOS .app 包内
	if exePath, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exePath); err == nil {
			exePath = resolved
		}
		// 如果运行在 macOS App Bundle 中 (如 .../Contents/MacOS/...)
		if strings.Contains(exePath, ".app/Contents/MacOS") {
			if ucd, err := os.UserConfigDir(); err == nil {
				return filepath.Join(ucd, "aliyun-oss-sync", "config.json")
			}
		}
		// 默认存放在可执行文件所在同级目录下（绿色便携模式）
		exeDir := filepath.Dir(exePath)
		return filepath.Join(exeDir, "config.json")
	}
	return filePath
}

// NewConfigManager 创建配置管理器
func NewConfigManager(filePath string) (*ConfigManager, error) {
	resolvedPath := ResolveConfigPath(filePath)
	cm := &ConfigManager{
		filePath: resolvedPath,
		config:   NewDefaultConfig(),
	}

	if err := cm.Load(); err != nil {
		if os.IsNotExist(err) {
			// 文件不存在则写出默认配置
			if err := cm.Save(); err != nil {
				return nil, fmt.Errorf("初始化配置失败: %w", err)
			}
		} else {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
	}
	return cm, nil
}

// Load 从磁盘读取配置
func (cm *ConfigManager) Load() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	data, err := os.ReadFile(cm.filePath)
	if err != nil {
		return err
	}

	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}

	// 补全默认项并规范化目录
	if cfg.IntervalSeconds < 2 {
		cfg.IntervalSeconds = 5
	}
	if cfg.SyncMode == "" {
		cfg.SyncMode = "bidirectional"
	}
	cfg.SyncDirs = NormalizeDirs(cfg.SyncDirs)

	cm.config = cfg
	return nil
}

// Save 保存配置至磁盘
func (cm *ConfigManager) Save() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 确保所在目录存在
	dir := filepath.Dir(cm.filePath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	cm.config.SyncDirs = NormalizeDirs(cm.config.SyncDirs)
	data, err := json.MarshalIndent(cm.config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cm.filePath, data, 0644)
}

// Get 获取当前配置副本
func (cm *ConfigManager) Get() AppConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	dirsCopy := make([]string, len(cm.config.SyncDirs))
	copy(dirsCopy, cm.config.SyncDirs)

	cfg := cm.config
	cfg.SyncDirs = dirsCopy
	return cfg
}

// Update 更新配置并保存
func (cm *ConfigManager) Update(newCfg AppConfig) error {
	if newCfg.IntervalSeconds < 2 {
		newCfg.IntervalSeconds = 2
	}
	if newCfg.SyncMode == "" {
		newCfg.SyncMode = "bidirectional"
	}
	newCfg.SyncDirs = NormalizeDirs(newCfg.SyncDirs)

	cm.mu.Lock()
	cm.config = newCfg
	cm.mu.Unlock()

	return cm.Save()
}

// NormalizeDirs 规范化目录路径：去除首尾空格与前导斜杠，确保非空目录以斜杠结尾，去除重复项
func NormalizeDirs(dirs []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, d := range dirs {
		cleaned := strings.TrimSpace(d)
		cleaned = strings.ReplaceAll(cleaned, "\\", "/")
		// 去除前导斜杠
		cleaned = strings.TrimPrefix(cleaned, "/")
		if cleaned == "" {
			continue
		}
		// 确保以 / 结尾代表目录前缀
		if !strings.HasSuffix(cleaned, "/") {
			cleaned = cleaned + "/"
		}
		if !seen[cleaned] {
			seen[cleaned] = true
			result = append(result, cleaned)
		}
	}
	return result
}
