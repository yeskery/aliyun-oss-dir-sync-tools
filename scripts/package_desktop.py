import os
import sys
import tarfile
import zipfile
from PIL import Image

def ensure_icns():
    png_path = "ui/dist/icon.png"
    icns_path = "ui/dist/icon.icns"
    if not os.path.exists(png_path):
        print(f"Error: {png_path} not found!")
        return False
    print("Generating standard macOS icon.icns from icon.png...")
    img = Image.open(png_path)
    img.save(icns_path, format="ICNS")
    print(f"Generated {icns_path} ({os.path.getsize(icns_path)} bytes)")
    return True

def package_macos_app(binary_path, output_zip):
    if not os.path.exists(binary_path):
        print(f"Binary not found: {binary_path}")
        return False
    
    app_name = "Aliyun-OSS-Sync.app"
    print(f"Packaging macOS App -> {output_zip}...")
    
    with zipfile.ZipFile(output_zip, "w", zipfile.ZIP_DEFLATED) as zf:
        # Directories
        for d in ["", "Contents", "Contents/MacOS", "Contents/Resources"]:
            arc_dir = f"{app_name}/{d}/" if d else f"{app_name}/"
            zinfo = zipfile.ZipInfo(arc_dir)
            zinfo.external_attr = (0o755 << 16) | 0o40000
            zf.writestr(zinfo, "")
            
        # Info.plist
        with open("scripts/Info.plist", "rb") as f:
            data = f.read()
        zinfo = zipfile.ZipInfo(f"{app_name}/Contents/Info.plist")
        zinfo.external_attr = 0o644 << 16
        zf.writestr(zinfo, data)
        
        # icon.icns
        with open("ui/dist/icon.icns", "rb") as f:
            data = f.read()
        zinfo = zipfile.ZipInfo(f"{app_name}/Contents/Resources/icon.icns")
        zinfo.external_attr = 0o644 << 16
        zf.writestr(zinfo, data)
        
        # Executable binary
        with open(binary_path, "rb") as f:
            data = f.read()
        zinfo = zipfile.ZipInfo(f"{app_name}/Contents/MacOS/aliyun-oss-sync")
        zinfo.external_attr = 0o755 << 16
        zf.writestr(zinfo, data)
        
    print(f"Success: {output_zip} ({os.path.getsize(output_zip)} bytes)")
    return True

def package_linux_bundle(binary_path, output_targz):
    if not os.path.exists(binary_path):
        print(f"Binary not found: {binary_path}")
        return False
        
    print(f"Packaging Linux Bundle -> {output_targz}...")
    
    install_script = """#!/bin/sh
set -e
INSTALL_DIR="$HOME/.local/bin"
DESKTOP_DIR="$HOME/.local/share/applications"
ICON_DIR="$HOME/.local/share/icons/hicolor/512x512/apps"

mkdir -p "$INSTALL_DIR" "$DESKTOP_DIR" "$ICON_DIR"

DIR="$(cd "$(dirname "$0")" && pwd)"

cp "$DIR/aliyun-oss-sync" "$INSTALL_DIR/aliyun-oss-sync"
chmod +x "$INSTALL_DIR/aliyun-oss-sync"

cp "$DIR/icon.png" "$ICON_DIR/aliyun-oss-sync.png"

sed -e "s|^Exec=.*|Exec=$INSTALL_DIR/aliyun-oss-sync|" \
    -e "s|^Icon=.*|Icon=$ICON_DIR/aliyun-oss-sync.png|" \
    "$DIR/aliyun-oss-sync.desktop" > "$DESKTOP_DIR/aliyun-oss-sync.desktop"
chmod +x "$DESKTOP_DIR/aliyun-oss-sync.desktop"

echo "===================================================="
echo " 阿里云OSS双端同步工作台 安装成功！"
echo " 可执行文件: $INSTALL_DIR/aliyun-oss-sync"
echo " 桌面启动图标已添加至系统应用菜单。"
echo "===================================================="
""".encode('utf-8')

    with tarfile.open(output_targz, "w:gz") as tar:
        # 1. binary
        ti = tarfile.TarInfo(name="aliyun-oss-sync/aliyun-oss-sync")
        ti.size = os.path.getsize(binary_path)
        ti.mode = 0o755
        with open(binary_path, "rb") as f:
            tar.addfile(ti, f)
            
        # 2. icon.png
        ti = tarfile.TarInfo(name="aliyun-oss-sync/icon.png")
        ti.size = os.path.getsize("ui/dist/icon.png")
        ti.mode = 0o644
        with open("ui/dist/icon.png", "rb") as f:
            tar.addfile(ti, f)
            
        # 3. desktop
        ti = tarfile.TarInfo(name="aliyun-oss-sync/aliyun-oss-sync.desktop")
        ti.size = os.path.getsize("scripts/aliyun-oss-sync.desktop")
        ti.mode = 0o644
        with open("scripts/aliyun-oss-sync.desktop", "rb") as f:
            tar.addfile(ti, f)
            
        # 4. install.sh
        import io
        ti = tarfile.TarInfo(name="aliyun-oss-sync/install.sh")
        ti.size = len(install_script)
        ti.mode = 0o755
        tar.addfile(ti, io.BytesIO(install_script))
        
    print(f"Success: {output_targz} ({os.path.getsize(output_targz)} bytes)")
    return True

if __name__ == "__main__":
    ensure_icns()
    
    # macOS
    package_macos_app("release/aliyun-oss-sync-darwin-arm64", "release/Aliyun-OSS-Sync-macOS-arm64.zip")
    package_macos_app("release/aliyun-oss-sync-darwin-amd64", "release/Aliyun-OSS-Sync-macOS-intel.zip")
    
    # Linux
    package_linux_bundle("release/aliyun-oss-sync-linux-amd64", "release/Aliyun-OSS-Sync-linux-amd64.tar.gz")
    package_linux_bundle("release/aliyun-oss-sync-linux-arm64", "release/Aliyun-OSS-Sync-linux-arm64.tar.gz")
