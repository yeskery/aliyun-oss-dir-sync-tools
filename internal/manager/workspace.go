package manager

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"aliyun_oss_dir_sync-tools/internal/config"
	"aliyun_oss_dir_sync-tools/internal/ossclient"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// SyncStatus 双端文件一致性状态
type SyncStatus string

const (
	StatusSynced SyncStatus = "SYNCED" // 两端均存在且一致
	StatusDiff   SyncStatus = "DIFF"   // 两端均存在但内容/大小不一致
	StatusOnlyA  SyncStatus = "ONLY_A" // 仅存在于 Bucket A (源端)
	StatusOnlyB  SyncStatus = "ONLY_B" // 仅存在于 Bucket B (目标端)
)

// FileItem 文件列表展示条目
type FileItem struct {
	Name         string     `json:"name"`
	Key          string     `json:"key"`
	Size         int64      `json:"size"`
	LastModified time.Time  `json:"last_modified"`
	Status       SyncStatus `json:"status"`
	InA          bool       `json:"in_a"`
	InB          bool       `json:"in_b"`
	IsImage      bool       `json:"is_image"`
	URL          string     `json:"url"`
	URLA         string     `json:"url_a,omitempty"`
	URLB         string     `json:"url_b,omitempty"`
	URLA_OSS     string     `json:"url_a_oss,omitempty"`
	URLA_CDN     string     `json:"url_a_cdn,omitempty"`
	URLB_OSS     string     `json:"url_b_oss,omitempty"`
	URLB_CDN     string     `json:"url_b_cdn,omitempty"`
	ACL          string     `json:"acl,omitempty"`
	ETagA        string     `json:"etag_a,omitempty"`
	ETagB        string     `json:"etag_b,omitempty"`
	CRCA         string     `json:"crc_a,omitempty"`
	CRCB         string     `json:"crc_b,omitempty"`
	SizeA        int64      `json:"size_a,omitempty"`
	SizeB        int64      `json:"size_b,omitempty"`
	ModTimeA     *time.Time `json:"mod_time_a,omitempty"`
	ModTimeB     *time.Time `json:"mod_time_b,omitempty"`
}

// SubDirItem 子目录条目
type SubDirItem struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// Breadcrumb 面包屑节点
type Breadcrumb struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// DirListing 目录浏览响应模型
type DirListing struct {
	CurrentDir  string       `json:"current_dir"`
	IsRoot      bool         `json:"is_root"`
	Breadcrumbs []Breadcrumb `json:"breadcrumbs"`
	SubDirs     []SubDirItem `json:"sub_dirs"`
	Files       []FileItem   `json:"files"`
}

// WorkspaceService 负责双端文件管理与双写/双删
type WorkspaceService struct {
	mu         sync.RWMutex
	cfgManager *config.ConfigManager
	clientA    *ossclient.OSSClient
	clientB    *ossclient.OSSClient
	crcCache   sync.Map // 缓存对象的 CRC64，避免高频重复请求 (key: bucket:key:etag)
}

// NewWorkspaceService 创建工作区服务
func NewWorkspaceService(cfgManager *config.ConfigManager) *WorkspaceService {
	ws := &WorkspaceService{
		cfgManager: cfgManager,
	}
	_ = ws.ReloadClients()
	return ws
}

func (ws *WorkspaceService) getCRC64(client *ossclient.OSSClient, bucketTag, key, etag string) string {
	if client == nil || key == "" {
		return ""
	}
	cacheKey := bucketTag + ":" + key + ":" + etag
	if v, ok := ws.crcCache.Load(cacheKey); ok {
		return v.(string)
	}
	crc, err := client.GetObjectCRC64(key)
	if err == nil && crc != "" {
		ws.crcCache.Store(cacheKey, crc)
		return crc
	}
	return ""
}

// ReloadClients 重新加载 OSS 客户端连接
func (ws *WorkspaceService) ReloadClients() error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	cfg := ws.cfgManager.Get()
	if cfg.BucketA.AccessKeyID == "" || cfg.BucketA.BucketName == "" ||
		cfg.BucketB.AccessKeyID == "" || cfg.BucketB.BucketName == "" {
		ws.clientA = nil
		ws.clientB = nil
		return nil
	}

	cA, errA := ossclient.NewClient(cfg.BucketA)
	if errA != nil {
		return fmt.Errorf("初始化 Bucket A 失败: %w", errA)
	}
	cB, errB := ossclient.NewClient(cfg.BucketB)
	if errB != nil {
		return fmt.Errorf("初始化 Bucket B 失败: %w", errB)
	}

	ws.clientA = cA
	ws.clientB = cB
	return nil
}

// IsConfigured 检查是否已完成凭据与目录配置
func (ws *WorkspaceService) IsConfigured() bool {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.clientA != nil && ws.clientB != nil
}

// ValidatePathInWhitelist 校验路径是否在配置的白名单目录内
func (ws *WorkspaceService) ValidatePathInWhitelist(path string) error {
	cfg := ws.cfgManager.Get()
	cleanPath := strings.TrimPrefix(strings.ReplaceAll(path, "\\", "/"), "/")

	if cleanPath == "" {
		return nil // 根目录由系统托管展示
	}

	for _, dir := range cfg.SyncDirs {
		cleanDir := strings.TrimPrefix(strings.ReplaceAll(dir, "\\", "/"), "/")
		if !strings.HasSuffix(cleanDir, "/") {
			cleanDir += "/"
		}
		if strings.HasPrefix(cleanPath, cleanDir) || strings.HasPrefix(cleanPath+"/", cleanDir) {
			return nil
		}
	}

	return fmt.Errorf("越权拦截：路径 [%s] 不在已允许的同步目录白名单内", path)
}

// CleanETag 清理并归一化 ETag 字符串（去除首尾双引号、空白并转为小写）
func CleanETag(etag string) string {
	return strings.Trim(strings.ToLower(etag), "\" \t\r\n")
}

// CompareFileStatus 比较两端对象，准确判断双端一致 (SYNCED) 还是存在差异 (DIFF)
// 结合 Size、ETag 以及 CRC64（兼容 Multipart 分片上传与普通上传比对）
func CompareFileStatus(fA, fB ossclient.ObjectMeta, crcA, crcB string) SyncStatus {
	cleanA := CleanETag(fA.ETag)
	cleanB := CleanETag(fB.ETag)

	// 1. 如果文件大小不同，必有差异
	if fA.Size != fB.Size {
		return StatusDiff
	}

	// 2. 如果两端均具备有效 ETag 且完全一致，判定为双端一致
	if cleanA != "" && cleanB != "" && cleanA == cleanB {
		return StatusSynced
	}

	// 3. 大小相同但 ETag 不匹配（常见于一端为普通上传，一端为分片上传 Multipart 带 -1）
	// 通过 CRC64 终极真实内容哈希进行准确判定
	if crcA != "" && crcB != "" {
		if crcA == crcB {
			return StatusSynced
		}
		return StatusDiff
	}

	// 4. 若无 CRC64（兜底），如果两端 ETag 不一致则判定为差异
	if cleanA != "" && cleanB != "" && cleanA != cleanB {
		return StatusDiff
	}

	return StatusSynced
}

// ListDirectory 获取指定目录下的子目录与文件
func (ws *WorkspaceService) ListDirectory(dir string) (*DirListing, error) {
	cfg := ws.cfgManager.Get()
	cleanDir := strings.TrimPrefix(strings.ReplaceAll(dir, "\\", "/"), "/")
	if cleanDir != "" && !strings.HasSuffix(cleanDir, "/") {
		cleanDir += "/"
	}

	// 1. 根目录：直接展示用户配置的所有同步根目录
	if cleanDir == "" {
		subDirs := make([]SubDirItem, 0)
		for _, d := range cfg.SyncDirs {
			cleanD := strings.TrimPrefix(strings.ReplaceAll(d, "\\", "/"), "/")
			if cleanD == "" {
				continue
			}
			if !strings.HasSuffix(cleanD, "/") {
				cleanD += "/"
			}
			name := strings.TrimSuffix(cleanD, "/")
			subDirs = append(subDirs, SubDirItem{
				Name: name,
				Path: cleanD,
			})
		}
		sort.Slice(subDirs, func(i, j int) bool {
			return subDirs[i].Name < subDirs[j].Name
		})

		return &DirListing{
			CurrentDir:  "",
			IsRoot:      true,
			Breadcrumbs: []Breadcrumb{{Name: "根目录 (全部同步目录)", Path: ""}},
			SubDirs:     subDirs,
			Files:       make([]FileItem, 0),
		}, nil
	}

	// 2. 校验目录越界
	if err := ws.ValidatePathInWhitelist(cleanDir); err != nil {
		return nil, err
	}

	ws.mu.RLock()
	cA := ws.clientA
	cB := ws.clientB
	ws.mu.RUnlock()

	if cA == nil || cB == nil {
		return nil, fmt.Errorf("OSS 客户端未配置或连接未初始化，请先在右上角完成存储桶配置")
	}

	// 3. 并发列取 Bucket A 和 Bucket B 的当前目录层级
	type listResult struct {
		subDirs []string
		files   []ossclient.ObjectMeta
		err     error
	}

	chA := make(chan listResult, 1)
	chB := make(chan listResult, 1)

	go func() {
		subDirs, files, err := cA.ListDirectory(cleanDir)
		chA <- listResult{subDirs: subDirs, files: files, err: err}
	}()
	go func() {
		subDirs, files, err := cB.ListDirectory(cleanDir)
		chB <- listResult{subDirs: subDirs, files: files, err: err}
	}()

	resA := <-chA
	resB := <-chB

	if resA.err != nil {
		return nil, fmt.Errorf("读取 Bucket A 目录失败: %w", resA.err)
	}
	if resB.err != nil {
		return nil, fmt.Errorf("读取 Bucket B 目录失败: %w", resB.err)
	}

	// 4. 合并子目录 (去重并排序)
	subDirMap := make(map[string]bool)
	combinedSubDirs := make([]SubDirItem, 0)

	for _, s := range resA.subDirs {
		subDirMap[s] = true
	}
	for _, s := range resB.subDirs {
		subDirMap[s] = true
	}

	for fullPath := range subDirMap {
		rel := strings.TrimPrefix(fullPath, cleanDir)
		name := strings.TrimSuffix(rel, "/")
		if name != "" {
			combinedSubDirs = append(combinedSubDirs, SubDirItem{
				Name: name,
				Path: fullPath,
			})
		}
	}
	sort.Slice(combinedSubDirs, func(i, j int) bool {
		return combinedSubDirs[i].Name < combinedSubDirs[j].Name
	})

	// 5. 合并文件列表并计算同步状态
	fileMapA := make(map[string]ossclient.ObjectMeta)
	fileMapB := make(map[string]ossclient.ObjectMeta)
	allKeys := make(map[string]bool)

	for _, f := range resA.files {
		fileMapA[f.Key] = f
		allKeys[f.Key] = true
	}
	for _, f := range resB.files {
		fileMapB[f.Key] = f
		allKeys[f.Key] = true
	}

	fileList := make([]FileItem, 0)
	for key := range allKeys {
		fA, hasA := fileMapA[key]
		fB, hasB := fileMapB[key]

		item := FileItem{
			Name:    filepath.Base(key),
			Key:     key,
			InA:     hasA,
			InB:     hasB,
			IsImage: isImageFile(key),
			URL:      BuildPublicFileURL(&cfg, key),
			URLA:     BuildBucketAFileURL(&cfg, key),
			URLB:     BuildBucketBFileURL(&cfg, key),
			URLA_OSS: BuildBucketAOSSURL(&cfg, key),
			URLA_CDN: BuildBucketACDNURL(&cfg, key),
			URLB_OSS: BuildBucketBOSSURL(&cfg, key),
			URLB_CDN: BuildBucketBCDNURL(&cfg, key),
		}

		if hasA && hasB {
			item.Size = fA.Size
			item.LastModified = fA.LastModified
			item.SizeA = fA.Size
			item.SizeB = fB.Size
			item.ModTimeA = &fA.LastModified
			item.ModTimeB = &fB.LastModified
			item.ETagA = fA.ETag
			item.ETagB = fB.ETag

			cleanA := CleanETag(fA.ETag)
			cleanB := CleanETag(fB.ETag)
			var crcA, crcB string
			if fA.Size == fB.Size && (cleanA == "" || cleanB == "" || cleanA != cleanB) {
				crcA = ws.getCRC64(cA, "a", key, cleanA)
				crcB = ws.getCRC64(cB, "b", key, cleanB)
				item.CRCA = crcA
				item.CRCB = crcB
			}

			item.Status = CompareFileStatus(fA, fB, crcA, crcB)
		} else if hasA && !hasB {
			item.Size = fA.Size
			item.LastModified = fA.LastModified
			item.SizeA = fA.Size
			item.ModTimeA = &fA.LastModified
			item.ETagA = fA.ETag
			item.Status = StatusOnlyA
		} else {
			item.Size = fB.Size
			item.LastModified = fB.LastModified
			item.SizeB = fB.Size
			item.ModTimeB = &fB.LastModified
			item.ETagB = fB.ETag
			item.Status = StatusOnlyB
		}

		fileList = append(fileList, item)
	}
	sort.Slice(fileList, func(i, j int) bool {
		return fileList[i].Name < fileList[j].Name
	})

	// 6. 生成面包屑路径
	breadcrumbs := buildBreadcrumbs(cleanDir, cfg.SyncDirs)

	return &DirListing{
		CurrentDir:  cleanDir,
		IsRoot:      false,
		Breadcrumbs: breadcrumbs,
		SubDirs:     combinedSubDirs,
		Files:       fileList,
	}, nil
}

// UploadDual 双写上传/覆盖文件到 Bucket A 与 Bucket B
func (ws *WorkspaceService) UploadDual(dir string, filename string, data []byte) error {
	cleanDir := strings.TrimPrefix(strings.ReplaceAll(dir, "\\", "/"), "/")
	if cleanDir != "" && !strings.HasSuffix(cleanDir, "/") {
		cleanDir += "/"
	}

	cleanFileName := strings.TrimSpace(strings.ReplaceAll(filename, "\\", "/"))
	var fullKey string
	if cleanFileName == "" || cleanFileName == "." || cleanFileName == "/" {
		if cleanDir == "" {
			return fmt.Errorf("文件名或目录名无效")
		}
		fullKey = cleanDir
	} else {
		cleanFileName = filepath.Base(cleanFileName)
		fullKey = cleanDir + cleanFileName
	}

	if err := ws.ValidatePathInWhitelist(fullKey); err != nil {
		return err
	}

	ws.mu.RLock()
	cA := ws.clientA
	cB := ws.clientB
	ws.mu.RUnlock()

	if cA == nil || cB == nil {
		return fmt.Errorf("OSS 凭据未就绪，请先完成配置")
	}

	// 双协程并发上传
	type uploadRes struct {
		bucket string
		err    error
	}
	ch := make(chan uploadRes, 2)

	go func() {
		err := cA.PutObject(fullKey, bytes.NewReader(data))
		ch <- uploadRes{bucket: "Bucket A", err: err}
	}()
	go func() {
		err := cB.PutObject(fullKey, bytes.NewReader(data))
		ch <- uploadRes{bucket: "Bucket B", err: err}
	}()

	var errMsgs []string
	for i := 0; i < 2; i++ {
		res := <-ch
		if res.err != nil {
			errMsgs = append(errMsgs, fmt.Sprintf("%s 上传失败: %v", res.bucket, res.err))
		}
	}

	if len(errMsgs) > 0 {
		return fmt.Errorf("%s", strings.Join(errMsgs, "; "))
	}

	return nil
}

// DeleteDual 双端并发删除指定文件
func (ws *WorkspaceService) DeleteDual(key string) error {
	cleanKey := strings.TrimPrefix(strings.ReplaceAll(key, "\\", "/"), "/")
	if err := ws.ValidatePathInWhitelist(cleanKey); err != nil {
		return err
	}

	ws.mu.RLock()
	cA := ws.clientA
	cB := ws.clientB
	ws.mu.RUnlock()

	if cA == nil || cB == nil {
		return fmt.Errorf("OSS 凭据未就绪")
	}

	type delRes struct {
		bucket string
		err    error
	}
	ch := make(chan delRes, 2)

	go func() {
		err := cA.DeleteObject(cleanKey)
		ch <- delRes{bucket: "Bucket A", err: err}
	}()
	go func() {
		err := cB.DeleteObject(cleanKey)
		ch <- delRes{bucket: "Bucket B", err: err}
	}()

	var errMsgs []string
	for i := 0; i < 2; i++ {
		res := <-ch
		if res.err != nil {
			errMsgs = append(errMsgs, fmt.Sprintf("%s: %v", res.bucket, res.err))
		}
	}

	if len(errMsgs) > 0 {
		return fmt.Errorf("删除异常: %s", strings.Join(errMsgs, "; "))
	}
	return nil
}

// CreateSubDir 在当前目录下创建子目录占位符
func (ws *WorkspaceService) CreateSubDir(parentDir, subDirName string) error {
	cleanParent := strings.TrimPrefix(strings.ReplaceAll(parentDir, "\\", "/"), "/")
	if cleanParent != "" && !strings.HasSuffix(cleanParent, "/") {
		cleanParent += "/"
	}

	cleanSub := strings.Trim(strings.ReplaceAll(subDirName, "\\", "/"), "/")
	if cleanSub == "" {
		return fmt.Errorf("目录名称不能为空")
	}

	dirKey := cleanParent + cleanSub + "/"
	if err := ws.ValidatePathInWhitelist(dirKey); err != nil {
		return err
	}

	// 上传空 byte 模拟目录
	return ws.UploadDual(cleanParent+cleanSub+"/", "", []byte{})
}

// DeleteDualDirectory 递归在双端 Bucket 中彻底同步删除整个文件夹及其下所有文件
func (ws *WorkspaceService) DeleteDualDirectory(dirPath string) (int, error) {
	cleanDir := strings.TrimPrefix(strings.ReplaceAll(dirPath, "\\", "/"), "/")
	if cleanDir != "" && !strings.HasSuffix(cleanDir, "/") {
		cleanDir += "/"
	}

	if cleanDir == "" {
		return 0, fmt.Errorf("禁止删除根目录")
	}

	cfg := ws.cfgManager.Get()
	// 禁止删除用户在设置中配置的白名单根目录本身（例如 static/）
	for _, syncDir := range cfg.SyncDirs {
		cleanSync := strings.TrimPrefix(strings.ReplaceAll(syncDir, "\\", "/"), "/")
		if !strings.HasSuffix(cleanSync, "/") {
			cleanSync += "/"
		}
		if cleanDir == cleanSync {
			return 0, fmt.Errorf("禁止直接删除配置的根同步目录【%s】，如需移除请前往设置编辑同步目录", syncDir)
		}
	}

	// 校验路径越界（必须在配置的白名单目录内）
	if err := ws.ValidatePathInWhitelist(cleanDir); err != nil {
		return 0, err
	}

	ws.mu.RLock()
	cA := ws.clientA
	cB := ws.clientB
	ws.mu.RUnlock()

	if cA == nil || cB == nil {
		return 0, fmt.Errorf("OSS 凭据未就绪")
	}

	type delRes struct {
		bucket  string
		deleted int
		err     error
	}
	ch := make(chan delRes, 2)

	go func() {
		cnt, err := cA.DeletePrefix(cleanDir)
		ch <- delRes{bucket: "Bucket A", deleted: cnt, err: err}
	}()
	go func() {
		cnt, err := cB.DeletePrefix(cleanDir)
		ch <- delRes{bucket: "Bucket B", deleted: cnt, err: err}
	}()

	var errMsgs []string
	maxDeleted := 0
	for i := 0; i < 2; i++ {
		res := <-ch
		if res.err != nil {
			errMsgs = append(errMsgs, fmt.Sprintf("%s: %v", res.bucket, res.err))
		}
		if res.deleted > maxDeleted {
			maxDeleted = res.deleted
		}
	}

	if len(errMsgs) > 0 {
		return maxDeleted, fmt.Errorf("删除目录失败: %s", strings.Join(errMsgs, "; "))
	}

	return maxDeleted, nil
}

// SyncSingleFile 将仅存在某一端的文件手动对齐复制到另一端
func (ws *WorkspaceService) SyncSingleFile(key, fromBucket string) error {
	cleanKey := strings.TrimPrefix(strings.ReplaceAll(key, "\\", "/"), "/")
	if err := ws.ValidatePathInWhitelist(cleanKey); err != nil {
		return err
	}

	ws.mu.RLock()
	cA := ws.clientA
	cB := ws.clientB
	ws.mu.RUnlock()

	if cA == nil || cB == nil {
		return fmt.Errorf("OSS 凭据未就绪")
	}

	if strings.ToLower(fromBucket) == "a" {
		return cB.CopyFromBucket(cA, cleanKey)
	}
	return cA.CopyFromBucket(cB, cleanKey)
}

// GetPreviewStream 获取图片或文件流用于界面即时预览（即便 Bucket 设为私有亦可直接预览）
func (ws *WorkspaceService) GetPreviewStream(bucket, key string) (data []byte, contentType string, err error) {
	cleanKey := strings.TrimPrefix(strings.ReplaceAll(key, "\\", "/"), "/")
	if err := ws.ValidatePathInWhitelist(cleanKey); err != nil {
		return nil, "", err
	}

	ws.mu.RLock()
	cA := ws.clientA
	cB := ws.clientB
	ws.mu.RUnlock()

	var client *ossclient.OSSClient
	if strings.ToLower(bucket) == "b" && cB != nil {
		client = cB
	} else if cA != nil {
		client = cA
	} else if cB != nil {
		client = cB
	} else {
		return nil, "", fmt.Errorf("OSS 客户端未就绪")
	}

	rc, err := client.GetObject(cleanKey)
	if err != nil {
		return nil, "", err
	}
	defer rc.Close()

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(rc); err != nil {
		return nil, "", err
	}

	ext := strings.ToLower(filepath.Ext(cleanKey))
	cType := "application/octet-stream"
	switch ext {
	case ".jpg", ".jpeg":
		cType = "image/jpeg"
	case ".png":
		cType = "image/png"
	case ".gif":
		cType = "image/gif"
	case ".webp":
		cType = "image/webp"
	case ".svg":
		cType = "image/svg+xml"
	}

	return buf.Bytes(), cType, nil
}

func isImageFile(key string) bool {
	ext := strings.ToLower(filepath.Ext(key))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".bmp", ".ico":
		return true
	default:
		return false
	}
}

func buildBreadcrumbs(cleanDir string, syncDirs ...[]string) []Breadcrumb {
	crumbs := make([]Breadcrumb, 0)
	crumbs = append(crumbs, Breadcrumb{
		Name: "根目录",
		Path: "",
	})
	if cleanDir == "" {
		return crumbs
	}

	cleanDir = strings.TrimPrefix(strings.ReplaceAll(cleanDir, "\\", "/"), "/")
	if !strings.HasSuffix(cleanDir, "/") {
		cleanDir += "/"
	}

	var configuredDirs []string
	if len(syncDirs) > 0 {
		configuredDirs = syncDirs[0]
	}

	// 尝试匹配所属的同步根目录白名单
	var matchedSyncDir string
	for _, d := range configuredDirs {
		sd := strings.TrimPrefix(strings.ReplaceAll(d, "\\", "/"), "/")
		if sd == "" {
			continue
		}
		if !strings.HasSuffix(sd, "/") {
			sd += "/"
		}
		if strings.HasPrefix(cleanDir, sd) {
			if len(sd) > len(matchedSyncDir) {
				matchedSyncDir = sd
			}
		}
	}

	if matchedSyncDir != "" {
		// 添加同步根目录作为第一个面包屑节点，保证点击该节点始终合法在白名单内
		crumbs = append(crumbs, Breadcrumb{
			Name: strings.TrimSuffix(matchedSyncDir, "/"),
			Path: matchedSyncDir,
		})

		// 计算其子路径层级
		subPath := strings.TrimPrefix(cleanDir, matchedSyncDir)
		subPath = strings.Trim(subPath, "/")
		if subPath != "" {
			parts := strings.Split(subPath, "/")
			curr := matchedSyncDir
			for _, p := range parts {
				if p == "" {
					continue
				}
				curr += p + "/"
				crumbs = append(crumbs, Breadcrumb{
					Name: p,
					Path: curr,
				})
			}
		}
	} else {
		// Fallback 通用逐级切分
		parts := strings.Split(strings.Trim(cleanDir, "/"), "/")
		curr := ""
		for _, p := range parts {
			if p == "" {
				continue
			}
			curr += p + "/"
			crumbs = append(crumbs, Breadcrumb{
				Name: p,
				Path: curr,
			})
		}
	}
	return crumbs
}

// BuildPublicFileURL 依据优先级规则生成文件的公开访问 URL：
// 1. 若 Bucket B 配置了 CDN 域名，优先使用 B 端 CDN 域名
// 2. 否则若 Bucket B 存储桶就绪，使用 B 端 OSS 域名
// 3. 否则若 Bucket A 配置了 CDN 域名，使用 A 端 CDN 域名
// 4. 否则使用 A 端 OSS 域名
func BuildPublicFileURL(cfg *config.AppConfig, key string) string {
	if cfg == nil {
		return ""
	}
	cleanKey := strings.TrimPrefix(strings.TrimSpace(strings.ReplaceAll(key, "\\", "/")), "/")
	if cleanKey == "" {
		return ""
	}

	formatDomainURL := func(domain string) string {
		domain = strings.TrimSpace(domain)
		domain = strings.TrimPrefix(domain, "http://")
		domain = strings.TrimPrefix(domain, "https://")
		domain = strings.TrimRight(domain, "/")
		if domain == "" {
			return ""
		}
		return fmt.Sprintf("https://%s/%s", domain, cleanKey)
	}

	formatOSSDomainURL := func(bucket, endpoint string) string {
		bucket = strings.TrimSpace(bucket)
		endpoint = strings.TrimSpace(endpoint)
		endpoint = strings.TrimPrefix(endpoint, "http://")
		endpoint = strings.TrimPrefix(endpoint, "https://")
		endpoint = strings.TrimRight(endpoint, "/")
		if bucket == "" || endpoint == "" {
			return ""
		}
		return fmt.Sprintf("https://%s.%s/%s", bucket, endpoint, cleanKey)
	}

	// 1. 优先使用 Bucket B 的 CDN 域名
	if cfg.BucketB.CDNDomain != "" {
		if u := formatDomainURL(cfg.BucketB.CDNDomain); u != "" {
			return u
		}
	}

	// 2. 其次使用 Bucket B 的 OSS 域名
	if cfg.BucketB.BucketName != "" && cfg.BucketB.Endpoint != "" {
		if u := formatOSSDomainURL(cfg.BucketB.BucketName, cfg.BucketB.Endpoint); u != "" {
			return u
		}
	}

	// 3. 再次使用 Bucket A 的 CDN 域名
	if cfg.BucketA.CDNDomain != "" {
		if u := formatDomainURL(cfg.BucketA.CDNDomain); u != "" {
			return u
		}
	}

	// 4. 兜底使用 Bucket A 的 OSS 域名
	if cfg.BucketA.BucketName != "" && cfg.BucketA.Endpoint != "" {
		if u := formatOSSDomainURL(cfg.BucketA.BucketName, cfg.BucketA.Endpoint); u != "" {
			return u
		}
	}

	return ""
}

// BuildBucketAFileURL 依据规则生成 Bucket A 的公开访问 URL：
// 若 Bucket A 配置了 CDN 域名，使用 CDN 域名；否则使用 Bucket A 的 OSS 默认域名
func BuildBucketAFileURL(cfg *config.AppConfig, key string) string {
	if cfg == nil {
		return ""
	}
	cleanKey := strings.TrimPrefix(strings.TrimSpace(strings.ReplaceAll(key, "\\", "/")), "/")
	if cleanKey == "" {
		return ""
	}

	// 1. 若配置了 CDN 域名，优先使用 CDN 域名
	if cfg.BucketA.CDNDomain != "" {
		domain := strings.TrimSpace(cfg.BucketA.CDNDomain)
		domain = strings.TrimPrefix(domain, "http://")
		domain = strings.TrimPrefix(domain, "https://")
		domain = strings.TrimRight(domain, "/")
		if domain != "" {
			return fmt.Sprintf("https://%s/%s", domain, cleanKey)
		}
	}

	// 2. 否则使用 Bucket A 的原生 OSS 域名
	bucket := strings.TrimSpace(cfg.BucketA.BucketName)
	endpoint := strings.TrimSpace(cfg.BucketA.Endpoint)
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimRight(endpoint, "/")
	if bucket != "" && endpoint != "" {
		return fmt.Sprintf("https://%s.%s/%s", bucket, endpoint, cleanKey)
	}

	return ""
}

// BuildBucketBFileURL 依据规则生成 Bucket B 的公开访问 URL：
// 若 Bucket B 配置了 CDN 域名，使用 CDN 域名；否则使用 Bucket B 的 OSS 默认域名
func BuildBucketBFileURL(cfg *config.AppConfig, key string) string {
	if cfg == nil {
		return ""
	}
	cleanKey := strings.TrimPrefix(strings.TrimSpace(strings.ReplaceAll(key, "\\", "/")), "/")
	if cleanKey == "" {
		return ""
	}

	// 1. 若配置了 CDN 域名，优先使用 CDN 域名
	if cfg.BucketB.CDNDomain != "" {
		domain := strings.TrimSpace(cfg.BucketB.CDNDomain)
		domain = strings.TrimPrefix(domain, "http://")
		domain = strings.TrimPrefix(domain, "https://")
		domain = strings.TrimRight(domain, "/")
		if domain != "" {
			return fmt.Sprintf("https://%s/%s", domain, cleanKey)
		}
	}

	// 2. 否则使用 Bucket B 的原生 OSS 域名
	bucket := strings.TrimSpace(cfg.BucketB.BucketName)
	endpoint := strings.TrimSpace(cfg.BucketB.Endpoint)
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimRight(endpoint, "/")
	if bucket != "" && endpoint != "" {
		return fmt.Sprintf("https://%s.%s/%s", bucket, endpoint, cleanKey)
	}

	return ""
}

// BuildBucketAOSSURL 生成 Bucket A 的原生 OSS 访问 URL
func BuildBucketAOSSURL(cfg *config.AppConfig, key string) string {
	if cfg == nil {
		return ""
	}
	cleanKey := strings.TrimPrefix(strings.TrimSpace(strings.ReplaceAll(key, "\\", "/")), "/")
	if cleanKey == "" {
		return ""
	}
	bucket := strings.TrimSpace(cfg.BucketA.BucketName)
	endpoint := strings.TrimSpace(cfg.BucketA.Endpoint)
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimRight(endpoint, "/")
	if bucket != "" && endpoint != "" {
		return fmt.Sprintf("https://%s.%s/%s", bucket, endpoint, cleanKey)
	}
	return ""
}

// BuildBucketACDNURL 生成 Bucket A 的 CDN 加速访问 URL（未配置则返回空）
func BuildBucketACDNURL(cfg *config.AppConfig, key string) string {
	if cfg == nil || cfg.BucketA.CDNDomain == "" {
		return ""
	}
	cleanKey := strings.TrimPrefix(strings.TrimSpace(strings.ReplaceAll(key, "\\", "/")), "/")
	if cleanKey == "" {
		return ""
	}
	domain := strings.TrimSpace(cfg.BucketA.CDNDomain)
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimRight(domain, "/")
	if domain != "" {
		return fmt.Sprintf("https://%s/%s", domain, cleanKey)
	}
	return ""
}

// BuildBucketBOSSURL 生成 Bucket B 的原生 OSS 访问 URL
func BuildBucketBOSSURL(cfg *config.AppConfig, key string) string {
	if cfg == nil {
		return ""
	}
	cleanKey := strings.TrimPrefix(strings.TrimSpace(strings.ReplaceAll(key, "\\", "/")), "/")
	if cleanKey == "" {
		return ""
	}
	bucket := strings.TrimSpace(cfg.BucketB.BucketName)
	endpoint := strings.TrimSpace(cfg.BucketB.Endpoint)
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimRight(endpoint, "/")
	if bucket != "" && endpoint != "" {
		return fmt.Sprintf("https://%s.%s/%s", bucket, endpoint, cleanKey)
	}
	return ""
}

// BuildBucketBCDNURL 生成 Bucket B 的 CDN 加速访问 URL（未配置则返回空）
func BuildBucketBCDNURL(cfg *config.AppConfig, key string) string {
	if cfg == nil || cfg.BucketB.CDNDomain == "" {
		return ""
	}
	cleanKey := strings.TrimPrefix(strings.TrimSpace(strings.ReplaceAll(key, "\\", "/")), "/")
	if cleanKey == "" {
		return ""
	}
	domain := strings.TrimSpace(cfg.BucketB.CDNDomain)
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimRight(domain, "/")
	if domain != "" {
		return fmt.Sprintf("https://%s/%s", domain, cleanKey)
	}
	return ""
}

// BucketURLItem 单端外网链接详情
type BucketURLItem struct {
	BucketName string `json:"bucket_name"`
	InBucket   bool   `json:"in_bucket"`
	OSSURL     string `json:"oss_url"`
	CDNURL     string `json:"cdn_url"`
	IsSigned   bool   `json:"is_signed"`
	Expires    int64  `json:"expires,omitempty"`
}

// FileURLsResponse 包含双端外网访问链接的数据结构
type FileURLsResponse struct {
	Key     string        `json:"key"`
	Signed  bool          `json:"signed"`
	Expires int64         `json:"expires"`
	BucketA BucketURLItem `json:"bucket_a"`
	BucketB BucketURLItem `json:"bucket_b"`
}

// GenerateFileURLs 生成双端 OSS 访问链接（支持带签名的临时访问链接与公共访问链接）
func (ws *WorkspaceService) GenerateFileURLs(key string, signed bool, expiresSec int64) (*FileURLsResponse, error) {
	cleanKey := strings.TrimPrefix(strings.ReplaceAll(key, "\\", "/"), "/")
	if err := ws.ValidatePathInWhitelist(cleanKey); err != nil {
		return nil, err
	}

	if expiresSec <= 0 {
		expiresSec = 3600
	}
	if expiresSec > 604800 { // 阿里云 OSS 签名链接最长通常支持 7 天 (604800 秒)
		expiresSec = 604800
	}

	ws.mu.RLock()
	cA := ws.clientA
	cB := ws.clientB
	cfg := ws.cfgManager.Get()
	ws.mu.RUnlock()

	resp := &FileURLsResponse{
		Key:     cleanKey,
		Signed:  signed,
		Expires: expiresSec,
		BucketA: BucketURLItem{
			BucketName: cfg.BucketA.BucketName,
			InBucket:   cA != nil,
			CDNURL:     BuildBucketACDNURL(&cfg, cleanKey),
		},
		BucketB: BucketURLItem{
			BucketName: cfg.BucketB.BucketName,
			InBucket:   cB != nil,
			CDNURL:     BuildBucketBCDNURL(&cfg, cleanKey),
		},
	}

	// 生成 Bucket A 的 OSS 链接
	if signed && cA != nil {
		signedURL, err := cA.GetSignURL(cleanKey, expiresSec)
		if err == nil && signedURL != "" {
			resp.BucketA.OSSURL = signedURL
			resp.BucketA.IsSigned = true
			resp.BucketA.Expires = expiresSec
		} else {
			resp.BucketA.OSSURL = BuildBucketAOSSURL(&cfg, cleanKey)
		}
	} else {
		resp.BucketA.OSSURL = BuildBucketAOSSURL(&cfg, cleanKey)
	}

	// 生成 Bucket B 的 OSS 链接
	if signed && cB != nil {
		signedURL, err := cB.GetSignURL(cleanKey, expiresSec)
		if err == nil && signedURL != "" {
			resp.BucketB.OSSURL = signedURL
			resp.BucketB.IsSigned = true
			resp.BucketB.Expires = expiresSec
		} else {
			resp.BucketB.OSSURL = BuildBucketBOSSURL(&cfg, cleanKey)
		}
	} else {
		resp.BucketB.OSSURL = BuildBucketBOSSURL(&cfg, cleanKey)
	}

	return resp, nil
}

// SetObjectACLDual 并发在 Bucket A 与 Bucket B 上为指定文件设置 ACL 权限
func (ws *WorkspaceService) SetObjectACLDual(key string, acl string) error {
	cleanKey := strings.TrimPrefix(strings.ReplaceAll(key, "\\", "/"), "/")
	if err := ws.ValidatePathInWhitelist(cleanKey); err != nil {
		return err
	}

	ws.mu.RLock()
	cA := ws.clientA
	cB := ws.clientB
	ws.mu.RUnlock()

	if cA == nil && cB == nil {
		return fmt.Errorf("OSS 客户端未就绪，请先配置存储桶")
	}

	var aclType oss.ACLType
	switch strings.ToLower(acl) {
	case "default", "继承bucket":
		aclType = oss.ACLDefault
	case "private", "私有":
		aclType = oss.ACLPrivate
	case "public-read", "公共读":
		aclType = oss.ACLPublicRead
	case "public-read-write", "公共读写":
		aclType = oss.ACLPublicReadWrite
	default:
		return fmt.Errorf("不支持的读写权限类型: %s", acl)
	}

	var wg sync.WaitGroup
	var errA, errB error

	if cA != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errA = cA.SetObjectACL(cleanKey, aclType)
		}()
	}

	if cB != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errB = cB.SetObjectACL(cleanKey, aclType)
		}()
	}

	wg.Wait()

	if errA != nil && errB != nil {
		return fmt.Errorf("两端设置权限均失败: 源端: %v; 目标端: %v", errA, errB)
	}
	if errA != nil {
		return fmt.Errorf("源端设置权限失败: %v (目标端已生效)", errA)
	}
	if errB != nil {
		return fmt.Errorf("目标端设置权限失败: %v (源端已生效)", errB)
	}
	return nil
}

// GetObjectACL 获取文件的 ACL 权限（优先从 B 端读取，若无则从 A 端读取）
func (ws *WorkspaceService) GetObjectACL(key string) (string, error) {
	cleanKey := strings.TrimPrefix(strings.ReplaceAll(key, "\\", "/"), "/")
	if err := ws.ValidatePathInWhitelist(cleanKey); err != nil {
		return "", err
	}

	ws.mu.RLock()
	cA := ws.clientA
	cB := ws.clientB
	ws.mu.RUnlock()

	if cB != nil {
		if acl, err := cB.GetObjectACL(cleanKey); err == nil && acl != "" {
			return acl, nil
		}
	}
	if cA != nil {
		if acl, err := cA.GetObjectACL(cleanKey); err == nil && acl != "" {
			return acl, nil
		}
	}
	return "default", nil
}
