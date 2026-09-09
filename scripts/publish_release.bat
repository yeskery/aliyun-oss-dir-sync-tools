@echo off
chcp 65001 >nul
echo ====================================================
echo   一键发布到 GitHub Releases (绕过执行策略限制)
echo ====================================================

set TOKEN=%~1
if "%TOKEN%"=="" (
    set /p TOKEN="请输入您的 GitHub Token (ghp_...): "
)

if "%TOKEN%"=="" (
    echo [错误] 未输入 Token，操作取消。
    pause
    exit /b 1
)

echo.
echo 正在执行发布脚本，请稍候...
echo.

powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0publish_release.ps1" -Token "%TOKEN%"

echo.
pause
