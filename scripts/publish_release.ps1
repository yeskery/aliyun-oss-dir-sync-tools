param(
    [Parameter(Mandatory=$false)]
    [string]$Token = $env:GITHUB_TOKEN,
    [string]$Tag = "v1.0.0",
    [string]$ReleaseName = "v1.0.0 阿里云 OSS 双端同步可视化文件管理工作台"
)

if (-not $Token) {
    Write-Host "请输入您的 GitHub Personal Access Token (PAT):" -ForegroundColor Yellow
    $Token = Read-Host
}
if (-not $Token) {
    Write-Host "❌ 未提供 GitHub Token，终止发布！" -ForegroundColor Red
    exit 1
}

$repo = "yeskery/aliyun-oss-dir-sync-tools"
$remoteUrl = "https://$($Token)@github.com/$repo.git"

Write-Host "🚀 [1/3] 正在推送代码至 GitHub 仓库 ($repo)..." -ForegroundColor Cyan
git remote remove origin 2>$null
git remote add origin $remoteUrl
git branch -M main
git push -u origin main --force

if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ 推送代码失败，请检查 Token 权限是否包含 repo！" -ForegroundColor Red
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
## 🌟 阿里云 OSS 双 Bucket 可视化文件管理工作台 v1.0.0

一款轻量、极速、无黑框的跨平台双端 OSS 对象存储管理客户端！

### 🚀 核心特性
- **双端并发同步与实时覆盖**：支持拖拽文件/文件夹批量双写至源端与目标端；
- **智能内容一致性对比**：支持普通上传与 Multipart 分片上传 (带 -1 ETag) 的全兼容 CRC64 内容校验；
- **全新文件夹彻底删除**：支持子文件夹卡片快捷删除与目录工具栏「删除本目录」；
- **全场景网格与列表切换**：支持全文件夹、全文件以及目录文件混排的完整列表表格视图；
- **快捷复制目录路径**：一键复制当前所在的 OSS 前缀路径；
- **带有效期与签名的外网直链复制**：支持私有 Bucket 生成防盗链临时授权 URL 与 CDN 加速链接。

### 📦 各平台下载解压即用
- **Windows 桌面版 (纯GUI无黑框)**: aliyun-oss-sync.exe
- **macOS Apple Silicon (M1/M2/M3/M4)**: aliyun-oss-sync-darwin-arm64
- **macOS Intel (x86_64)**: aliyun-oss-sync-darwin-amd64
- **Linux 64位服务器版 (x86_64)**: aliyun-oss-sync-linux-amd64
- **Linux ARM64 服务器版**: aliyun-oss-sync-linux-arm64
"@

$createReleasePayload = @{
    tag_name = $Tag
    target_commitish = "main"
    name = $ReleaseName
    body = $releaseBody
    draft = $false
    prerelease = $false
} | ConvertTo-Json -Depth 4

$releaseRes = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases" -Method Post -Headers $headers -Body ([System.Text.Encoding]::UTF8.GetBytes($createReleasePayload)) -ContentType "application/json; charset=utf-8"

$uploadUrlBase = $releaseRes.upload_url -replace "\{\?name,label\}", ""
Write-Host "✅ Release 创建成功! ID: $($releaseRes.id)" -ForegroundColor Green

# 3. 上传各平台编译产物
Write-Host "📤 [3/3] 正在上传各平台可执行文件至 Releases..." -ForegroundColor Cyan

$assets = @(
    "release/aliyun-oss-sync.exe",
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

        $null = Invoke-RestMethod -Uri $uploadUrl -Method Post -Headers $uploadHeaders -Body $fileBytes
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
