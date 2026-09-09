package syncer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"aliyun_oss_dir_sync-tools/internal/ossclient"
)

// ObjectState 记录已知对象在双端 Bucket 的最新状态
type ObjectState struct {
	Key          string    `json:"key"`
	ETagA        string    `json:"etag_a"`
	ETagB        string    `json:"etag_b"`
	ModTimeA     time.Time `json:"mod_time_a"`
	ModTimeB     time.Time `json:"mod_time_b"`
	SizeA        int64     `json:"size_a"`
	SizeB        int64     `json:"size_b"`
	InA          bool      `json:"in_a"`
	InB          bool      `json:"in_b"`
	LastSyncTime time.Time `json:"last_sync_time"`
}

// StateManager 管理本地同步状态快照
type StateManager struct {
	mu        sync.RWMutex
	filePath  string
	snapshots map[string]ObjectState
}

// NewStateManager 创建状态管理器
func NewStateManager(filePath string) (*StateManager, error) {
	sm := &StateManager{
		filePath:  filePath,
		snapshots: make(map[string]ObjectState),
	}

	if err := sm.Load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return sm, nil
}

// Load 读取持久化快照
func (sm *StateManager) Load() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	data, err := os.ReadFile(sm.filePath)
	if err != nil {
		return err
	}

	var stateMap map[string]ObjectState
	if err := json.Unmarshal(data, &stateMap); err != nil {
		return err
	}

	sm.snapshots = stateMap
	return nil
}

// Save 将当前快照原子持久化到磁盘
func (sm *StateManager) Save() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	dir := filepath.Dir(sm.filePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	data, err := json.MarshalIndent(sm.snapshots, "", "  ")
	if err != nil {
		return err
	}

	tmpFile := sm.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return err
	}

	return os.Rename(tmpFile, sm.filePath)
}

// IsKeyAllowed 检查 Key 是否位于配置的同步目录白名单内
func IsKeyAllowed(key string, allowedDirs []string) bool {
	if len(allowedDirs) == 0 {
		return false
	}
	cleanKey := strings.TrimPrefix(strings.ReplaceAll(key, "\\", "/"), "/")
	for _, dir := range allowedDirs {
		cleanDir := strings.TrimPrefix(strings.ReplaceAll(dir, "\\", "/"), "/")
		if cleanDir == "" {
			continue
		}
		if !strings.HasSuffix(cleanDir, "/") {
			cleanDir += "/"
		}
		if strings.HasPrefix(cleanKey, cleanDir) {
			return true
		}
	}
	return false
}

// ComputePlan 对比 Bucket A、Bucket B 与上次快照，生成本次需要执行的同步指令
func (sm *StateManager) ComputePlan(
	objectsA map[string]ossclient.ObjectMeta,
	objectsB map[string]ossclient.ObjectMeta,
	allowedDirs []string,
	syncMode string,
) []ossclient.SyncPlanItem {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var plan []ossclient.SyncPlanItem
	allKeys := make(map[string]struct{})

	// 1. 搜集候选键并严格做白名单过滤
	for k := range objectsA {
		if IsKeyAllowed(k, allowedDirs) {
			allKeys[k] = struct{}{}
		}
	}
	for k := range objectsB {
		if IsKeyAllowed(k, allowedDirs) {
			allKeys[k] = struct{}{}
		}
	}
	for k := range sm.snapshots {
		if IsKeyAllowed(k, allowedDirs) {
			allKeys[k] = struct{}{}
		}
	}

	// 2. 状态机差分判断
	for key := range allKeys {
		currA, hasA := objectsA[key]
		currB, hasB := objectsB[key]
		old, hasOld := sm.snapshots[key]

		// 情况 1: A 和 B 均存在该文件
		if hasA && hasB {
			if !hasOld {
				// 首次扫描到两边均有，对比 ETag
				if currA.ETag != currB.ETag {
					if currA.LastModified.After(currB.LastModified) {
						plan = append(plan, ossclient.SyncPlanItem{
							Action:  ossclient.ActionCopyAtoB,
							Key:     key,
							Reason:  "首次同步发现内容不一致，以较新的 A 文件覆盖 B",
							Size:    currA.Size,
							ModTime: currA.LastModified,
						})
					} else {
						plan = append(plan, ossclient.SyncPlanItem{
							Action:  ossclient.ActionCopyBtoA,
							Key:     key,
							Reason:  "首次同步发现内容不一致，以较新的 B 文件覆盖 A",
							Size:    currB.Size,
							ModTime: currB.LastModified,
						})
					}
				}
				// 若 ETag 相同，已一致，不需要动作
			} else {
				// 之前记录过
				aModified := currA.ETag != old.ETagA || !old.InA
				bModified := currB.ETag != old.ETagB || !old.InB

				if aModified && !bModified {
					plan = append(plan, ossclient.SyncPlanItem{
						Action:  ossclient.ActionCopyAtoB,
						Key:     key,
						Reason:  "Bucket A 中的文件被修改，同步至 Bucket B",
						Size:    currA.Size,
						ModTime: currA.LastModified,
					})
				} else if bModified && !aModified {
					plan = append(plan, ossclient.SyncPlanItem{
						Action:  ossclient.ActionCopyBtoA,
						Key:     key,
						Reason:  "Bucket B 中的文件被修改，同步至 Bucket A",
						Size:    currB.Size,
						ModTime: currB.LastModified,
					})
				} else if aModified && bModified && currA.ETag != currB.ETag {
					// 两边同时修改冲突，按更新时间靠后覆盖
					if currA.LastModified.After(currB.LastModified) {
						plan = append(plan, ossclient.SyncPlanItem{
							Action:  ossclient.ActionCopyAtoB,
							Key:     key,
							Reason:  "两侧均有变动发生冲突，以较新的 A 文件覆盖 B",
							Size:    currA.Size,
							ModTime: currA.LastModified,
						})
					} else {
						plan = append(plan, ossclient.SyncPlanItem{
							Action:  ossclient.ActionCopyBtoA,
							Key:     key,
							Reason:  "两侧均有变动发生冲突，以较新的 B 文件覆盖 A",
							Size:    currB.Size,
							ModTime: currB.LastModified,
						})
					}
				}
			}
			continue
		}

		// 情况 2: 仅 Bucket A 存在
		if hasA && !hasB {
			if hasOld && old.InB {
				// 之前 B 存在，但现在 B 没了 -> 用户删除了 B 上的文件！
				plan = append(plan, ossclient.SyncPlanItem{
					Action:  ossclient.ActionDeleteA,
					Key:     key,
					Reason:  "Bucket B 中的文件已被删除，级联删除 Bucket A 中的文件",
					Size:    currA.Size,
					ModTime: currA.LastModified,
				})
			} else {
				// 之前 B 没有，现在 A 有 -> 用户在 A 新增了文件！
				plan = append(plan, ossclient.SyncPlanItem{
					Action:  ossclient.ActionCopyAtoB,
					Key:     key,
					Reason:  "检测到 Bucket A 新增上传文件，同步至 Bucket B",
					Size:    currA.Size,
					ModTime: currA.LastModified,
				})
			}
			continue
		}

		// 情况 3: 仅 Bucket B 存在
		if !hasA && hasB {
			if hasOld && old.InA {
				// 之前 A 存在，但现在 A 没了 -> 用户删除了 A 上的文件！
				plan = append(plan, ossclient.SyncPlanItem{
					Action:  ossclient.ActionDeleteB,
					Key:     key,
					Reason:  "Bucket A 中的文件已被删除，级联删除 Bucket B 中的文件",
					Size:    currB.Size,
					ModTime: currB.LastModified,
				})
			} else {
				// 之前 A 没有，现在 B 有 -> 用户在 B 新增了文件！
				plan = append(plan, ossclient.SyncPlanItem{
					Action:  ossclient.ActionCopyBtoA,
					Key:     key,
					Reason:  "检测到 Bucket B 新增上传文件，同步至 Bucket A",
					Size:    currB.Size,
					ModTime: currB.LastModified,
				})
			}
			continue
		}

		// 情况 4: 两端均已不存在（!hasA && !hasB），但在快照中有残留记录，将在后续同步完成后由状态维护移除
	}

	// 3. 根据指定的同步模式过滤动作
	var filteredPlan []ossclient.SyncPlanItem
	for _, item := range plan {
		if syncMode == "a_to_b" {
			// 仅允许 A 往 B 推，或者 A 删除导致 B 删除
			if item.Action == ossclient.ActionCopyBtoA || item.Action == ossclient.ActionDeleteA {
				continue
			}
		} else if syncMode == "b_to_a" {
			// 仅允许 B 往 A 推，或者 B 删除导致 A 删除
			if item.Action == ossclient.ActionCopyAtoB || item.Action == ossclient.ActionDeleteB {
				continue
			}
		}
		filteredPlan = append(filteredPlan, item)
	}

	return filteredPlan
}

// RecordSuccess 动作成功执行后更新快照
func (sm *StateManager) RecordSuccess(item ossclient.SyncPlanItem, metaA, metaB *ossclient.ObjectMeta) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	switch item.Action {
	case ossclient.ActionCopyAtoB:
		st := ObjectState{
			Key:          item.Key,
			InA:          true,
			InB:          true,
			LastSyncTime: now,
		}
		if metaA != nil {
			st.ETagA = metaA.ETag
			st.ModTimeA = metaA.LastModified
			st.SizeA = metaA.Size
		}
		if metaB != nil {
			st.ETagB = metaB.ETag
			st.ModTimeB = metaB.LastModified
			st.SizeB = metaB.Size
		} else if metaA != nil {
			// 若未单独拉取 B 的新元数据，使用 A 的 ETag 作为基准
			st.ETagB = metaA.ETag
			st.SizeB = metaA.Size
			st.ModTimeB = now
		}
		sm.snapshots[item.Key] = st

	case ossclient.ActionCopyBtoA:
		st := ObjectState{
			Key:          item.Key,
			InA:          true,
			InB:          true,
			LastSyncTime: now,
		}
		if metaB != nil {
			st.ETagB = metaB.ETag
			st.ModTimeB = metaB.LastModified
			st.SizeB = metaB.Size
		}
		if metaA != nil {
			st.ETagA = metaA.ETag
			st.ModTimeA = metaA.LastModified
			st.SizeA = metaA.Size
		} else if metaB != nil {
			st.ETagA = metaB.ETag
			st.SizeA = metaB.Size
			st.ModTimeA = now
		}
		sm.snapshots[item.Key] = st

	case ossclient.ActionDeleteA, ossclient.ActionDeleteB:
		// 删除成功后彻底从状态快照中移除
		delete(sm.snapshots, item.Key)
	}
}

// CleanUntrackedKeys 移除不在允许目录白名单中或者两侧均不存在的快照项
func (sm *StateManager) CleanUntrackedKeys(objectsA, objectsB map[string]ossclient.ObjectMeta, allowedDirs []string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for k := range sm.snapshots {
		if !IsKeyAllowed(k, allowedDirs) {
			delete(sm.snapshots, k)
			continue
		}
		_, hasA := objectsA[k]
		_, hasB := objectsB[k]
		if !hasA && !hasB {
			delete(sm.snapshots, k)
		}
	}
}

// UpdateSnapshotBaseline 在首次运行或者两侧完全对齐后，刷新快照
func (sm *StateManager) UpdateSnapshotBaseline(objectsA, objectsB map[string]ossclient.ObjectMeta, allowedDirs []string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	for k, a := range objectsA {
		if !IsKeyAllowed(k, allowedDirs) {
			continue
		}
		b, hasB := objectsB[k]
		st := ObjectState{
			Key:          k,
			InA:          true,
			ETagA:        a.ETag,
			ModTimeA:     a.LastModified,
			SizeA:        a.Size,
			InB:          hasB,
			LastSyncTime: now,
		}
		if hasB {
			st.ETagB = b.ETag
			st.ModTimeB = b.LastModified
			st.SizeB = b.Size
		}
		sm.snapshots[k] = st
	}

	for k, b := range objectsB {
		if !IsKeyAllowed(k, allowedDirs) {
			continue
		}
		if _, exists := sm.snapshots[k]; !exists {
			sm.snapshots[k] = ObjectState{
				Key:          k,
				InA:          false,
				InB:          true,
				ETagB:        b.ETag,
				ModTimeB:     b.LastModified,
				SizeB:        b.Size,
				LastSyncTime: now,
			}
		}
	}
}
