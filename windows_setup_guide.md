# Windows 下构建 Mattermost 插件指南

在Windows环境下，原始的Makefile可能无法直接使用。我为你提供了两个专门的Windows构建脚本。

## 准备工作

### 1. 安装必要工具

**必需安装：**
- [Go语言](https://golang.org/dl/) - 下载并安装最新版本
- 确保Go已添加到PATH环境变量中

**可选安装（推荐）：**
- [Git for Windows](https://gitforwindows.org/) - 提供tar命令支持
- 或者安装 [7-Zip](https://www.7-zip.org/) 用于压缩文件

### 2. 验证安装

打开命令提示符或PowerShell，运行：
```bash
go version
```

应该显示Go版本信息。

## 构建方法

你有二种构建选择：

### 方法1：使用PowerShell脚本（推荐高级用户）

1. 右键点击 `build.ps1` → "用PowerShell运行"
2. 或在PowerShell中运行：
   ```powershell
   .\build.ps1
   或
   PowerShell -ExecutionPolicy Bypass -File .\build.ps1
   ```

### 方法2：手动构建

如果脚本不工作，可以手动执行以下步骤：

```batch
:: 创建目录
mkdir server\dist
mkdir dist\com.example.received-aggregator
mkdir dist\com.example.received-aggregator\server\dist

:: 构建不同平台版本
set GOOS=linux
set GOARCH=amd64
go build -o server\dist\plugin-linux-amd64 plugin.go

set GOOS=darwin
set GOARCH=amd64
go build -o server\dist\plugin-darwin-amd64 plugin.go

set GOOS=windows
set GOARCH=amd64
go build -o server\dist\plugin-windows-amd64.exe plugin.go

:: 复制文件
copy plugin.json dist\com.example.received-aggregator\
copy server\dist\* dist\com.example.received-aggregator\server\dist\

:: 创建压缩包（如果有tar命令）
cd dist
tar -czf com.example.received-aggregator-1.0.0.tar.gz com.example.received-aggregator
```

## 常见问题解决

### 问题1：Go命令找不到
**现象**：`'go' is not recognized as an internal or external command`

**解决方案**：
1. 确保Go已正确安装
2. 检查PATH环境变量是否包含Go的bin目录
3. 重启命令提示符或PowerShell

### 问题2：权限问题
**现象**：PowerShell显示执行策略错误

**解决方案**：
```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

### 问题3：没有tar命令
**现象**：无法创建.tar.gz文件

**解决方案**：
1. 安装Git for Windows（推荐）
2. 或者手动压缩 `dist/com.example.received-aggregator` 文件夹
3. 使用7-Zip等工具创建tar.gz格式

### 问题4：构建失败
**现象**：编译时出现错误

**解决方案**：
1. 检查plugin.go文件是否完整
2. 运行 `go mod tidy` 下载依赖
3. 确保网络连接正常

## 文件结构检查

构建前确保你的目录结构如下：
```
your-plugin-folder/
├── plugin.json
├── plugin.go
├── go.mod
├── build.bat           # 批处理脚本
├── build.ps1          # PowerShell脚本
└── README.md
```

## 构建输出

成功构建后，你会看到：
```
dist/
├── com.example.received-aggregator-1.0.0.tar.gz  # 这是要上传的文件
└── com.example.received-aggregator/              # 解压后的内容
    ├── plugin.json
    └── server/
        └── dist/
            ├── plugin-linux-amd64
            ├── plugin-darwin-amd64
            └── plugin-windows-amd64.exe
```

## 安装到Mattermost

1. 登录Mattermost管理员界面
2. 进入 **System Console** → **Plugins** → **Management**
3. 点击 **Choose File** 选择 `com.example.received-aggregator-1.0.0.tar.gz`
4. 上传并启用插件
5. 在插件设置中配置触发词

## 调试提示

如果插件无法正常工作：

1. **检查Mattermost日志**：
   - 在System Console中查看日志
   - 查找与插件相关的错误信息

2. **验证插件状态**：
   - 确保插件已启用
   - 检查插件配置是否正确

3. **测试功能**：
   - 在测试频道发送"收到"
   - 检查消息是否按预期聚合

## 性能优化建议

- 将"最大回溯消息数"设置为10-20之间
- 避免设置过多触发词
- 在高流量频道中谨慎使用

构建成功后，你就可以在Mattermost中享受自动消息聚合功能了！