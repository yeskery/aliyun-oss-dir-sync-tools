@echo off
chcp 65001 >nul
echo ====================================================
echo   一键跨平台打包: 阿里云 OSS 双向目录同步工作台
echo ====================================================

if not exist release mkdir release

set CGO_ENABLED=0

echo.
echo [1/5] 正在编译 Windows 64位桌面版 (纯GUI无黑框)...
set GOOS=windows
set GOARCH=amd64
go build -ldflags="-H windowsgui -s -w" -o release\aliyun-oss-sync.exe .
if %ERRORLEVEL% equ 0 (
    echo [成功] release\aliyun-oss-sync.exe
) else (
    echo [失败] Windows 64位桌面版编译失败!
)

echo.
echo [2/5] 正在编译 macOS Apple Silicon 芯片 (M1/M2/M3/M4 arm64)...
set GOOS=darwin
set GOARCH=arm64
go build -ldflags="-s -w" -o release\aliyun-oss-sync-darwin-arm64 .
if %ERRORLEVEL% equ 0 (
    echo [成功] release\aliyun-oss-sync-darwin-arm64
) else (
    echo [失败] macOS arm64 编译失败!
)

echo.
echo [3/5] 正在编译 macOS Intel 芯片 (x86_64 amd64)...
set GOOS=darwin
set GOARCH=amd64
go build -ldflags="-s -w" -o release\aliyun-oss-sync-darwin-amd64 .
if %ERRORLEVEL% equ 0 (
    echo [成功] release\aliyun-oss-sync-darwin-amd64
) else (
    echo [失败] macOS Intel amd64 编译失败!
)

echo.
echo [4/5] 正在编译 Linux 64位服务器版 (x86_64 amd64)...
set GOOS=linux
set GOARCH=amd64
go build -ldflags="-s -w" -o release\aliyun-oss-sync-linux-amd64 .
if %ERRORLEVEL% equ 0 (
    echo [成功] release\aliyun-oss-sync-linux-amd64
) else (
    echo [失败] Linux amd64 编译失败!
)

echo.
echo [5/5] 正在编译 Linux ARM64 服务器版...
set GOOS=linux
set GOARCH=arm64
go build -ldflags="-s -w" -o release\aliyun-oss-sync-linux-arm64 .
if %ERRORLEVEL% equ 0 (
    echo [成功] release\aliyun-oss-sync-linux-arm64
) else (
    echo [失败] Linux arm64 编译失败!
)

:: 恢复默认系统变量
set GOOS=
set GOARCH=

echo.
echo ====================================================
echo   全部平台打包完成！输出文件位于 release 目录下:
echo ====================================================
dir release
echo.
