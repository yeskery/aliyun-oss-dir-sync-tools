param(
    [Parameter(Mandatory=$false)]
    [string]$Token = $env:GITHUB_TOKEN,
    [string]$Tag = "v1.0.1",
    [string]$ReleaseName = "v1.0.1 阿里云 OSS 双端同步可视化工作台 (凭据/窗口/文件夹上传升级)",
    [string]$Proxy = "http://127.0.0.1:10808"
)

[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

if (-not $Token) {
    Write-Host "请输入您的 GitHub Personal Access Token (PAT):" -ForegroundColor Yellow
    $Token = Read-Host
}
if (-not $Token) {
    Write-Host "❌ 未提供 GitHub Token，终止发布！" -ForegroundColor Red
    exit 1
}

$repo = "yeskery/aliyun-oss-dir-sync-tools"
$remoteUrl = "https://oauth2:$($Token)@github.com/$repo.git"

Write-Host "🚀 [1/3] 正在推送代码至 GitHub 仓库 ($repo)..." -ForegroundColor Cyan
git -c credential.helper="" -c http.proxy=$Proxy push -u $remoteUrl main --force

if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ 推送代码失败，请检查 Token 权限是否包含 repo 或网络代理！" -ForegroundColor Red
    exit 1
}
Write-Host "✅ 代码成功推送至 GitHub!" -ForegroundColor Green

# 2. 创建 GitHub Release
Write-Host "📦 [2/3] 正在创建 GitHub Release ($Tag)..." -ForegroundColor Cyan
$headers = @{
    "Authorization" = "token $Token"
    "Accept" = "application/vnd.github.v3+json"
}

$releaseBody = @"
## 🌟 阿里云 OSS 双 Bucket 可视化文件管理工作台 v1.0.1

一款轻量、极速、无黑框的跨平台双端 OSS 对象存储管理客户端！

### 🚀 v1.0.1 更新内容
- **🎯 凭证复用优化**：点击「复用A凭据」时仅复制 AccessKeyID / AccessKeySecret，保留目标端原有的 Bucket 名称与 CDN 加速域名，不再误覆盖；
- **🖥️ 登录窗口防死锁与全屏居中**：修正主窗口关闭事件逻辑，窗口关闭后可随时再次点击登录按钮唤起，登录窗口在屏幕正中央显示；
- **🔴 凭证失效状态同步**：连接过期或失效时，顶部状态指示器清晰提示「🔴 凭据已失效 (点击重新登录)」，支持一键唤起登录框；
- **📁 支持直接上传文件夹**：修复文件夹上传时报「Incorrect bucket name or bucket name is too long」错误，原生递归上传文件夹及内部所有子文件并自动创建前缀。

### 📦 各平台下载解压即用
- **Windows 桌面版 (含橙色图标，纯GUI无黑框)**: `aliyun-oss-sync.exe`
- **macOS Apple Silicon 桌面版 (含橙色图标，解压得 .app)**: `Aliyun-OSS-Sync-macOS-arm64.zip`
- **macOS Intel 芯片桌面版 (含橙色图标，解压得 .app)**: `Aliyun-OSS-Sync-macOS-intel.zip`
- **Linux 64位桌面分发包 (含橙色图标与快捷方式)**: `Aliyun-OSS-Sync-linux-amd64.tar.gz`
- **Linux ARM64 桌面分发包 (含橙色图标与快捷方式)**: `Aliyun-OSS-Sync-linux-arm64.tar.gz`
- **macOS 命令行原生程序**: `aliyun-oss-sync-darwin-arm64` / `aliyun-oss-sync-darwin-amd64`
- **Linux 命令行原生程序**: `aliyun-oss-sync-linux-amd64` / `aliyun-oss-sync-linux-arm64`
"@

$createReleasePayload = @{
    tag_name = $Tag
    target_commitish = "main"
    name = $ReleaseName
    body = $releaseBody
    draft = $false
    prerelease = $false
} | ConvertTo-Json -Depth 4

$restParams = @{
    Uri = "https://api.github.com/repos/$repo/releases"
    Method = "Post"
    Headers = $headers
    Body = ([System.Text.Encoding]::UTF8.GetBytes($createReleasePayload))
    ContentType = "application/json; charset=utf-8"
}
if ($Proxy) {
    $restParams["Proxy"] = $Proxy
}

$releaseRes = Invoke-RestMethod @restParams

$uploadUrlBase = $releaseRes.upload_url -replace "\{\?name,label\}", ""
Write-Host "✅ Release 创建成功! ID: $($releaseRes.id)" -ForegroundColor Green

# 3. 上传各平台编译产物
Write-Host "📤 [3/3] 正在上传各平台可执行文件至 Releases..." -ForegroundColor Cyan

$assets = @(
    "release/aliyun-oss-sync.exe",
    "release/Aliyun-OSS-Sync-macOS-arm64.zip",
    "release/Aliyun-OSS-Sync-macOS-intel.zip",
    "release/Aliyun-OSS-Sync-linux-amd64.tar.gz",
    "release/Aliyun-OSS-Sync-linux-arm64.tar.gz",
    "release/aliyun-oss-sync-darwin-arm64",
    "release/aliyun-oss-sync-darwin-amd64",
    "release/aliyun-oss-sync-linux-amd64",
    "release/aliyun-oss-sync-linux-arm64"
)

foreach ($assetPath in $assets) {
    if (Test-Path $assetPath) {
        $fileName = Split-Path $assetPath -Leaf
        Write-Host "   正在上传: $fileName ..." -ForegroundColor Gray
        $uploadUrl = "$uploadUrlBase?name=$fileName"
        $fileBytes = [System.IO.File]::ReadAllBytes((Resolve-Path $assetPath))
        
        $uploadHeaders = @{
            "Authorization" = "token $Token"
            "Content-Type" = "application/octet-stream"
        }

        $uploadParams = @{
            Uri = $uploadUrl
            Method = "Post"
            Headers = $uploadHeaders
            Body = $fileBytes
        }
        if ($Proxy) {
            $uploadParams["Proxy"] = $Proxy
        }

        $null = Invoke-RestMethod @uploadParams
        Write-Host "   ✅ $fileName 上传成功!" -ForegroundColor Green
    } else {
        Write-Host "   ⚠️ 文件不存在跳过: $assetPath" -ForegroundColor Yellow
    }
}

# 清理 remote 中的 token 避免留在 git 本地明文配置中
git remote set-url origin "https://github.com/$repo.git"

Write-Host "====================================================" -ForegroundColor Cyan
Write-Host "🎉 发布完成！项目主页与 Releases 页面:" -ForegroundColor Green
Write-Host "   代码仓库: https://github.com/$repo" -ForegroundColor White
Write-Host "   下载页面: https://github.com/$repo/releases/tag/$Tag" -ForegroundColor White
Write-Host "====================================================" -ForegroundColor Cyan
