package ossclient

import "time"

// ObjectMeta 描述 OSS 中单个对象的关键元数据
type ObjectMeta struct {
	Key          string    `json:"key"`
	ETag         string    `json:"etag"`
	LastModified time.Time `json:"last_modified"`
	Size         int64     `json:"size"`
}

// ActionType 同步动作枚举
type ActionType string

const (
	ActionCopyAtoB ActionType = "COPY_A_TO_B" // 从 Bucket A 同步/复制到 Bucket B
	ActionCopyBtoA ActionType = "COPY_B_TO_A" // 从 Bucket B 同步/复制到 Bucket A
	ActionDeleteA  ActionType = "DELETE_A"     // 删除 Bucket A 中的文件
	ActionDeleteB  ActionType = "DELETE_B"     // 删除 Bucket B 中的文件
)

// SyncPlanItem 表示差分引擎产出的单个执行指令
type SyncPlanItem struct {
	Action   ActionType `json:"action"`
	Key      string     `json:"key"`
	Reason   string     `json:"reason"`
	Size     int64      `json:"size"`
	ModTime  time.Time  `json:"mod_time"`
}
