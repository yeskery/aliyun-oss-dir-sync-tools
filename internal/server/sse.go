package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"aliyun_oss_dir_sync-tools/internal/syncer"
)

// HandleSSE 处理 SSE 实时日志流连接
func HandleSSE(logger *syncer.EventLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// 订阅日志通道
		ch := logger.Subscribe()
		defer logger.Unsubscribe(ch)

		// 先推送最近历史日志
		recent := logger.GetRecent(50)
		for _, entry := range recent {
			data, _ := json.Marshal(entry)
			fmt.Fprintf(w, "data: %s\n\n", data)
		}
		flusher.Flush()

		notify := r.Context().Done()
		for {
			select {
			case <-notify:
				return
			case entry, ok := <-ch:
				if !ok {
					return
				}
				data, err := json.Marshal(entry)
				if err == nil {
					fmt.Fprintf(w, "data: %s\n\n", data)
					flusher.Flush()
				}
			}
		}
	}
}
