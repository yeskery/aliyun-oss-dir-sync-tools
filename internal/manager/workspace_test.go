package manager

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"aliyun_oss_dir_sync-tools/internal/config"
	"aliyun_oss_dir_sync-tools/internal/ossclient"
)

func TestWorkspaceWhitelistValidation(t *testing.T) {
	cm, err := config.NewConfigManager("dummy_config.json")
	if err != nil {
		t.Fatalf("初始化配置失败: %v", err)
	}

	_ = cm.Update(config.AppConfig{
		SyncDirs: []string{"images/", "static/assets/"},
	})

	ws := &WorkspaceService{
		cfgManager: cm,
	}

	// 场景 1: 白名单目录及其下级文件必须允许
	validPaths := []string{
		"images/",
		"images/logo.png",
		"images/sub/2026/banner.webp",
		"static/assets/",
		"static/assets/app.js",
	}

	for _, p := range validPaths {
		if err := ws.ValidatePathInWhitelist(p); err != nil {
			t.Errorf("有效路径 [%s] 被错误拦截: %v", p, err)
		}
	}

	// 场景 2: 白名单外部的路径必须被严格拦截
	invalidPaths := []string{
		"secrets/",
		"backup/db.sql",
		"other/pic.jpg",
		"images_fake/test.png",
		"static/other/secret.txt",
	}

	for _, p := range invalidPaths {
		if err := ws.ValidatePathInWhitelist(p); err == nil {
			t.Errorf("非法路径 [%s] 应当被拦截但未被拦截", p)
		}
	}
}

func TestBreadcrumbsBuilding(t *testing.T) {
	crumbs := buildBreadcrumbs("images/banners/2026/")
	if len(crumbs) != 4 {
		t.Fatalf("期望 4 级面包屑，实际产生 %d 级", len(crumbs))
	}

	expected := []struct {
		name string
		path string
	}{
		{"根目录", ""},
		{"images", "images/"},
		{"banners", "images/banners/"},
		{"2026", "images/banners/2026/"},
	}

	for i, exp := range expected {
		if crumbs[i].Name != exp.name || crumbs[i].Path != exp.path {
			t.Errorf("第 %d 级面包屑不符合预期: 期望 {%s, %s}，实际 {%s, %s}",
				i, exp.name, exp.path, crumbs[i].Name, crumbs[i].Path)
		}
	}
}

func TestBreadcrumbsWithSyncDirs(t *testing.T) {
	syncDirs := []string{"static/images/"}
	crumbs := buildBreadcrumbs("static/images/setting/sub/", syncDirs)
	if len(crumbs) != 4 {
		t.Fatalf("期望 4 级面包屑，实际产生 %d 级: %+v", len(crumbs), crumbs)
	}

	expected := []struct {
		name string
		path string
	}{
		{"根目录", ""},
		{"static/images", "static/images/"},
		{"setting", "static/images/setting/"},
		{"sub", "static/images/setting/sub/"},
	}

	for i, exp := range expected {
		if crumbs[i].Name != exp.name || crumbs[i].Path != exp.path {
			t.Errorf("第 %d 级面包屑不符合预期: 期望 {%s, %s}，实际 {%s, %s}",
				i, exp.name, exp.path, crumbs[i].Name, crumbs[i].Path)
		}
	}
}

func TestDirListingJSONEmptySlices(t *testing.T) {
	dl := DirListing{
		CurrentDir:  "static/images/setting/",
		IsRoot:      false,
		Breadcrumbs: make([]Breadcrumb, 0),
		SubDirs:     make([]SubDirItem, 0),
		Files:       make([]FileItem, 0),
	}

	data, err := json.Marshal(dl)
	if err != nil {
		t.Fatalf("JSON 序列化失败: %v", err)
	}

	str := string(data)
	// 验证绝对不能输出 null 给前端，必须是空数组 []
	if strings.Contains(str, `"sub_dirs":null`) {
		t.Errorf("sub_dirs 不可序列化为 null")
	}
	if strings.Contains(str, `"files":null`) {
		t.Errorf("files 不可序列化为 null")
	}
	if !strings.Contains(str, `"sub_dirs":[]`) {
		t.Errorf("sub_dirs 必须序列化为 []")
	}
	if !strings.Contains(str, `"files":[]`) {
		t.Errorf("files 必须序列化为 []")
	}
}

func TestBuildPublicFileURL(t *testing.T) {
	key := "static/images/common/empty-coupon.png"

	// 场景 1: 仅配置 Bucket A OSS
	cfg1 := &config.AppConfig{
		BucketA: config.BucketConfig{
			BucketName: "dxtx-test-oss",
			Endpoint:   "oss-cn-shenzhen.aliyuncs.com",
		},
	}
	url1 := BuildPublicFileURL(cfg1, key)
	expected1 := "https://dxtx-test-oss.oss-cn-shenzhen.aliyuncs.com/static/images/common/empty-coupon.png"
	if url1 != expected1 {
		t.Errorf("场景 1 期望 [%s]，实际 [%s]", expected1, url1)
	}

	// 场景 2: Bucket A 同时配置了 CDN
	cfg2 := &config.AppConfig{
		BucketA: config.BucketConfig{
			BucketName: "dxtx-test-oss",
			Endpoint:   "oss-cn-shenzhen.aliyuncs.com",
			CDNDomain:  "assets-dev.dongvoyage.cn",
		},
	}
	url2 := BuildPublicFileURL(cfg2, key)
	expected2 := "https://assets-dev.dongvoyage.cn/static/images/common/empty-coupon.png"
	if url2 != expected2 {
		t.Errorf("场景 2 期望 [%s]，实际 [%s]", expected2, url2)
	}

	// 场景 3: Bucket A 和 Bucket B 均配置，优先使用 Bucket B 的 OSS 域名
	cfg3 := &config.AppConfig{
		BucketA: config.BucketConfig{
			BucketName: "dxtx-test-oss",
			Endpoint:   "oss-cn-shenzhen.aliyuncs.com",
		},
		BucketB: config.BucketConfig{
			BucketName: "dxtx-prod-oss",
			Endpoint:   "oss-cn-hangzhou.aliyuncs.com",
		},
	}
	url3 := BuildPublicFileURL(cfg3, key)
	expected3 := "https://dxtx-prod-oss.oss-cn-hangzhou.aliyuncs.com/static/images/common/empty-coupon.png"
	if url3 != expected3 {
		t.Errorf("场景 3 期望 [%s]，实际 [%s]", expected3, url3)
	}

	// 场景 4: Bucket B 配置了 CDN，最高优先级命中 Bucket B 的 CDN 域名
	cfg4 := &config.AppConfig{
		BucketA: config.BucketConfig{
			BucketName: "dxtx-test-oss",
			Endpoint:   "oss-cn-shenzhen.aliyuncs.com",
			CDNDomain:  "assets-dev.dongvoyage.cn",
		},
		BucketB: config.BucketConfig{
			BucketName: "dxtx-prod-oss",
			Endpoint:   "oss-cn-hangzhou.aliyuncs.com",
			CDNDomain:  "assets.dongvoyage.cn",
		},
	}
	url4 := BuildPublicFileURL(cfg4, key)
	expected4 := "https://assets.dongvoyage.cn/static/images/common/empty-coupon.png"
	if url4 != expected4 {
		t.Errorf("场景 4 期望 [%s]，实际 [%s]", expected4, url4)
	}
}

func TestBuildBucketAAndBFileURL(t *testing.T) {
	key := "static/images/topic/draft-entry.png"

	// 1. A 和 B 均配置了 CDN 域名
	cfgBothCDN := &config.AppConfig{
		BucketA: config.BucketConfig{
			BucketName: "dxtx-test-oss",
			Endpoint:   "oss-cn-shenzhen.aliyuncs.com",
			CDNDomain:  "assets-dev.dongvoyage.cn",
		},
		BucketB: config.BucketConfig{
			BucketName: "dxtx-prod-oss",
			Endpoint:   "oss-cn-shenzhen.aliyuncs.com",
			CDNDomain:  "assets-prod.dongvoyage.cn",
		},
	}
	urlA1 := BuildBucketAFileURL(cfgBothCDN, key)
	urlB1 := BuildBucketBFileURL(cfgBothCDN, key)
	wantA1 := "https://assets-dev.dongvoyage.cn/static/images/topic/draft-entry.png"
	wantB1 := "https://assets-prod.dongvoyage.cn/static/images/topic/draft-entry.png"
	if urlA1 != wantA1 {
		t.Errorf("A 端使用 CDN 期望 [%s]，实际 [%s]", wantA1, urlA1)
	}
	if urlB1 != wantB1 {
		t.Errorf("B 端使用 CDN 期望 [%s]，实际 [%s]", wantB1, urlB1)
	}

	// 2. A 和 B 均未配置 CDN 域名，应回退至各自 OSS 默认域名
	cfgNoCDN := &config.AppConfig{
		BucketA: config.BucketConfig{
			BucketName: "dxtx-test-oss",
			Endpoint:   "oss-cn-shenzhen.aliyuncs.com",
		},
		BucketB: config.BucketConfig{
			BucketName: "dxtx-prod-oss",
			Endpoint:   "oss-cn-hangzhou.aliyuncs.com",
		},
	}
	urlA2 := BuildBucketAFileURL(cfgNoCDN, key)
	urlB2 := BuildBucketBFileURL(cfgNoCDN, key)
	wantA2 := "https://dxtx-test-oss.oss-cn-shenzhen.aliyuncs.com/static/images/topic/draft-entry.png"
	wantB2 := "https://dxtx-prod-oss.oss-cn-hangzhou.aliyuncs.com/static/images/topic/draft-entry.png"
	if urlA2 != wantA2 {
		t.Errorf("A 端使用 OSS 默认期望 [%s]，实际 [%s]", wantA2, urlA2)
	}
	if urlB2 != wantB2 {
		t.Errorf("B 端使用 OSS 默认期望 [%s]，实际 [%s]", wantB2, urlB2)
	}

	// 3. 仅 A 配置 CDN，B 未配置 CDN
	cfgOnlyACDN := &config.AppConfig{
		BucketA: config.BucketConfig{
			BucketName: "dxtx-test-oss",
			Endpoint:   "oss-cn-shenzhen.aliyuncs.com",
			CDNDomain:  "assets-dev.dongvoyage.cn",
		},
		BucketB: config.BucketConfig{
			BucketName: "dxtx-prod-oss",
			Endpoint:   "oss-cn-shenzhen.aliyuncs.com",
		},
	}
	urlA3 := BuildBucketAFileURL(cfgOnlyACDN, key)
	urlB3 := BuildBucketBFileURL(cfgOnlyACDN, key)
	wantA3 := "https://assets-dev.dongvoyage.cn/static/images/topic/draft-entry.png"
	wantB3 := "https://dxtx-prod-oss.oss-cn-shenzhen.aliyuncs.com/static/images/topic/draft-entry.png"
	if urlA3 != wantA3 {
		t.Errorf("A 端使用 CDN 期望 [%s]，实际 [%s]", wantA3, urlA3)
	}
	if urlB3 != wantB3 {
		t.Errorf("B 端使用 OSS 默认期望 [%s]，实际 [%s]", wantB3, urlB3)
	}
}

func TestBuildSpecificOSSAndCDNURLs(t *testing.T) {
	key := "static/images/topic/draft-entry.png"

	cfg := &config.AppConfig{
		BucketA: config.BucketConfig{
			BucketName: "dxtx-test-oss",
			Endpoint:   "oss-cn-shenzhen.aliyuncs.com",
			CDNDomain:  "assets-dev.dongvoyage.cn",
		},
		BucketB: config.BucketConfig{
			BucketName: "dxtx-prod-oss",
			Endpoint:   "oss-cn-shenzhen.aliyuncs.com",
		},
	}

	// Bucket A
	ossA := BuildBucketAOSSURL(cfg, key)
	cdnA := BuildBucketACDNURL(cfg, key)
	wantOssA := "https://dxtx-test-oss.oss-cn-shenzhen.aliyuncs.com/static/images/topic/draft-entry.png"
	wantCdnA := "https://assets-dev.dongvoyage.cn/static/images/topic/draft-entry.png"
	if ossA != wantOssA {
		t.Errorf("BuildBucketAOSSURL = %s, want %s", ossA, wantOssA)
	}
	if cdnA != wantCdnA {
		t.Errorf("BuildBucketACDNURL = %s, want %s", cdnA, wantCdnA)
	}

	// Bucket B (无 CDN)
	ossB := BuildBucketBOSSURL(cfg, key)
	cdnB := BuildBucketBCDNURL(cfg, key)
	wantOssB := "https://dxtx-prod-oss.oss-cn-shenzhen.aliyuncs.com/static/images/topic/draft-entry.png"
	if ossB != wantOssB {
		t.Errorf("BuildBucketBOSSURL = %s, want %s", ossB, wantOssB)
	}
	if cdnB != "" {
		t.Errorf("BuildBucketBCDNURL should be empty when no CDN, got %s", cdnB)
	}
}

func TestFileItemDiffSerialization(t *testing.T) {
	now := time.Now()
	later := now.Add(time.Hour)
	item := FileItem{
		Name:         "private.html",
		Key:          "static/private.html",
		Size:         15200,
		LastModified: now,
		Status:       StatusDiff,
		InA:          true,
		InB:          true,
		SizeA:        15200,
		SizeB:        12000,
		ModTimeA:     &now,
		ModTimeB:     &later,
		ETagA:        "e10adc3949ba59abbe56e057f20f883e",
		ETagB:        "c33367701511b4f6020ec61ded352059",
	}

	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}

	if parsed["status"] != "DIFF" {
		t.Errorf("status 期望 DIFF, 得到 %v", parsed["status"])
	}
	if parsed["size_a"].(float64) != 15200 || parsed["size_b"].(float64) != 12000 {
		t.Errorf("size_a/size_b 不匹配: %v, %v", parsed["size_a"], parsed["size_b"])
	}
	if parsed["etag_a"] != "e10adc3949ba59abbe56e057f20f883e" || parsed["etag_b"] != "c33367701511b4f6020ec61ded352059" {
		t.Errorf("etag_a/etag_b 不匹配: %v, %v", parsed["etag_a"], parsed["etag_b"])
	}
}

func TestGenerateFileURLs(t *testing.T) {
	cm, err := config.NewConfigManager("dummy_config2.json")
	if err != nil {
		t.Fatalf("初始化配置失败: %v", err)
	}

	_ = cm.Update(config.AppConfig{
		BucketA: config.BucketConfig{
			BucketName: "bucket-a-test",
			Endpoint:   "oss-cn-hangzhou.aliyuncs.com",
			CDNDomain:  "cdn-a.example.com",
		},
		BucketB: config.BucketConfig{
			BucketName: "bucket-b-test",
			Endpoint:   "oss-cn-shenzhen.aliyuncs.com",
		},
		SyncDirs: []string{"static/"},
	})

	ws := NewWorkspaceService(cm)

	// 1. 测试未签名链接
	resUnsigned, err := ws.GenerateFileURLs("static/delete.webp", false, 3600)
	if err != nil {
		t.Fatalf("GenerateFileURLs 失败: %v", err)
	}
	if resUnsigned.Signed {
		t.Errorf("expected Signed = false")
	}
	if resUnsigned.BucketA.CDNURL != "https://cdn-a.example.com/static/delete.webp" {
		t.Errorf("Bucket A CDNURL 期望 https://cdn-a.example.com/static/delete.webp, 得到 %s", resUnsigned.BucketA.CDNURL)
	}
	if resUnsigned.BucketA.OSSURL != "https://bucket-a-test.oss-cn-hangzhou.aliyuncs.com/static/delete.webp" {
		t.Errorf("Bucket A OSSURL 期望 https://bucket-a-test.oss-cn-hangzhou.aliyuncs.com/static/delete.webp, 得到 %s", resUnsigned.BucketA.OSSURL)
	}

	// 2. 测试白名单拦截
	_, err = ws.GenerateFileURLs("secret/password.txt", true, 3600)
	if err == nil {
		t.Errorf("expected whitelist check to fail for unauthorized path")
	}
}

func TestCompareFileStatus(t *testing.T) {
	// 场景 1: 大小相同且 ETag 相同 -> SYNCED
	fA1 := ossclient.ObjectMeta{Key: "a.txt", Size: 2, ETag: `"098f6bcd4621d373cade4e832627b4f6"`}
	fB1 := ossclient.ObjectMeta{Key: "a.txt", Size: 2, ETag: `098f6bcd4621d373cade4e832627b4f6`}
	if st := CompareFileStatus(fA1, fB1, "", ""); st != StatusSynced {
		t.Errorf("场景 1 期望 SYNCED，实际得到 %s", st)
	}

	// 场景 2: 大小相同但内容被修改 (ETag 不同，且 CRC64 不同) -> DIFF
	fA2 := ossclient.ObjectMeta{Key: "a.txt", Size: 2, ETag: `"098f6bcd4621d373cade4e832627b4f6"`}
	fB2 := ossclient.ObjectMeta{Key: "a.txt", Size: 2, ETag: `"ad0234829205b9033196ba818f7a872b"`}
	if st := CompareFileStatus(fA2, fB2, "11111", "22222"); st != StatusDiff {
		t.Errorf("场景 2（大小相同但内容不同）期望 DIFF，实际得到 %s", st)
	}

	// 场景 3: 大小相同，一端为分片上传 Multipart(带 -1) 一端为普通上传，但两者 CRC64 一致 -> SYNCED
	fA3 := ossclient.ObjectMeta{Key: "a.txt", Size: 2, ETag: `"3E5162A3FAB2D26904CE8E6676843EB8-1"`}
	fB3 := ossclient.ObjectMeta{Key: "a.txt", Size: 2, ETag: `"1B267619C4812CC46EE281747884CA50"`}
	if st := CompareFileStatus(fA3, fB3, "13351556452576382918", "13351556452576382918"); st != StatusSynced {
		t.Errorf("场景 3（分片与普通上传但内容 CRC64 一致）期望 SYNCED，实际得到 %s", st)
	}

	// 场景 4: 大小不同 -> DIFF
	fA4 := ossclient.ObjectMeta{Key: "a.txt", Size: 2, ETag: `"098f6bcd4621d373cade4e832627b4f6"`}
	fB4 := ossclient.ObjectMeta{Key: "a.txt", Size: 5, ETag: `"ad0234829205b9033196ba818f7a872b"`}
	if st := CompareFileStatus(fA4, fB4, "", ""); st != StatusDiff {
		t.Errorf("场景 4 期望 DIFF，实际得到 %s", st)
	}

	// 场景 5: ETag 大小写与双引号兼容
	fA5 := ossclient.ObjectMeta{Key: "a.txt", Size: 100, ETag: `"D41D8CD98F00B204E9800998ECF8427E"`}
	fB5 := ossclient.ObjectMeta{Key: "a.txt", Size: 100, ETag: `d41d8cd98f00b204e9800998ecf8427e`}
	if st := CompareFileStatus(fA5, fB5, "", ""); st != StatusSynced {
		t.Errorf("场景 5 期望 SYNCED，实际得到 %s", st)
	}
}

func TestDeleteDualDirectoryValidation(t *testing.T) {
	cm, err := config.NewConfigManager("dummy_config2.json")
	if err != nil {
		t.Fatalf("初始化配置失败: %v", err)
	}

	_ = cm.Update(config.AppConfig{
		SyncDirs: []string{"images/", "static/assets/"},
	})

	ws := &WorkspaceService{
		cfgManager: cm,
	}

	// 1. 禁止删除根目录
	if _, err := ws.DeleteDualDirectory(""); err == nil || !strings.Contains(err.Error(), "禁止删除根目录") {
		t.Errorf("期望拒绝删除空根目录，实际得到 err: %v", err)
	}

	// 2. 禁止删除白名单根同步目录本身
	if _, err := ws.DeleteDualDirectory("images/"); err == nil || !strings.Contains(err.Error(), "禁止直接删除配置的根同步目录") {
		t.Errorf("期望拒绝直接删除配置的根同步目录 images/，实际得到 err: %v", err)
	}
	if _, err := ws.DeleteDualDirectory("static/assets"); err == nil || !strings.Contains(err.Error(), "禁止直接删除配置的根同步目录") {
		t.Errorf("期望拒绝直接删除配置的根同步目录 static/assets，实际得到 err: %v", err)
	}

	// 3. 禁止删除白名单之外的目录
	if _, err := ws.DeleteDualDirectory("etc/passwd/"); err == nil {
		t.Errorf("期望拒绝越界删除非白名单目录，实际得到: nil")
	}

	// 4. 允许合法的子目录通过校验（未连接客户端时预期返回 OSS 凭据未就绪，而不是白名单/根目录拦截）
	if _, err := ws.DeleteDualDirectory("images/test_sub/"); err == nil || !strings.Contains(err.Error(), "OSS 凭据未就绪") {
		t.Errorf("期望通过白名单和根目录校验后报错凭据未就绪，实际得到 err: %v", err)
	}
}

func TestCheckAuthHealth(t *testing.T) {
	ws := &WorkspaceService{}

	// 未配置
	h := ws.CheckAuthHealth()
	if h.Valid || h.Expired {
		t.Errorf("未配置时 Valid 或 Expired 应为 false，实际得到: %+v", h)
	}

	// 模拟已配置
	ws.clientA = &ossclient.OSSClient{}
	ws.clientB = &ossclient.OSSClient{}

	h = ws.CheckAuthHealth()
	if !h.Valid || h.Expired {
		t.Errorf("正常时应 Valid=true, Expired=false，实际得到: %+v", h)
	}

	// 模拟过期错误
	ws.RecordAuthError("Request Error: SecurityTokenExpired The security token has expired.")
	h = ws.CheckAuthHealth()
	if h.Valid || !h.Expired {
		t.Errorf("记录过期错误后应 Valid=false, Expired=true，实际得到: %+v", h)
	}

	// 清除错误
	ws.ClearAuthError()
	h = ws.CheckAuthHealth()
	if !h.Valid || h.Expired {
		t.Errorf("清除错误后应恢复 Valid=true, Expired=false，实际得到: %+v", h)
	}
}

func TestUploadDualSubfolderValidation(t *testing.T) {
	cm, err := config.NewConfigManager("dummy_config.json")
	if err != nil {
		t.Fatalf("初始化配置失败: %v", err)
	}

	_ = cm.Update(config.AppConfig{
		SyncDirs: []string{"images/", "static/"},
	})

	ws := &WorkspaceService{
		cfgManager: cm,
	}

	// 测试带多级子目录的文件名传入，校验白名单是否能通过 (未初始化客户端时预期报错 OSS 凭据未就绪，而不是越界拦截)
	err = ws.UploadDual("images/", "sub/deep/test.jpg", []byte("hello"))
	if err == nil || !strings.Contains(err.Error(), "OSS 凭据未就绪") {
		t.Errorf("多级子目录上传白名单校验未通过，期望凭据未就绪，实际得到: %v", err)
	}

	// 测试空目录创建
	err = ws.UploadDual("static/", "new_folder/", []byte{})
	if err == nil || !strings.Contains(err.Error(), "OSS 凭据未就绪") {
		t.Errorf("目录占位对象上传白名单校验未通过，期望凭据未就绪，实际得到: %v", err)
	}

	// 测试越界恶意跳转目录 ../ 应当被过滤后依然受白名单保护
	err = ws.UploadDual("images/", "../../../etc/passwd", []byte("bad"))
	if err == nil {
		t.Errorf("越界路径应当被拦截，但返回了 nil")
	}
}



