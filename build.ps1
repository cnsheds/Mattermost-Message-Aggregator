# Mattermost Plugin Build Script for Windows PowerShell
param(
    [switch]$Help
)

if ($Help) {
    Write-Host "Mattermost Plugin Build Script"
    Write-Host "Usage: .\build.ps1"
    Write-Host ""
    Write-Host "This script builds a Mattermost plugin for multiple platforms"
    Write-Host "and creates a plugin bundle ready for installation."
    exit 0
}

# 设置插件信息
$PLUGIN_ID = "message_merger"
$PLUGIN_VERSION = "1.0.0"
$BUNDLE_NAME = "$PLUGIN_ID-$PLUGIN_VERSION.tar.gz"

Write-Host "Building Mattermost Plugin..." -ForegroundColor Green
Write-Host "Plugin ID: $PLUGIN_ID" -ForegroundColor Yellow
Write-Host "Version: $PLUGIN_VERSION" -ForegroundColor Yellow

# 检查Go是否安装
try {
    $goVersion = & go version 2>$null
    Write-Host "Found Go: $goVersion" -ForegroundColor Green
} catch {
    Write-Host "Error: Go is not installed or not in PATH" -ForegroundColor Red
    Write-Host "Please install Go from https://golang.org/dl/" -ForegroundColor Yellow
    Read-Host "Press Enter to exit"
    exit 1
}

# 检查plugin.json文件
if (!(Test-Path "plugin.json")) {
    Write-Host "Error: plugin.json not found" -ForegroundColor Red
    Write-Host "Please make sure you're in the plugin directory" -ForegroundColor Yellow
    Read-Host "Press Enter to exit"
    exit 1
}

# 检查并初始化Go模块
Write-Host "Initializing Go module dependencies..." -ForegroundColor Cyan

if (!(Test-Path "go.mod")) {
    Write-Host "Initializing go.mod..." -ForegroundColor Yellow
    & go mod init MessageMerger
}

# 下载并整理依赖
Write-Host "Downloading dependencies..." -ForegroundColor Yellow
try {
    & go mod tidy
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Running go mod download..." -ForegroundColor Yellow
        & go mod download
    }
    Write-Host "✓ Dependencies ready" -ForegroundColor Green
} catch {
    Write-Host "Warning: Some dependency issues detected, attempting to continue..." -ForegroundColor Yellow
}

# 创建构建目录
Write-Host "Creating build directories..." -ForegroundColor Cyan
$directories = @(
    "dist",
    "dist/$PLUGIN_ID",
    "dist/$PLUGIN_ID/server",
    "dist/$PLUGIN_ID/server/dist",
    "dist/$PLUGIN_ID/webapp",
    "dist/$PLUGIN_ID/webapp/dist"
)

foreach ($dir in $directories) {
    if (!(Test-Path $dir)) {
        New-Item -ItemType Directory -Path $dir -Force | Out-Null
        Write-Host "Created directory: $dir" -ForegroundColor Gray
    }
}

# 构建服务器端插件
Write-Host "Building server plugin for multiple platforms..." -ForegroundColor Cyan

$platforms = @(
    @{OS="linux"; ARCH="amd64"; OUTPUT="plugin-linux-amd64"}
    # 暂时注释掉其他平台，但保留代码以便将来使用
    # @{OS="darwin"; ARCH="amd64"; OUTPUT="plugin-darwin-amd64"},
    # @{OS="windows"; ARCH="amd64"; OUTPUT="plugin-windows-amd64.exe"}
)

foreach ($platform in $platforms) {
    Write-Host "Building for $($platform.OS) $($platform.ARCH)..." -ForegroundColor Yellow

    $env:GOOS = $platform.OS
    $env:GOARCH = $platform.ARCH

    $outputPath = "dist/$PLUGIN_ID/server/dist/$($platform.OUTPUT)"
    $buildCmd = "go build -o $outputPath plugin.go"

    try {
        Invoke-Expression $buildCmd
        if ($LASTEXITCODE -eq 0) {
            Write-Host "✓ Successfully built $($platform.OUTPUT)" -ForegroundColor Green
        } else {
            throw "Build failed with exit code $LASTEXITCODE"
        }
    } catch {
        Write-Host "✗ Error building for $($platform.OS) $($platform.ARCH): $($_.Exception.Message)" -ForegroundColor Red
        Read-Host "Press Enter to exit"
        exit 1
    }
}

# 清理环境变量
Remove-Item Env:GOOS -ErrorAction SilentlyContinue
Remove-Item Env:GOARCH -ErrorAction SilentlyContinue

# 复制文件到dist目录
Write-Host "Copying files to distribution directory..." -ForegroundColor Cyan

# 复制plugin.json
Copy-Item "plugin.json" "dist/$PLUGIN_ID/" -Force
Write-Host "✓ Copied plugin.json" -ForegroundColor Green

# 复制webapp/dist/main.js
if (Test-Path "webapp/dist/main.js") {
    Copy-Item "webapp/dist/main.js" "dist/$PLUGIN_ID/webapp/dist/" -Force
    Write-Host "✓ Copied webapp/dist/main.js" -ForegroundColor Green
} else {
    Write-Host "Warning: webapp/dist/main.js not found" -ForegroundColor Yellow
    Write-Host "Make sure to build the webapp component first" -ForegroundColor Yellow
}

Write-Host "✓ Server binaries built directly in dist directory" -ForegroundColor Green

# 创建插件包
Write-Host "Creating plugin bundle..." -ForegroundColor Cyan

# 检查tar命令是否可用
try {
    $tarVersion = & tar --version 2>$null
    Write-Host "Found tar command" -ForegroundColor Green

    # 使用tar创建压缩包 (在当前目录执行，指定相对路径)
    try {
        $currentDir = Get-Location
        Write-Host "Current directory: $currentDir" -ForegroundColor Gray
        Write-Host "Creating tar archive..." -ForegroundColor Yellow

        # 切换到dist目录再执行tar命令
        Push-Location "dist"
        try {
            $tarCommand = "tar --transform=`"s/$PLUGIN_ID/server/dist/plugin-linux-amd64/&/`" --mode=755 -czf `"$BUNDLE_NAME`" `"$PLUGIN_ID`""
            Write-Host "Executing: $tarCommand" -ForegroundColor Gray
            Invoke-Expression $tarCommand
        } finally {
            Pop-Location
        }

        if ($LASTEXITCODE -eq 0) {
            if (Test-Path "dist/$BUNDLE_NAME") {
                $bundleSize = (Get-Item "dist/$BUNDLE_NAME").Length
                Write-Host "✓ Plugin bundle created: dist/$BUNDLE_NAME ($bundleSize bytes)" -ForegroundColor Green

                # 验证tar文件内容
                Write-Host "Verifying tar file contents:" -ForegroundColor Cyan
                & tar -tzf "dist/$BUNDLE_NAME" | ForEach-Object { Write-Host "  $_" -ForegroundColor Gray }
            } else {
                throw "Bundle file was not created"
            }
        } else {
            throw "tar command failed with exit code $LASTEXITCODE"
        }
    } catch {
        Write-Host "Error creating tar archive: $($_.Exception.Message)" -ForegroundColor Red
        Write-Host "Attempting alternative method..." -ForegroundColor Yellow

        # 备用方法：切换到dist目录再执行tar
        Push-Location "dist"
        try {
            Write-Host "Alternative: Working in dist directory" -ForegroundColor Gray
            & tar -czf $BUNDLE_NAME $PLUGIN_ID
            if ($LASTEXITCODE -eq 0 -and (Test-Path $BUNDLE_NAME)) {
                $bundleSize = (Get-Item $BUNDLE_NAME).Length
                Write-Host "✓ Plugin bundle created with alternative method: $BUNDLE_NAME ($bundleSize bytes)" -ForegroundColor Green
            } else {
                throw "Alternative method also failed"
            }
        } catch {
            Write-Host "Alternative method failed: $($_.Exception.Message)" -ForegroundColor Red
            Write-Host "Plugin files are available in: dist/$PLUGIN_ID" -ForegroundColor Yellow
        } finally {
            Pop-Location
        }
    }
} catch {
    Write-Host "tar command not found, trying to create ZIP file..." -ForegroundColor Yellow

    # 使用PowerShell创建ZIP文件 (.NET 4.5+)
    try {
        $zipPath = "dist/$PLUGIN_ID-$PLUGIN_VERSION.zip"
        Add-Type -AssemblyName System.IO.Compression.FileSystem
        [System.IO.Compression.ZipFile]::CreateFromDirectory(
            (Resolve-Path "dist/$PLUGIN_ID").Path,
            (Resolve-Path "dist").Path + "/$PLUGIN_ID-$PLUGIN_VERSION.zip"
        )
        Write-Host "✓ Plugin ZIP file created: $zipPath" -ForegroundColor Green
        Write-Host "Note: Mattermost prefers .tar.gz files. Consider installing Git for Windows for tar support." -ForegroundColor Yellow
    } catch {
        Write-Host "Error creating ZIP file: $($_.Exception.Message)" -ForegroundColor Red
        Write-Host "Plugin files are manually available in: dist/$PLUGIN_ID" -ForegroundColor Yellow
        Write-Host "You can manually compress this directory for upload to Mattermost." -ForegroundColor Yellow
    }
}

# 显示构建结果
Write-Host ""
Write-Host "=== Build Summary ===" -ForegroundColor Green
Write-Host "Plugin ID: $PLUGIN_ID" -ForegroundColor White
Write-Host "Version: $PLUGIN_VERSION" -ForegroundColor White

if (Test-Path "dist/$BUNDLE_NAME") {
    Write-Host "Bundle: dist/$BUNDLE_NAME" -ForegroundColor Green
    Write-Host ""
    Write-Host "✓ Ready for installation in Mattermost!" -ForegroundColor Green
} elseif (Test-Path "dist/$PLUGIN_ID-$PLUGIN_VERSION.zip") {
    Write-Host "ZIP File: dist/$PLUGIN_ID-$PLUGIN_VERSION.zip" -ForegroundColor Green
    Write-Host ""
    Write-Host "✓ Ready for installation in Mattermost! (ZIP format)" -ForegroundColor Green
} else {
    Write-Host "Plugin Directory: dist/$PLUGIN_ID" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Please manually compress the plugin directory for installation." -ForegroundColor Yellow
}

Write-Host ""
Write-Host "Next steps:" -ForegroundColor Cyan
Write-Host "1. Go to Mattermost System Console > Plugins > Management" -ForegroundColor White
Write-Host "2. Click 'Choose File' and upload the bundle" -ForegroundColor White
Write-Host "3. Enable the plugin" -ForegroundColor White
Write-Host "4. Configure the plugin settings as needed" -ForegroundColor White

#Read-Host "Press Enter to exit"