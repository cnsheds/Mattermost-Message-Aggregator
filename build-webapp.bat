@echo off
echo Building MessageMerger Webapp...

:: 检查Node.js和npm是否安装
where npm >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo Error: npm is not installed or not in PATH
    echo Please install Node.js from https://nodejs.org/
    pause
    exit /b 1
)

:: 创建webapp目录结构
echo Creating webapp directory structure...
if not exist webapp mkdir webapp
if not exist webapp\src mkdir webapp\src
if not exist webapp\dist mkdir webapp\dist

:: 复制文件到webapp目录
echo Copying webapp files...
if exist webapp\src\index.js (
    echo Webapp source file already exists, skipping copy
) else (
    echo Please ensure index.js is in webapp\src\index.js
)

:: 切换到webapp目录
cd webapp

:: 安装依赖
echo Installing npm dependencies...
if not exist package.json (
    echo Error: package.json not found in webapp directory
    echo Please ensure package.json is in the webapp folder
    pause
    cd ..
    exit /b 1
)

call npm install
if %ERRORLEVEL% NEQ 0 (
    echo Error: npm install failed
    pause
    cd ..
    exit /b 1
)

:: 构建webapp
echo Building webapp bundle...
call npm run build
if %ERRORLEVEL% NEQ 0 (
    echo Error: webapp build failed
    pause
    cd ..
    exit /b 1
)

:: 检查构建结果
if exist dist\main.js (
    echo ? Webapp built successfully: webapp\dist\main.js
    for %%A in (dist\main.js) do echo Bundle size: %%~zA bytes
) else (
    echo ? Error: main.js was not generated
    pause
    cd ..
    exit /b 1
)

cd ..

echo.
echo ? Webapp build completed!
echo File location: webapp\dist\main.js
echo.
echo Next steps:
echo 1. Run the main build script to include webapp in plugin bundle
echo 2. The webapp will automatically highlight usernames in merged messages
echo.

pause