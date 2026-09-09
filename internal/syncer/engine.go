package syncer

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"aliyun_oss_dir_sync-tools/internal/config"
	"aliyun_oss_dir_sync-tools/internal/ossclient"
)

// EngineStatus 同步引擎运行状态
type EngineStatus string

const (
	StatusStopped EngineStatus = "STOPPED"
	StatusRunning EngineStatus = "RUNNING"
	StatusSyncing EngineStatus = "SYNCING"
	StatusError   EngineStatus = "ERROR"
)

// SyncStats 同步统计数据
type SyncStats struct {
	TotalScans          int64     `json:"total_scans"`
	TotalFilesMonitored int       `json:"total_files_monitored"`
	TotalSynced         int64     `json:"total_synced"`
	TotalUploaded       int64     `json:"total_uploaded"`
	TotalDeleted        int64     `json:"total_deleted"`
	TotalFailed         int64     `json:"total_failed"`
	LastScanTime        time.Time `json:"last_scan_time"`
	LastSyncTime        time.Time `json:"last_sync_time"`
	LastErrorMessage    string    `json:"last_error_message,omitempty"`
}

// Engine 同步调度核心引擎
type Engine struct {
	mu           sync.RWMutex
	cfgManager   *config.ConfigManager
	stateManager *StateManager
	logger       *EventLogger

	clientA *ossclient.OSSClient
	clientB *ossclient.OSSClient

	status     EngineStatus
	stats      SyncStats
	cancelFunc context.CancelFunc
	isScanning int32 // 原子布尔标志，防止重入
}

// NewEngine 创建同步引擎
func NewEngine(cfgManager *config.ConfigManager, stateManager *StateManager, logger *EventLogger) *Engine {
	return &Engine{
		cfgManager:   cfgManager,
		stateManager: stateManager,
		logger:       logger,
		status:       StatusStopped,
	}
}

// GetStatus 获取当前运行状态和统计信息
func (e *Engine) GetStatus() (EngineStatus, SyncStats) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.status, e.stats
}

// Start 启动同步引擎
func (e *Engine) Start() error {
	e.mu.Lock()
	if e.status == StatusRunning || e.status == StatusSyncing {
		e.mu.Unlock()
		return fmt.Errorf("同步引擎已经在运行中")
	}

	cfg := e.cfgManager.Get()

	// 校验配置基本有效性
	if cfg.BucketA.AccessKeyID == "" || cfg.BucketA.BucketName == "" ||
		cfg.BucketB.AccessKeyID == "" || cfg.BucketB.BucketName == "" {
		e.mu.Unlock()
		return fmt.Errorf("Bucket A 或 Bucket B 的 AK/SK/Bucket 名称尚未配置完整，请先保存配置")
	}
	if len(cfg.SyncDirs) == 0 {
		e.mu.Unlock()
		return fmt.Errorf("请至少配置一个同步目录前缀（如 images/）")
	}

	// 初始化客户端
	cA, err := ossclient.NewClient(cfg.BucketA)
	if err != nil {
		e.mu.Unlock()
		return fmt.Errorf("初始化 Bucket A 失败: %w", err)
	}
	cB, err := ossclient.NewClient(cfg.BucketB)
	if err != nil {
		e.mu.Unlock()
		return fmt.Errorf("初始化 Bucket B 失败: %w", err)
	}

	e.clientA = cA
	e.clientB = cB
	e.status = StatusRunning

	ctx, cancel := context.WithCancel(context.Background())
	e.cancelFunc = cancel
	e.mu.Unlock()

	e.logger.Info("同步引擎已启动，监控目录: %v，轮询间隔: %d秒，模式: %s", cfg.SyncDirs, cfg.IntervalSeconds, cfg.SyncMode)

	// 启动后台轮询 Goroutine
	go e.runLoop(ctx, cfg.IntervalSeconds)

	// 启动时立即执行一次同步扫描
	go e.TriggerSyncNow()

	return nil
}

// Stop 停止同步引擎
func (e *Engine) Stop() {
	e.mu.Lock()
	if e.cancelFunc != nil {
		e.cancelFunc()
		e.cancelFunc = nil
	}
	e.status = StatusStopped
	e.mu.Unlock()

	e.logger.Info("同步引擎已停止")
}

// TriggerSyncNow 触发一次立即同步
func (e *Engine) TriggerSyncNow() {
	e.executeScanCycle()
}

// runLoop 定时轮询主循环
func (e *Engine) runLoop(ctx context.Context, intervalSeconds int) {
	if intervalSeconds < 2 {
		intervalSeconds = 2
	}
	ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.executeScanCycle()
		}
	}
}

// executeScanCycle 执行单次双向扫描与同步动作
func (e *Engine) executeScanCycle() {
	// 避免并发执行导致状态混乱
	if !atomic.CompareAndSwapInt32(&e.isScanning, 0, 1) {
		return
	}
	defer atomic.StoreInt32(&e.isScanning, 0)

	e.mu.Lock()
	if e.status == StatusStopped {
		e.mu.Unlock()
		return
	}
	prevStatus := e.status
	e.status = StatusSyncing
	cA := e.clientA
	cB := e.clientB
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		if e.status == StatusSyncing {
			e.status = prevStatus
		}
		e.mu.Unlock()
	}()

	cfg := e.cfgManager.Get()
	allObjsA := make(map[string]ossclient.ObjectMeta)
	allObjsB := make(map[string]ossclient.ObjectMeta)

	// 1. 扫描所有白名单配置目录
	for _, dir := range cfg.SyncDirs {
		objsA, err := cA.ListObjectsUnderPrefix(dir)
		if err != nil {
			e.recordError(fmt.Sprintf("列出 Bucket A 目录 [%s] 失败: %v", dir, err))
			return
		}
		for k, v := range objsA {
			allObjsA[k] = v
		}

		objsB, err := cB.ListObjectsUnderPrefix(dir)
		if err != nil {
			e.recordError(fmt.Sprintf("列出 Bucket B 目录 [%s] 失败: %v", dir, err))
			return
		}
		for k, v := range objsB {
			allObjsB[k] = v
		}
	}

	totalMonitored := len(allObjsA)
	for k := range allObjsB {
		if _, ok := allObjsA[k]; !ok {
			totalMonitored++
		}
	}

	// 2. 状态机差分计算需要执行的任务
	plan := e.stateManager.ComputePlan(allObjsA, allObjsB, cfg.SyncDirs, cfg.SyncMode)

	e.mu.Lock()
	e.stats.TotalScans++
	e.stats.TotalFilesMonitored = totalMonitored
	e.stats.LastScanTime = time.Now()
	e.mu.Unlock()

	// 若无需同步，定期清理废弃快照并返回
	if len(plan) == 0 {
		e.stateManager.CleanUntrackedKeys(allObjsA, allObjsB, cfg.SyncDirs)
		return
	}

	e.logger.Info("扫描完成，检测到 %d 个文件需要同步/删除操作", len(plan))

	// 3. 逐项执行任务
	for _, item := range plan {
		var execErr error

		switch item.Action {
		case ossclient.ActionCopyAtoB:
			e.logger.Upload("正在从 [Bucket A] 同步至 [Bucket B]: %s (原因: %s)", item.Key, item.Reason)
			execErr = cB.CopyFromBucket(cA, item.Key)
			if execErr == nil {
				metaA := allObjsA[item.Key]
				e.stateManager.RecordSuccess(item, &metaA, nil)
				e.logger.Success("已成功同步至 [Bucket B]: %s", item.Key)
				e.recordSyncSuccess(true, false)
			}

		case ossclient.ActionCopyBtoA:
			e.logger.Upload("正在从 [Bucket B] 同步至 [Bucket A]: %s (原因: %s)", item.Key, item.Reason)
			execErr = cA.CopyFromBucket(cB, item.Key)
			if execErr == nil {
				metaB := allObjsB[item.Key]
				e.stateManager.RecordSuccess(item, nil, &metaB)
				e.logger.Success("已成功同步至 [Bucket A]: %s", item.Key)
				e.recordSyncSuccess(true, false)
			}

		case ossclient.ActionDeleteA:
			e.logger.Delete("正在删除 [Bucket A] 对应文件: %s (原因: %s)", item.Key, item.Reason)
			execErr = cA.DeleteObject(item.Key)
			if execErr == nil {
				e.stateManager.RecordSuccess(item, nil, nil)
				e.logger.Success("已成功删除 [Bucket A] 文件: %s", item.Key)
				e.recordSyncSuccess(false, true)
			}

		case ossclient.ActionDeleteB:
			e.logger.Delete("正在删除 [Bucket B] 对应文件: %s (原因: %s)", item.Key, item.Reason)
			execErr = cB.DeleteObject(item.Key)
			if execErr == nil {
				e.stateManager.RecordSuccess(item, nil, nil)
				e.logger.Success("已成功删除 [Bucket B] 文件: %s", item.Key)
				e.recordSyncSuccess(false, true)
			}
		}

		if execErr != nil {
			e.recordError(fmt.Sprintf("执行 %s 失败 [%s]: %v", item.Action, item.Key, execErr))
		}
	}

	// 4. 持久化最新状态快照
	if err := e.stateManager.Save(); err != nil {
		e.logger.Warn("持久化同步快照失败: %v", err)
	}
}

func (e *Engine) recordSyncSuccess(isUpload, isDelete bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.stats.TotalSynced++
	if isUpload {
		e.stats.TotalUploaded++
	}
	if isDelete {
		e.stats.TotalDeleted++
	}
	e.stats.LastSyncTime = time.Now()
}

func (e *Engine) recordError(errMsg string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.stats.TotalFailed++
	e.stats.LastErrorMessage = errMsg
	e.logger.Error("%s", errMsg)
}
