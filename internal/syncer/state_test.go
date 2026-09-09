package syncer

import (
	"testing"
	"time"

	"aliyun_oss_dir_sync-tools/internal/ossclient"
)

func TestStateDiffingScenarios(t *testing.T) {
	sm := &StateManager{
		filePath:  "dummy_state.json",
		snapshots: make(map[string]ObjectState),
	}

	allowedDirs := []string{"images/", "static/"}
	t0 := time.Now().Add(-10 * time.Minute)
	t1 := time.Now().Add(-5 * time.Minute)
	t2 := time.Now()

	// 场景 1: Bucket A 新增上传图片 images/banner.png (Bucket B 暂无，快照暂无)
	// 期望: ActionCopyAtoB
	{
		objsA := map[string]ossclient.ObjectMeta{
			"images/banner.png": {Key: "images/banner.png", ETag: "etag1", LastModified: t1, Size: 1024},
		}
		objsB := map[string]ossclient.ObjectMeta{}

		plan := sm.ComputePlan(objsA, objsB, allowedDirs, "bidirectional")
		if len(plan) != 1 {
			t.Fatalf("场景1失败: 期望产生 1 条同步动作，实际产生 %d 条", len(plan))
		}
		if plan[0].Action != ossclient.ActionCopyAtoB || plan[0].Key != "images/banner.png" {
			t.Fatalf("场景1失败: 期望动作 COPY_A_TO_B images/banner.png, 实际是 %v %s", plan[0].Action, plan[0].Key)
		}

		// 模拟执行成功，更新快照
		metaA := objsA["images/banner.png"]
		sm.RecordSuccess(plan[0], &metaA, nil)
	}

	// 场景 2: Bucket B 新增上传 static/icon.svg (Bucket A 暂无)
	// 期望: ActionCopyBtoA
	{
		objsA := map[string]ossclient.ObjectMeta{
			"images/banner.png": {Key: "images/banner.png", ETag: "etag1", LastModified: t1, Size: 1024},
		}
		objsB := map[string]ossclient.ObjectMeta{
			"images/banner.png": {Key: "images/banner.png", ETag: "etag1", LastModified: t1, Size: 1024},
			"static/icon.svg":   {Key: "static/icon.svg", ETag: "etag2", LastModified: t1, Size: 512},
		}

		plan := sm.ComputePlan(objsA, objsB, allowedDirs, "bidirectional")
		if len(plan) != 1 {
			t.Fatalf("场景2失败: 期望产生 1 条动作，实际产生 %d 条", len(plan))
		}
		if plan[0].Action != ossclient.ActionCopyBtoA || plan[0].Key != "static/icon.svg" {
			t.Fatalf("场景2失败: 期望动作 COPY_B_TO_A static/icon.svg, 实际是 %v %s", plan[0].Action, plan[0].Key)
		}

		metaB := objsB["static/icon.svg"]
		sm.RecordSuccess(plan[0], nil, &metaB)
	}

	// 场景 3: 用户在 Bucket A 中删除了 images/banner.png (此前两边均存在)
	// 期望: ActionDeleteB (级联删除 Bucket B 中的对应文件)
	{
		objsA := map[string]ossclient.ObjectMeta{} // A 中已被删除
		objsB := map[string]ossclient.ObjectMeta{
			"images/banner.png": {Key: "images/banner.png", ETag: "etag1", LastModified: t1, Size: 1024},
			"static/icon.svg":   {Key: "static/icon.svg", ETag: "etag2", LastModified: t1, Size: 512},
		}

		plan := sm.ComputePlan(objsA, objsB, allowedDirs, "bidirectional")
		foundDeleteB := false
		for _, item := range plan {
			if item.Key == "images/banner.png" && item.Action == ossclient.ActionDeleteB {
				foundDeleteB = true
				sm.RecordSuccess(item, nil, nil)
			}
		}
		if !foundDeleteB {
			t.Fatalf("场景3失败: 未检测到 Bucket A 删除导致的 DELETE_B 动作: %v", plan)
		}
	}

	// 场景 4: 用户在 Bucket B 中删除了 static/icon.svg
	// 期望: ActionDeleteA (级联删除 Bucket A 中的对应文件)
	{
		objsA := map[string]ossclient.ObjectMeta{
			"static/icon.svg": {Key: "static/icon.svg", ETag: "etag2", LastModified: t1, Size: 512},
		}
		objsB := map[string]ossclient.ObjectMeta{} // B 中已被删除

		plan := sm.ComputePlan(objsA, objsB, allowedDirs, "bidirectional")
		foundDeleteA := false
		for _, item := range plan {
			if item.Key == "static/icon.svg" && item.Action == ossclient.ActionDeleteA {
				foundDeleteA = true
				sm.RecordSuccess(item, nil, nil)
			}
		}
		if !foundDeleteA {
			t.Fatalf("场景4失败: 未检测到 Bucket B 删除导致的 DELETE_A 动作: %v", plan)
		}
	}

	// 场景 5: 白名单之外的文件操作 (如 other/secret.doc) 必须被完全忽略！
	{
		objsA := map[string]ossclient.ObjectMeta{
			"other/secret.doc": {Key: "other/secret.doc", ETag: "etag_secret", LastModified: t2, Size: 100},
		}
		objsB := map[string]ossclient.ObjectMeta{
			"temp/temp.log": {Key: "temp/temp.log", ETag: "etag_log", LastModified: t2, Size: 100},
		}

		plan := sm.ComputePlan(objsA, objsB, allowedDirs, "bidirectional")
		if len(plan) != 0 {
			t.Fatalf("场景5白名单过滤失败: 白名单外的文件被纳入同步动作: %v", plan)
		}
	}

	// 场景 6: 文件更新检测 (Bucket A 更新了 static/logo.png)
	{
		// 建立基线
		sm.snapshots["static/logo.png"] = ObjectState{
			Key:      "static/logo.png",
			InA:      true,
			InB:      true,
			ETagA:    "v1",
			ETagB:    "v1",
			ModTimeA: t0,
			ModTimeB: t0,
		}

		objsA := map[string]ossclient.ObjectMeta{
			"static/logo.png": {Key: "static/logo.png", ETag: "v2", LastModified: t2, Size: 2048},
		}
		objsB := map[string]ossclient.ObjectMeta{
			"static/logo.png": {Key: "static/logo.png", ETag: "v1", LastModified: t0, Size: 1024},
		}

		plan := sm.ComputePlan(objsA, objsB, allowedDirs, "bidirectional")
		if len(plan) != 1 || plan[0].Action != ossclient.ActionCopyAtoB {
			t.Fatalf("场景6失败: 期望产生 COPY_A_TO_B 更新动作, 实际是: %v", plan)
		}
	}
}
