package server

import (
	"io/fs"
	"net/http"
)

// SetupRouter 配置所有 HTTP 路由和静态前端文件服务
func SetupRouter(api *APIHandler, staticFS fs.FS) http.Handler {
	mux := http.NewServeMux()

	// API 路由
	mux.HandleFunc("/api/status", api.GetStatus)
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			api.SaveConfig(w, r)
		} else {
			api.GetConfig(w, r)
		}
	})
	mux.HandleFunc("/api/test-conn", api.TestConnection)
	mux.HandleFunc("/api/files", api.ListFiles)
	mux.HandleFunc("/api/upload", api.UploadFiles)
	mux.HandleFunc("/api/delete", api.DeleteFile)
	mux.HandleFunc("/api/delete-dir", api.DeleteDir)
	mux.HandleFunc("/api/create-dir", api.CreateDir)
	mux.HandleFunc("/api/sync-file", api.SyncFile)
	mux.HandleFunc("/api/preview", api.PreviewFile)
	mux.HandleFunc("/api/file-urls", api.GetFileURLs)
	mux.HandleFunc("/api/file-acl", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			api.SetFileACL(w, r)
		} else {
			api.GetFileACL(w, r)
		}
	})
	mux.HandleFunc("/api/auth/open-login", api.OpenAccountLogin)
	mux.HandleFunc("/api/auth/login-status", api.GetAccountLoginStatus)
	mux.HandleFunc("/api/auth/cancel-login", api.CancelAccountLogin)
	mux.HandleFunc("/api/auth/parse-code", api.ParseAuthCode)
	mux.HandleFunc("/api/auth/list-buckets", api.ListBucketsUsingCreds)

	// 嵌入式静态前端路由
	fileServer := http.FileServer(http.FS(staticFS))
	mux.Handle("/", fileServer)

	// 统一跨域与安全响应头中间件
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		mux.ServeHTTP(w, r)
	})
}
