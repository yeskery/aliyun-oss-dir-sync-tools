package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"

	"aliyun_oss_dir_sync-tools/internal/aliyunauth"
	"aliyun_oss_dir_sync-tools/internal/config"
	"aliyun_oss_dir_sync-tools/internal/manager"
	"aliyun_oss_dir_sync-tools/internal/ossclient"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// APIHandler 处理所有 RESTful API 请求
type APIHandler struct {
	cfgManager *config.ConfigManager
	workspace  *manager.WorkspaceService
}

// NewAPIHandler 创建 API 处理器
func NewAPIHandler(cfgManager *config.ConfigManager, workspace *manager.WorkspaceService) *APIHandler {
	return &APIHandler{
		cfgManager: cfgManager,
		workspace:  workspace,
	}
}

// JSONResponse 通用响应体
type JSONResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func sendJSON(w http.ResponseWriter, status int, resp JSONResponse) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

// GetStatus 获取当前系统就绪状态与摘要
func (h *APIHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	cfg := h.cfgManager.Get()
	configured := h.workspace.IsConfigured()
	authStatus := h.workspace.CheckAuthHealth()

	sendJSON(w, http.StatusOK, JSONResponse{
		Success: true,
		Data: map[string]interface{}{
			"configured":   configured,
			"auth_valid":   authStatus.Valid,
			"auth_expired": authStatus.Expired,
			"auth_message": authStatus.Message,
			"bucket_a": map[string]interface{}{
				"name":        cfg.BucketA.Name,
				"bucket_name": cfg.BucketA.BucketName,
				"endpoint":    cfg.BucketA.Endpoint,
				"cdn_domain":  cfg.BucketA.CDNDomain,
				"auth_type":   cfg.BucketA.AuthType,
				"has_token":   cfg.BucketA.SecurityToken != "",
			},
			"bucket_b": map[string]interface{}{
				"name":        cfg.BucketB.Name,
				"bucket_name": cfg.BucketB.BucketName,
				"endpoint":    cfg.BucketB.Endpoint,
				"cdn_domain":  cfg.BucketB.CDNDomain,
				"auth_type":   cfg.BucketB.AuthType,
				"has_token":   cfg.BucketB.SecurityToken != "",
			},
			"sync_dirs": cfg.SyncDirs,
		},
	})
}

// GetConfig 获取全部配置
func (h *APIHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	cfg := h.cfgManager.Get()
	sendJSON(w, http.StatusOK, JSONResponse{Success: true, Data: cfg})
}

// SaveConfig 保存配置并热重载连接
func (h *APIHandler) SaveConfig(w http.ResponseWriter, r *http.Request) {
	var newCfg config.AppConfig
	if err := json.NewDecoder(r.Body).Decode(&newCfg); err != nil {
		sendJSON(w, http.StatusBadRequest, JSONResponse{Success: false, Message: "解析配置 JSON 失败: " + err.Error()})
		return
	}

	if err := h.cfgManager.Update(newCfg); err != nil {
		sendJSON(w, http.StatusInternalServerError, JSONResponse{Success: false, Message: "保存配置文件失败: " + err.Error()})
		return
	}

	_ = h.workspace.ReloadClients()
	sendJSON(w, http.StatusOK, JSONResponse{Success: true, Message: "配置保存成功！", Data: h.cfgManager.Get()})
}

// TestConnection 测试 Bucket 连通性
func (h *APIHandler) TestConnection(w http.ResponseWriter, r *http.Request) {
	var req config.BucketConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, JSONResponse{Success: false, Message: "参数解析失败: " + err.Error()})
		return
	}

	client, err := ossclient.NewClient(req)
	if err != nil {
		sendJSON(w, http.StatusOK, JSONResponse{Success: false, Message: err.Error()})
		return
	}

	if err := client.TestConnection(); err != nil {
		sendJSON(w, http.StatusOK, JSONResponse{Success: false, Message: err.Error()})
		return
	}

	sendJSON(w, http.StatusOK, JSONResponse{Success: true, Message: "连接成功！凭据有效且 Bucket 可正常访问。"})
}

// ListFiles 列出指定目录下的文件与子目录
func (h *APIHandler) ListFiles(w http.ResponseWriter, r *http.Request) {
	dir := r.URL.Query().Get("dir")
	listing, err := h.workspace.ListDirectory(dir)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, JSONResponse{Success: false, Message: err.Error()})
		return
	}

	sendJSON(w, http.StatusOK, JSONResponse{Success: true, Data: listing})
}

// UploadFiles 处理拖拽或按钮上传（双端并发写入覆盖）
func (h *APIHandler) UploadFiles(w http.ResponseWriter, r *http.Request) {
	// 最大 500MB 限制
	if err := r.ParseMultipartForm(500 << 20); err != nil {
		sendJSON(w, http.StatusBadRequest, JSONResponse{Success: false, Message: "解析上传文件失败: " + err.Error()})
		return
	}

	dir := r.FormValue("dir")
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		sendJSON(w, http.StatusBadRequest, JSONResponse{Success: false, Message: "未检测到上传文件"})
		return
	}

	var uploadedCount int
	var errMsgs []string

	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			errMsgs = append(errMsgs, fmt.Sprintf("%s 打开失败: %v", fileHeader.Filename, err))
			continue
		}

		data, err := io.ReadAll(file)
		_ = file.Close()
		if err != nil {
			errMsgs = append(errMsgs, fmt.Sprintf("%s 读取失败: %v", fileHeader.Filename, err))
			continue
		}

		if err := h.workspace.UploadDual(dir, fileHeader.Filename, data); err != nil {
			errMsgs = append(errMsgs, fmt.Sprintf("%s 上传异常: %v", fileHeader.Filename, err))
		} else {
			uploadedCount++
		}
	}

	if len(errMsgs) > 0 {
		sendJSON(w, http.StatusMultiStatus, JSONResponse{
			Success: uploadedCount > 0,
			Message: fmt.Sprintf("上传完成，成功 %d 个，失败 %d 个: %s", uploadedCount, len(errMsgs), strings.Join(errMsgs, "; ")),
		})
		return
	}

	sendJSON(w, http.StatusOK, JSONResponse{
		Success: true,
		Message: fmt.Sprintf("已成功并发上传/覆盖 %d 个文件至两端 Bucket！", uploadedCount),
	})
}

// DeleteFile 处理文件或目录删除（双端同步彻底删除）
func (h *APIHandler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key  string   `json:"key"`
		Keys []string `json:"keys"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, JSONResponse{Success: false, Message: "参数解析失败: " + err.Error()})
		return
	}

	keysToDelete := req.Keys
	if req.Key != "" {
		keysToDelete = append(keysToDelete, req.Key)
	}

	if len(keysToDelete) == 0 {
		sendJSON(w, http.StatusBadRequest, JSONResponse{Success: false, Message: "待删除键不能为空"})
		return
	}

	var deletedCount int
	var errMsgs []string
	for _, k := range keysToDelete {
		if err := h.workspace.DeleteDual(k); err != nil {
			errMsgs = append(errMsgs, fmt.Sprintf("%s 删除失败: %v", k, err))
		} else {
			deletedCount++
		}
	}

	if len(errMsgs) > 0 {
		sendJSON(w, http.StatusMultiStatus, JSONResponse{
			Success: deletedCount > 0,
			Message: fmt.Sprintf("删除完成，成功 %d 个，失败 %d 个: %s", deletedCount, len(errMsgs), strings.Join(errMsgs, "; ")),
		})
		return
	}

	sendJSON(w, http.StatusOK, JSONResponse{
		Success: true,
		Message: fmt.Sprintf("已在两端 Bucket 中同步删除 %d 个文件！", deletedCount),
	})
}

// CreateDir 创建子目录
func (h *APIHandler) CreateDir(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ParentDir string `json:"parent_dir"`
		Name      string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, JSONResponse{Success: false, Message: "参数解析失败: " + err.Error()})
		return
	}

	if err := h.workspace.CreateSubDir(req.ParentDir, req.Name); err != nil {
		sendJSON(w, http.StatusBadRequest, JSONResponse{Success: false, Message: err.Error()})
		return
	}

	sendJSON(w, http.StatusOK, JSONResponse{Success: true, Message: "目录创建成功！"})
}

// DeleteDir 删除整个文件夹及其下所有文件（双端彻底同步删除）
func (h *APIHandler) DeleteDir(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Dir string `json:"dir"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, JSONResponse{Success: false, Message: "参数解析失败: " + err.Error()})
		return
	}

	cleanDir := strings.TrimSpace(req.Dir)
	if cleanDir == "" {
		sendJSON(w, http.StatusBadRequest, JSONResponse{Success: false, Message: "待删除目录路径不能为空"})
		return
	}

	deletedCount, err := h.workspace.DeleteDualDirectory(cleanDir)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, JSONResponse{Success: false, Message: err.Error()})
		return
	}

	dirName := path.Base(strings.TrimSuffix(cleanDir, "/"))
	sendJSON(w, http.StatusOK, JSONResponse{
		Success: true,
		Message: fmt.Sprintf("文件夹【%s】已从双端 Bucket 中彻底同步删除（共清理 %d 个关联对象）！", dirName, deletedCount),
	})
}

// SyncFile 手动将单侧文件对齐到另一侧
func (h *APIHandler) SyncFile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key  string `json:"key"`
		From string `json:"from"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, JSONResponse{Success: false, Message: "参数解析失败: " + err.Error()})
		return
	}

	if err := h.workspace.SyncSingleFile(req.Key, req.From); err != nil {
		sendJSON(w, http.StatusBadRequest, JSONResponse{Success: false, Message: "同步失败: " + err.Error()})
		return
	}

	sendJSON(w, http.StatusOK, JSONResponse{Success: true, Message: "文件已成功对齐同步至对侧！"})
}

// PreviewFile 获取图片流用于界面缩略图与预览
func (h *APIHandler) PreviewFile(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	bucket := r.URL.Query().Get("bucket")

	data, cType, err := h.workspace.GetPreviewStream(bucket, key)
	if err != nil {
		http.Error(w, "无法获取文件流: "+err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", cType)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// GetFileURLs 获取文件外网访问链接（支持生成带有效期的 OSS 鉴权签名链接）
func (h *APIHandler) GetFileURLs(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		sendJSON(w, http.StatusBadRequest, JSONResponse{Success: false, Message: "文件 key 不能为空"})
		return
	}

	signedParam := r.URL.Query().Get("signed")
	signed := true
	if signedParam == "0" || strings.ToLower(signedParam) == "false" {
		signed = false
	}

	expires := int64(3600)
	if expStr := r.URL.Query().Get("expires"); expStr != "" {
		if val, err := strconv.ParseInt(expStr, 10, 64); err == nil && val > 0 {
			expires = val
		}
	}

	urls, err := h.workspace.GenerateFileURLs(key, signed, expires)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, JSONResponse{Success: false, Message: err.Error()})
		return
	}

	sendJSON(w, http.StatusOK, JSONResponse{Success: true, Data: urls})
}

// OpenAccountLogin 拉起独立 Edge/Chrome 窗口进行阿里云账号/RAM 登录
func (h *APIHandler) OpenAccountLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Target string `json:"target"` // "bucket_a" or "bucket_b"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, JSONResponse{Success: false, Message: "参数解析失败: " + err.Error()})
		return
	}

	if req.Target != "bucket_a" && req.Target != "bucket_b" {
		req.Target = "bucket_a"
	}

	info, err := aliyunauth.GetLoginManager().StartLogin(req.Target)
	if err != nil {
		sendJSON(w, http.StatusOK, JSONResponse{Success: false, Message: err.Error()})
		return
	}

	sendJSON(w, http.StatusOK, JSONResponse{Success: true, Message: info.Message, Data: info})
}

// GetAccountLoginStatus 查询账号登录状态与凭据
func (h *APIHandler) GetAccountLoginStatus(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("target")
	if target == "" {
		target = "bucket_a"
	}

	info := aliyunauth.GetLoginManager().GetStatus(target)
	sendJSON(w, http.StatusOK, JSONResponse{Success: true, Data: info})
}

// CancelAccountLogin 取消登录
func (h *APIHandler) CancelAccountLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Target string `json:"target"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Target == "" {
		req.Target = "bucket_a"
	}

	aliyunauth.GetLoginManager().CancelLogin(req.Target)
	sendJSON(w, http.StatusOK, JSONResponse{Success: true, Message: "已取消登录"})
}

// ParseAuthCode 解析 Base64 或 JSON 格式的阿里云授权码
func (h *APIHandler) ParseAuthCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, JSONResponse{Success: false, Message: "参数解析失败: " + err.Error()})
		return
	}

	parsed, err := aliyunauth.ParseAuthCode(req.Code)
	if err != nil {
		sendJSON(w, http.StatusOK, JSONResponse{Success: false, Message: "解析授权码失败: " + err.Error()})
		return
	}

	sendJSON(w, http.StatusOK, JSONResponse{Success: true, Data: parsed})
}

// ListBucketsUsingCreds 使用当前凭据获取可访问的 Bucket 列表
func (h *APIHandler) ListBucketsUsingCreds(w http.ResponseWriter, r *http.Request) {
	var req config.BucketConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, JSONResponse{Success: false, Message: "参数解析失败: " + err.Error()})
		return
	}

	endpoint := strings.TrimSpace(req.Endpoint)
	if endpoint == "" {
		endpoint = "oss-cn-hangzhou.aliyuncs.com"
	}
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimRight(endpoint, "/")

	var opts []oss.ClientOption
	if token := strings.TrimSpace(req.SecurityToken); token != "" {
		opts = append(opts, oss.SecurityToken(token))
	}

	client, err := oss.New(endpoint, strings.TrimSpace(req.AccessKeyID), strings.TrimSpace(req.AccessKeySecret), opts...)
	if err != nil {
		sendJSON(w, http.StatusOK, JSONResponse{Success: false, Message: "初始化 OSS 客户端失败: " + err.Error()})
		return
	}

	res, err := client.ListBuckets(oss.MaxKeys(100))
	if err != nil {
		sendJSON(w, http.StatusOK, JSONResponse{Success: false, Message: "获取 Bucket 列表失败 (可能缺少 ListBuckets 权限): " + err.Error()})
		return
	}

	var buckets []string
	for _, b := range res.Buckets {
		buckets = append(buckets, b.Name)
	}

	sendJSON(w, http.StatusOK, JSONResponse{Success: true, Data: buckets})
}

// GetFileACL 获取文件的读写权限
func (h *APIHandler) GetFileACL(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		sendJSON(w, http.StatusBadRequest, JSONResponse{Success: false, Message: "缺少 key 参数"})
		return
	}

	acl, err := h.workspace.GetObjectACL(key)
	if err != nil {
		sendJSON(w, http.StatusOK, JSONResponse{Success: false, Message: "获取权限失败: " + err.Error()})
		return
	}

	sendJSON(w, http.StatusOK, JSONResponse{
		Success: true,
		Data: map[string]string{
			"key": key,
			"acl": acl,
		},
	})
}

// SetFileACL 双端设置指定文件的读写权限
func (h *APIHandler) SetFileACL(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key string `json:"key"`
		ACL string `json:"acl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, JSONResponse{Success: false, Message: "参数解析失败: " + err.Error()})
		return
	}

	req.Key = strings.TrimSpace(req.Key)
	req.ACL = strings.TrimSpace(req.ACL)
	if req.Key == "" || req.ACL == "" {
		sendJSON(w, http.StatusBadRequest, JSONResponse{Success: false, Message: "key 和 acl 不能为空"})
		return
	}

	if err := h.workspace.SetObjectACLDual(req.Key, req.ACL); err != nil {
		sendJSON(w, http.StatusOK, JSONResponse{Success: false, Message: err.Error()})
		return
	}

	sendJSON(w, http.StatusOK, JSONResponse{
		Success: true,
		Message: fmt.Sprintf("文件 [%s] 读写权限已成功设置为 [%s] (双端同步生效)", req.Key, req.ACL),
	})
}

