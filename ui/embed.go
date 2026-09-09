package ui

import (
	"embed"
	"io/fs"
)

//go:embed dist/*
var distFS embed.FS

// GetFS 返回可直接挂载到 http.FileServer 的 dist 子文件系统
func GetFS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}
