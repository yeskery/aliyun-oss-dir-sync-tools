@echo off
chcp 65001 >nul
echo ====================================================
echo   正在编译 阿里云 OSS 双向目录同步工具 (Windows 64位)
echo ====================================================

set CGO_ENABLED=0
set GOOS=windows
set GOARCH=amd64

if not exist release mkdir release

go build -ldflags="-H windowsgui -s -w" -o release\aliyun-oss-sync.exe .

if %ERRORLEVEL% equ 0 (
    echo.
    echo 编译成功！输出文件: release\aliyun-oss-sync.exe
    echo.
    echo 提示: 双击 release\aliyun-oss-sync.exe 即可直接启动桌面客户端！
) else (
    echo.
    echo 编译失败，请检查上方错误提示。
)
pause
