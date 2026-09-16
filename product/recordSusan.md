


V5--------------------------------------------------------------------------------------
# Susan品牌化构建验证保姆级步骤

本指南提供Windows环境下Susan品牌化项目的完整构建和验证步骤。

## 前置条件检查

### 1. 检查必需工具
打开PowerShell，依次检查以下工具是否安装：

```powershell
# 检查Go版本
go version
# 应该显示: go version go1.26.x windows/amd64

# 检查CMake版本
cmake --version
# 应该显示: cmake version 3.24.x 或更高

# 检查Visual Studio构建工具
where cl
# 应该显示cl.exe路径，如果没有需要安装Visual Studio 2022 C++构建工具

# 检查Ninja（推荐）
where ninja
# 应该显示ninja路径，如果没有可以从GitHub下载
```

### 2. 如果缺少工具，安装指南
- **Go**: 访问 https://go.dev/dl 下载Windows安装包
- **CMake**: 访问 https://cmake.org/download/ 下载Windows安装包
- **Visual Studio**: 安装Visual Studio 2022 Community，选择"使用C++的桌面开发"工作负载
- **Ninja**: 从 https://github.com/ninja-build/ninja/releases 下载ninja.exe，放到PATH路径中

## 快速Go构建验证（推荐先做）

### 3. 快速构建Go二进制
在项目根目录 `d:\projects\susanAssist\susanPlatform\Susan` 打开PowerShell：

```powershell
# 进入项目目录
cd d:\projects\susanAssist\susanPlatform\Susan

# 快速构建（仅Go代码，不包含C++原生代码）
go build -o susan.exe .

# 验证构建产物
dir susan.exe
# 应该看到susan.exe文件
```

### 4. 测试基本功能
```powershell
# 测试版本信息
.\susan.exe --version
# 应该显示版本号

# 测试帮助信息
.\susan.exe --help
# 应该显示帮助文本，检查是否还有"ollama"残留

# 测试serve命令（需要先有原生payload）
# .\susan.exe serve
```

## 完整CMake构建验证

### 5. 清理旧的构建产物
```powershell
# 清理之前的构建
if (Test-Path build) { Remove-Item -Recurse -Force build }
if (Test-Path dist) { Remove-Item -Recurse -Force dist }
```

### 6. 配置CMake项目
```powershell
# 基础配置（CPU only）
cmake -B build -G Ninja .

# 如果有NVIDIA GPU，使用CUDA后端
# cmake -B build -G Ninja -DOLLAMA_LLAMA_BACKENDS=cuda_v13 -DCMAKE_CUDA_ARCHITECTURES=native

# 如果有AMD GPU，使用ROCm后端
# cmake -B build -G Ninja -DOLLAMA_LLAMA_BACKENDS=rocm_v7_2
```

### 7. 构建项目
```powershell
# 并行构建（使用8个线程）
cmake --build build --parallel 8

# 预计时间：CPU构建10-30分钟，GPU构建30-60分钟
```

### 8. 验证构建产物
```powershell
# 检查Go二进制
dir susan.exe

# 检查原生库
dir build\lib\ollama

# 应该看到llama-server.exe和相关库文件
```

## 运行时验证

### 9. 处理本地Ollama服务冲突
如果本地已安装Ollama，需要先停止其服务或使用不同端口：

**方法1：停止本地Ollama服务（推荐）**
```powershell
# 停止Ollama服务
taskkill /IM ollama.exe /F
taskkill /IM "ollama app.exe" /F

# 或者通过服务管理器停止
# 打开"服务"管理器，找到Ollama相关服务并停止

# 验证端口已释放
netstat -ano | findstr :11434
# 应该没有输出或显示不同的进程
```

**方法2：使用不同端口启动Susan服务**
```powershell
# 设置环境变量指定不同端口
$env:OLLAMA_HOST = "localhost:11435"

# 启动Susan服务
.\susan.exe serve

# 服务将在 http://localhost:11435 运行
```

### 10. 启动Susan服务
```powershell
# 启动Susan服务（使用默认端口11434）
.\susan.exe serve

# 或者使用不同端口（如果方法2）
$env:OLLAMA_HOST = "localhost:11435"
.\susan.exe serve

# 服务将在后台运行，保持此PowerShell窗口打开
```

### 11. 在新的PowerShell窗口测试API
打开新的PowerShell窗口：

**重要提示：PowerShell的curl是Invoke-WebRequest的别名，语法与Linux curl不同。使用以下PowerShell兼容的命令：**

**如果使用默认端口11434：**
```powershell
# 测试健康检查 - 获取模型标签（正确的list端点）
Invoke-WebRequest -Uri "http://localhost:11434/api/tags" -Method GET

# 测试版本信息
Invoke-WebRequest -Uri "http://localhost:11434/api/version" -Method GET

# 测试模型信息
$body = '{"name":"gemma2:9b"}' | ConvertTo-Json
Invoke-WebRequest -Uri "http://localhost:11434/api/show" -Method POST -Body $body -ContentType "application/json"

# 测试聊天接口（注意：先设置变量，然后使用）
$chatBody = @{
    model = "gemma2:9b"
    messages = @(@{role = "user"; content = "Hello Susan"})
    stream = $false
} | ConvertTo-Json -Depth 10
Invoke-WebRequest -Uri "http://localhost:11434/api/chat" -Method POST -Body $chatBody -ContentType "application/json"

# 测试生成接口
$generateBody = @{
    model = "gemma2:9b"
    prompt = "Hello Susan"
    stream = $false
} | ConvertTo-Json -Depth 10
Invoke-WebRequest -Uri "http://localhost:11434/api/generate" -Method POST -Body $generateBody -ContentType "application/json"

# 测试运行中的模型
Invoke-WebRequest -Uri "http://localhost:11434/api/ps" -Method GET

# 测试嵌入
$embedBody = @{
    model = "gemma2:9b"
    input = "Hello Susan"
} | ConvertTo-Json -Depth 10
Invoke-WebRequest -Uri "http://localhost:11434/api/embed" -Method POST -Body $embedBody -ContentType "application/json"
```

**如果使用端口11435：**
```powershell
# 测试健康检查
Invoke-WebRequest -Uri "http://localhost:11435/api/tags" -Method GET

# 测试版本信息
Invoke-WebRequest -Uri "http://localhost:11435/api/version" -Method GET

# 测试模型信息
$body = '{"name":"gemma2:9b"}' | ConvertTo-Json
Invoke-WebRequest -Uri "http://localhost:11435/api/show" -Method POST -Body $body -ContentType "application/json"

# 测试聊天接口
$chatBody = @{
    model = "gemma2:9b"
    messages = @(@{role = "user"; content = "Hello Susan"})
    stream = $false
} | ConvertTo-Json -Depth 10
Invoke-WebRequest -Uri "http://localhost:11435/api/chat" -Method POST -Body $chatBody -ContentType "application/json"

# 测试生成接口
$generateBody = @{
    model = "gemma2:9b"
    prompt = "Hello Susan"
    stream = $false
} | ConvertTo-Json -Depth 10
Invoke-WebRequest -Uri "http://localhost:11435/api/generate" -Method POST -Body $generateBody -ContentType "application/json"

# 测试运行中的模型
Invoke-WebRequest -Uri "http://localhost:11435/api/ps" -Method GET

# 测试嵌入
$embedBody = @{
    model = "gemma2:9b"
    input = "Hello Susan"
} | ConvertTo-Json -Depth 10
Invoke-WebRequest -Uri "http://localhost:11435/api/embed" -Method POST -Body $embedBody -ContentType "application/json"
```

**或者使用真正的curl（如果已安装）：**
```powershell
# 如果安装了真正的curl.exe（不是PowerShell别名）
curl.exe http://localhost:11435/api/tags
curl.exe http://localhost:11435/api/version
curl.exe -X POST http://localhost:11435/api/show -H "Content-Type: application/json" -d '{"name":"gemma2:9b"}'
curl.exe -X POST http://localhost:11435/api/chat -H "Content-Type: application/json" -d '{"model":"gemma2:9b","messages":[{"role":"user","content":"Hello Susan"}],"stream":false}'
curl.exe http://localhost:11435/api/ps
```

**主要API端点说明：**
- `/api/tags` - 获取本地模型列表（正确的list端点，不是 `/api/list`）
- `/api/version` - 获取服务器版本信息
- `/api/show` - 显示模型详细信息
- `/api/chat` - 聊天接口
- `/api/generate` - 文本生成接口
- `/api/ps` - 列出运行中的模型
- `/api/embed` - 生成文本嵌入
- `/api/pull` - 拉取模型（需要模型名称）
- `/api/delete` - 删除模型（需要模型名称）

**注意：**
- 确保使用已安装的模型名称（从 `.\susan.exe list` 获取）
- PowerShell的curl需要使用Invoke-WebRequest语法或curl.exe
- 变量设置后需要立即使用，否则会失效

### 12. 解决API返回空模型列表问题

**问题现象**：`.\susan.exe list` 能显示模型，但 `http://localhost:11435/api/tags` 返回 `{"models":[]}`

**可能原因**：Susan服务使用了不同的配置目录，没有找到原有模型

**诊断步骤**：
```powershell
# 检查Susan配置目录
dir $env:USERPROFILE\.susan

# 检查Ollama配置目录
dir $env:USERPROFILE\.ollama

# 检查环境变量指向
$env:OLLAMA_MODELS
$env:SUSAN_MODELS

# 检查当前进程的环境变量
[System.Environment]::GetEnvironmentVariables()
```

**解决方案1：复制模型到Susan目录**
```powershell
# 如果Ollama目录有模型，复制到Susan目录
if (Test-Path $env:USERPROFILE\.ollama) {
    if (!(Test-Path $env:USERPROFILE\.susan)) {
        New-Item -ItemType Directory -Path $env:USERPROFILE\.susan
    }
    Copy-Item -Recurse -Force $env:USERPROFILE\.ollama\* $env:USERPROFILE\.susan\
}
```

**解决方案2：设置环境变量指向原模型目录**
```powershell
# 设置Susan使用Ollama的模型目录
$env:OLLAMA_MODELS = "$env:USERPROFILE\.ollama"

# 重启Susan服务
# 先停止当前服务（Ctrl+C）
# 然后重新启动
.\susan.exe serve
```

**解决方案3：完整构建包含原生代码**
```powershell
# 如果只是Go构建，可能缺少原生库支持
# 需要完整CMake构建
cmake -B build -G Ninja .
cmake --build build --parallel 8
.\susan.exe serve
```

### 13. 验证确实是Susan服务而不是Ollama
```powershell
# 检查进程名
tasklist | findstr susan
# 应该看到susan.exe进程

# 检查响应头中的品牌信息
curl -I http://localhost:11434/api/tags
# 查看响应头，确认是Susan服务

# 或者检查版本信息
curl http://localhost:11434/api/version
# 确认版本信息正确
```

### 11. 测试CLI命令
```powershell
# 测试模型运行（需要先下载模型）
.\susan.exe run gemma4 "Hello Susan"

# 测试模型列表
.\susan.exe list

# 测试模型信息
.\susan.exe show gemma4
```

## 品牌化验证检查

### 12. 检查二进制文件名
```powershell
# 确认没有ollama.exe残留
dir ollama.exe
# 应该显示"找不到文件"

# 确认susan.exe存在
dir susan.exe
# 应该显示文件信息
```

### 13. 检查配置目录
```powershell
# 检查用户目录
dir $env:USERPROFILE\.susan

# 检查旧目录是否自动迁移
dir $env:USERPROFILE\.ollama

# 检查日志文件（如果运行过服务）
dir $env:USERPROFILE\.susan\logs
```

### 14. 检查环境变量兼容性
```powershell
# 测试OLLAMA_MODELS是否仍然有效
$env:OLLAMA_MODELS = "D:\MyModels"
.\susan.exe serve

# 测试SUSAN_MODELS是否有效
$env:SUSAN_MODELS = "D:\MyModels"
.\susan.exe serve
```

### 15. 检查用户可见输出
```powershell
# 检查帮助信息中的品牌名称
.\susan.exe --help | Select-String "ollama"
# 应该没有匹配结果

# 检查错误消息中的品牌名称
.\susan.exe nonexistent-command
# 错误消息中应该显示"Susan"而不是"Ollama"
```

## 故障排除

### 常见问题1: CMake配置失败
```powershell
# 确保Visual Studio环境变量已设置
# 在"Developer Command Prompt for VS 2022"中运行构建

# 或者在PowerShell中手动设置
$env:PATH = "C:\Program Files\Microsoft Visual Studio\2022\Community\VC\Tools\MSVC\*\bin\Hostx64\x64;$env:PATH"
```

### 常见问题2: Go构建失败
```powershell
# 清理Go缓存
go clean -cache

# 重新下载依赖
go mod download

# 重新构建
go build -o susan.exe .
```

### 常见问题3: 原生代码构建失败
```powershell
# 检查C++编译器
where cl

# 如果没有，安装Visual Studio 2022 C++构建工具
# 或使用Visual Studio Installer安装"使用C++的桌面开发"工作负载
```

### 常见问题4: 服务启动失败
```powershell
# 检查端口占用
netstat -ano | findstr :11434

# 如果被占用，终止进程
taskkill /PID <进程ID> /F

# 检查日志
type $env:USERPROFILE\.susan\logs\app.log
```

## 验证清单

完成以下检查项确认品牌化成功：

- [ ] susan.exe文件存在且可运行
- [ ] ollama.exe文件不存在
- [ ] `.\susan.exe --version` 显示正确版本
- [ ] `.\susan.exe --help` 中无"ollama"残留
- [ ] `.\susan.exe serve` 成功启动
- [ ] API响应正常（curl测试通过）
- [ ] 配置目录为 ~/.susan
- [ ] 环境变量OLLAMA_*仍然兼容
- [ ] 错误消息中显示"Susan"品牌
- [ ] 日志文件路径正确

## 下一步

验证通过后，可以继续：
1. 阶段1.2: 修改路径检测逻辑
2. 阶段2: 修改用户可见品牌名称
3. 阶段3: 修改配置目录和环境变量
4. 阶段4: Logo和视觉资源替换

如果遇到构建失败，请记录错误信息并检查上述故障排除步骤。


V4--------------------------------------------------------------------------------------
# Susan品牌化构建验证保姆级步骤

本指南提供Windows环境下Susan品牌化项目的完整构建和验证步骤。

## 前置条件检查

### 1. 检查必需工具
打开PowerShell，依次检查以下工具是否安装：

```powershell
# 检查Go版本
go version
# 应该显示: go version go1.26.x windows/amd64

# 检查CMake版本
cmake --version
# 应该显示: cmake version 3.24.x 或更高

# 检查Visual Studio构建工具
where cl
# 应该显示cl.exe路径，如果没有需要安装Visual Studio 2022 C++构建工具

# 检查Ninja（推荐）
where ninja
# 应该显示ninja路径，如果没有可以从GitHub下载
```

### 2. 如果缺少工具，安装指南
- **Go**: 访问 https://go.dev/dl 下载Windows安装包
- **CMake**: 访问 https://cmake.org/download/ 下载Windows安装包
- **Visual Studio**: 安装Visual Studio 2022 Community，选择"使用C++的桌面开发"工作负载
- **Ninja**: 从 https://github.com/ninja-build/ninja/releases 下载ninja.exe，放到PATH路径中

## 快速Go构建验证（推荐先做）

### 3. 快速构建Go二进制
在项目根目录 `d:\projects\susanAssist\susanPlatform\Susan` 打开PowerShell：

```powershell
# 进入项目目录
cd d:\projects\susanAssist\susanPlatform\Susan

# 快速构建（仅Go代码，不包含C++原生代码）
go build -o susan.exe .

# 验证构建产物
dir susan.exe
# 应该看到susan.exe文件
```

### 4. 测试基本功能
```powershell
# 测试版本信息
.\susan.exe --version
# 应该显示版本号

# 测试帮助信息
.\susan.exe --help
# 应该显示帮助文本，检查是否还有"ollama"残留

# 测试serve命令（需要先有原生payload）
# .\susan.exe serve
```

## 完整CMake构建验证

### 5. 清理旧的构建产物
```powershell
# 清理之前的构建
if (Test-Path build) { Remove-Item -Recurse -Force build }
if (Test-Path dist) { Remove-Item -Recurse -Force dist }
```

### 6. 配置CMake项目
```powershell
# 基础配置（CPU only）
cmake -B build -G Ninja .

# 如果有NVIDIA GPU，使用CUDA后端
# cmake -B build -G Ninja -DOLLAMA_LLAMA_BACKENDS=cuda_v13 -DCMAKE_CUDA_ARCHITECTURES=native

# 如果有AMD GPU，使用ROCm后端
# cmake -B build -G Ninja -DOLLAMA_LLAMA_BACKENDS=rocm_v7_2
```

### 7. 构建项目
```powershell
# 并行构建（使用8个线程）
cmake --build build --parallel 8

# 预计时间：CPU构建10-30分钟，GPU构建30-60分钟
```

### 8. 验证构建产物
```powershell
# 检查Go二进制
dir susan.exe

# 检查原生库
dir build\lib\ollama

# 应该看到llama-server.exe和相关库文件
```

## 运行时验证

### 9. 处理本地Ollama服务冲突
如果本地已安装Ollama，需要先停止其服务或使用不同端口：

**方法1：停止本地Ollama服务（推荐）**
```powershell
# 停止Ollama服务
taskkill /IM ollama.exe /F
taskkill /IM "ollama app.exe" /F

# 或者通过服务管理器停止
# 打开"服务"管理器，找到Ollama相关服务并停止

# 验证端口已释放
netstat -ano | findstr :11434
# 应该没有输出或显示不同的进程
```

**方法2：使用不同端口启动Susan服务**
```powershell
# 设置环境变量指定不同端口
$env:OLLAMA_HOST = "localhost:11435"

# 启动Susan服务
.\susan.exe serve

# 服务将在 http://localhost:11435 运行
```

### 10. 启动Susan服务
```powershell
# 启动Susan服务（使用默认端口11434）
.\susan.exe serve

# 或者使用不同端口（如果方法2）
$env:OLLAMA_HOST = "localhost:11435"
.\susan.exe serve

# 服务将在后台运行，保持此PowerShell窗口打开
```

### 11. 在新的PowerShell窗口测试API
打开新的PowerShell窗口：

**重要提示：PowerShell的curl是Invoke-WebRequest的别名，语法与Linux curl不同。使用以下PowerShell兼容的命令：**

**如果使用默认端口11434：**
```powershell
# 测试健康检查 - 获取模型标签
Invoke-WebRequest -Uri "http://localhost:11434/api/tags" -Method GET

# 测试模型信息
$body = '{"name":"gemma2:9b"} | ConvertTo-Json
Invoke-WebRequest -Uri "http://localhost:11434/api/show" -Method POST -Body $body -ContentType "application/json"

# 测试聊天接口
$chatBody = @{
    model = "gemma2:9b"
    messages = @(@{role = "user"; content = "Hello Susan"})
    stream = $false
} | ConvertTo-Json -Depth 10
Invoke-WebRequest -Uri "http://localhost:11434/api/chat" -Method POST -Body $chatBody -ContentType "application/json"
```

**如果使用端口11435：**
```powershell
# 测试健康检查
Invoke-WebRequest -Uri "http://localhost:11435/api/tags" -Method GET

# 测试模型信息
$body = '{"name":"gemma2:9b"} | ConvertTo-Json
Invoke-WebRequest -Uri "http://localhost:11435/api/show" -Method POST -Body $body -ContentType "application/json"

# 测试聊天接口
$chatBody = @{
    model = "gemma2:9b"
    messages = @(@{role = "user"; content = "Hello Susan"})
    stream = $false
} | ConvertTo-Json -Depth 10
Invoke-WebRequest -Uri "http://localhost:11435/api/chat" -Method POST -Body $chatBody -ContentType "application/json"
```

**或者使用真正的curl（如果已安装）：**
```powershell
# 如果安装了真正的curl.exe（不是PowerShell别名）
curl.exe http://localhost:11435/api/tags
curl.exe -X POST http://localhost:11435/api/show -H "Content-Type: application/json" -d '{"name":"gemma2:9b"}'
```

**注意：**
- `/api/tags` 是正确的端点，用于获取模型列表（不是 `/api/list`）
- `/api/show` 用于显示模型详细信息
- 确保使用已安装的模型名称（从 `.\susan.exe list` 获取）
- PowerShell的curl需要使用Invoke-WebRequest语法或curl.exe

### 12. 验证确实是Susan服务而不是Ollama
```powershell
# 检查进程名
tasklist | findstr susan
# 应该看到susan.exe进程

# 检查响应头中的品牌信息
curl -I http://localhost:11434/api/tags
# 查看响应头，确认是Susan服务

# 或者检查版本信息
curl http://localhost:11434/api/version
# 确认版本信息正确
```

### 11. 测试CLI命令
```powershell
# 测试模型运行（需要先下载模型）
.\susan.exe run gemma4 "Hello Susan"

# 测试模型列表
.\susan.exe list

# 测试模型信息
.\susan.exe show gemma4
```

## 品牌化验证检查

### 12. 检查二进制文件名
```powershell
# 确认没有ollama.exe残留
dir ollama.exe
# 应该显示"找不到文件"

# 确认susan.exe存在
dir susan.exe
# 应该显示文件信息
```

### 13. 检查配置目录
```powershell
# 检查用户目录
dir $env:USERPROFILE\.susan

# 检查旧目录是否自动迁移
dir $env:USERPROFILE\.ollama

# 检查日志文件（如果运行过服务）
dir $env:USERPROFILE\.susan\logs
```

### 14. 检查环境变量兼容性
```powershell
# 测试OLLAMA_MODELS是否仍然有效
$env:OLLAMA_MODELS = "D:\MyModels"
.\susan.exe serve

# 测试SUSAN_MODELS是否有效
$env:SUSAN_MODELS = "D:\MyModels"
.\susan.exe serve
```

### 15. 检查用户可见输出
```powershell
# 检查帮助信息中的品牌名称
.\susan.exe --help | Select-String "ollama"
# 应该没有匹配结果

# 检查错误消息中的品牌名称
.\susan.exe nonexistent-command
# 错误消息中应该显示"Susan"而不是"Ollama"
```

## 故障排除

### 常见问题1: CMake配置失败
```powershell
# 确保Visual Studio环境变量已设置
# 在"Developer Command Prompt for VS 2022"中运行构建

# 或者在PowerShell中手动设置
$env:PATH = "C:\Program Files\Microsoft Visual Studio\2022\Community\VC\Tools\MSVC\*\bin\Hostx64\x64;$env:PATH"
```

### 常见问题2: Go构建失败
```powershell
# 清理Go缓存
go clean -cache

# 重新下载依赖
go mod download

# 重新构建
go build -o susan.exe .
```

### 常见问题3: 原生代码构建失败
```powershell
# 检查C++编译器
where cl

# 如果没有，安装Visual Studio 2022 C++构建工具
# 或使用Visual Studio Installer安装"使用C++的桌面开发"工作负载
```

### 常见问题4: 服务启动失败
```powershell
# 检查端口占用
netstat -ano | findstr :11434

# 如果被占用，终止进程
taskkill /PID <进程ID> /F

# 检查日志
type $env:USERPROFILE\.susan\logs\app.log
```

## 验证清单

完成以下检查项确认品牌化成功：

- [ ] susan.exe文件存在且可运行
- [ ] ollama.exe文件不存在
- [ ] `.\susan.exe --version` 显示正确版本
- [ ] `.\susan.exe --help` 中无"ollama"残留
- [ ] `.\susan.exe serve` 成功启动
- [ ] API响应正常（curl测试通过）
- [ ] 配置目录为 ~/.susan
- [ ] 环境变量OLLAMA_*仍然兼容
- [ ] 错误消息中显示"Susan"品牌
- [ ] 日志文件路径正确

## 下一步

验证通过后，可以继续：
1. 阶段1.2: 修改路径检测逻辑
2. 阶段2: 修改用户可见品牌名称
3. 阶段3: 修改配置目录和环境变量
4. 阶段4: Logo和视觉资源替换

如果遇到构建失败，请记录错误信息并检查上述故障排除步骤。


V3--------------------------------------------------------------------------------------
# Susan品牌化构建验证保姆级步骤

本指南提供Windows环境下Susan品牌化项目的完整构建和验证步骤。

## 前置条件检查

### 1. 检查必需工具
打开PowerShell，依次检查以下工具是否安装：

```powershell
# 检查Go版本
go version
# 应该显示: go version go1.26.x windows/amd64

# 检查CMake版本
cmake --version
# 应该显示: cmake version 3.24.x 或更高

# 检查Visual Studio构建工具
where cl
# 应该显示cl.exe路径，如果没有需要安装Visual Studio 2022 C++构建工具

# 检查Ninja（推荐）
where ninja
# 应该显示ninja路径，如果没有可以从GitHub下载
```

### 2. 如果缺少工具，安装指南
- **Go**: 访问 https://go.dev/dl 下载Windows安装包
- **CMake**: 访问 https://cmake.org/download/ 下载Windows安装包
- **Visual Studio**: 安装Visual Studio 2022 Community，选择"使用C++的桌面开发"工作负载
- **Ninja**: 从 https://github.com/ninja-build/ninja/releases 下载ninja.exe，放到PATH路径中

## 快速Go构建验证（推荐先做）

### 3. 快速构建Go二进制
在项目根目录 `d:\projects\susanAssist\susanPlatform\Susan` 打开PowerShell：

```powershell
# 进入项目目录
cd d:\projects\susanAssist\susanPlatform\Susan

# 快速构建（仅Go代码，不包含C++原生代码）
go build -o susan.exe .

# 验证构建产物
dir susan.exe
# 应该看到susan.exe文件
```

### 4. 测试基本功能
```powershell
# 测试版本信息
.\susan.exe --version
# 应该显示版本号

# 测试帮助信息
.\susan.exe --help
# 应该显示帮助文本，检查是否还有"ollama"残留

# 测试serve命令（需要先有原生payload）
# .\susan.exe serve
```

## 完整CMake构建验证

### 5. 清理旧的构建产物
```powershell
# 清理之前的构建
if (Test-Path build) { Remove-Item -Recurse -Force build }
if (Test-Path dist) { Remove-Item -Recurse -Force dist }
```

### 6. 配置CMake项目
```powershell
# 基础配置（CPU only）
cmake -B build -G Ninja .

# 如果有NVIDIA GPU，使用CUDA后端
# cmake -B build -G Ninja -DOLLAMA_LLAMA_BACKENDS=cuda_v13 -DCMAKE_CUDA_ARCHITECTURES=native

# 如果有AMD GPU，使用ROCm后端
# cmake -B build -G Ninja -DOLLAMA_LLAMA_BACKENDS=rocm_v7_2
```

### 7. 构建项目
```powershell
# 并行构建（使用8个线程）
cmake --build build --parallel 8

# 预计时间：CPU构建10-30分钟，GPU构建30-60分钟
```

### 8. 验证构建产物
```powershell
# 检查Go二进制
dir susan.exe

# 检查原生库
dir build\lib\ollama

# 应该看到llama-server.exe和相关库文件
```

## 运行时验证

### 9. 处理本地Ollama服务冲突
如果本地已安装Ollama，需要先停止其服务或使用不同端口：

**方法1：停止本地Ollama服务（推荐）**
```powershell
# 停止Ollama服务
taskkill /IM ollama.exe /F
taskkill /IM "ollama app.exe" /F

# 或者通过服务管理器停止
# 打开"服务"管理器，找到Ollama相关服务并停止

# 验证端口已释放
netstat -ano | findstr :11434
# 应该没有输出或显示不同的进程
```

**方法2：使用不同端口启动Susan服务**
```powershell
# 设置环境变量指定不同端口
$env:OLLAMA_HOST = "localhost:11435"

# 启动Susan服务
.\susan.exe serve

# 服务将在 http://localhost:11435 运行
```

### 10. 启动Susan服务
```powershell
# 启动Susan服务（使用默认端口11434）
.\susan.exe serve

# 或者使用不同端口（如果方法2）
$env:OLLAMA_HOST = "localhost:11435"
.\susan.exe serve

# 服务将在后台运行，保持此PowerShell窗口打开
```

### 11. 在新的PowerShell窗口测试API
打开新的PowerShell窗口：

**如果使用默认端口11434：**
```powershell
# 测试健康检查 - 获取模型标签（正确的API端点）
curl http://localhost:11434/api/tags

# 测试模型信息（使用正确的API端点）
curl http://localhost:11434/api/show -d '{"name":"gemma2:9b"}'

# 如果有模型，测试聊天接口（使用已存在的模型）
curl http://localhost:11434/api/chat -d '{
  "model": "gemma2:9b",
  "messages": [{"role": "user", "content": "Hello Susan"}],
  "stream": false
}'
```

**如果使用端口11435：**
```powershell
# 测试健康检查
curl http://localhost:11435/api/tags

# 测试模型信息
curl http://localhost:11435/api/show -d '{"name":"gemma2:9b"}'

# 测试聊天接口
curl http://localhost:11435/api/chat -d '{
  "model": "gemma2:9b",
  "messages": [{"role": "user", "content": "Hello Susan"}],
  "stream": false
}'
```

**注意：**
- `/api/tags` 是正确的端点，用于获取模型列表（不是 `/api/list`）
- `/api/show` 用于显示模型详细信息
- 确保使用已安装的模型名称（从 `.\susan.exe list` 获取）

### 12. 验证确实是Susan服务而不是Ollama
```powershell
# 检查进程名
tasklist | findstr susan
# 应该看到susan.exe进程

# 检查响应头中的品牌信息
curl -I http://localhost:11434/api/tags
# 查看响应头，确认是Susan服务

# 或者检查版本信息
curl http://localhost:11434/api/version
# 确认版本信息正确
```

### 11. 测试CLI命令
```powershell
# 测试模型运行（需要先下载模型）
.\susan.exe run gemma4 "Hello Susan"

# 测试模型列表
.\susan.exe list

# 测试模型信息
.\susan.exe show gemma4
```

## 品牌化验证检查

### 12. 检查二进制文件名
```powershell
# 确认没有ollama.exe残留
dir ollama.exe
# 应该显示"找不到文件"

# 确认susan.exe存在
dir susan.exe
# 应该显示文件信息
```

### 13. 检查配置目录
```powershell
# 检查用户目录
dir $env:USERPROFILE\.susan

# 检查旧目录是否自动迁移
dir $env:USERPROFILE\.ollama

# 检查日志文件（如果运行过服务）
dir $env:USERPROFILE\.susan\logs
```

### 14. 检查环境变量兼容性
```powershell
# 测试OLLAMA_MODELS是否仍然有效
$env:OLLAMA_MODELS = "D:\MyModels"
.\susan.exe serve

# 测试SUSAN_MODELS是否有效
$env:SUSAN_MODELS = "D:\MyModels"
.\susan.exe serve
```

### 15. 检查用户可见输出
```powershell
# 检查帮助信息中的品牌名称
.\susan.exe --help | Select-String "ollama"
# 应该没有匹配结果

# 检查错误消息中的品牌名称
.\susan.exe nonexistent-command
# 错误消息中应该显示"Susan"而不是"Ollama"
```

## 故障排除

### 常见问题1: CMake配置失败
```powershell
# 确保Visual Studio环境变量已设置
# 在"Developer Command Prompt for VS 2022"中运行构建

# 或者在PowerShell中手动设置
$env:PATH = "C:\Program Files\Microsoft Visual Studio\2022\Community\VC\Tools\MSVC\*\bin\Hostx64\x64;$env:PATH"
```

### 常见问题2: Go构建失败
```powershell
# 清理Go缓存
go clean -cache

# 重新下载依赖
go mod download

# 重新构建
go build -o susan.exe .
```

### 常见问题3: 原生代码构建失败
```powershell
# 检查C++编译器
where cl

# 如果没有，安装Visual Studio 2022 C++构建工具
# 或使用Visual Studio Installer安装"使用C++的桌面开发"工作负载
```

### 常见问题4: 服务启动失败
```powershell
# 检查端口占用
netstat -ano | findstr :11434

# 如果被占用，终止进程
taskkill /PID <进程ID> /F

# 检查日志
type $env:USERPROFILE\.susan\logs\app.log
```

## 验证清单

完成以下检查项确认品牌化成功：

- [ ] susan.exe文件存在且可运行
- [ ] ollama.exe文件不存在
- [ ] `.\susan.exe --version` 显示正确版本
- [ ] `.\susan.exe --help` 中无"ollama"残留
- [ ] `.\susan.exe serve` 成功启动
- [ ] API响应正常（curl测试通过）
- [ ] 配置目录为 ~/.susan
- [ ] 环境变量OLLAMA_*仍然兼容
- [ ] 错误消息中显示"Susan"品牌
- [ ] 日志文件路径正确

## 下一步

验证通过后，可以继续：
1. 阶段1.2: 修改路径检测逻辑
2. 阶段2: 修改用户可见品牌名称
3. 阶段3: 修改配置目录和环境变量
4. 阶段4: Logo和视觉资源替换

如果遇到构建失败，请记录错误信息并检查上述故障排除步骤。


V2--------------------------------------------------------------------------------------
# Susan品牌化构建验证保姆级步骤

本指南提供Windows环境下Susan品牌化项目的完整构建和验证步骤。

## 前置条件检查

### 1. 检查必需工具
打开PowerShell，依次检查以下工具是否安装：

```powershell
# 检查Go版本
go version
# 应该显示: go version go1.26.x windows/amd64

# 检查CMake版本
cmake --version
# 应该显示: cmake version 3.24.x 或更高

# 检查Visual Studio构建工具
where cl
# 应该显示cl.exe路径，如果没有需要安装Visual Studio 2022 C++构建工具

# 检查Ninja（推荐）
where ninja
# 应该显示ninja路径，如果没有可以从GitHub下载
```

### 2. 如果缺少工具，安装指南
- **Go**: 访问 https://go.dev/dl 下载Windows安装包
- **CMake**: 访问 https://cmake.org/download/ 下载Windows安装包
- **Visual Studio**: 安装Visual Studio 2022 Community，选择"使用C++的桌面开发"工作负载
- **Ninja**: 从 https://github.com/ninja-build/ninja/releases 下载ninja.exe，放到PATH路径中

## 快速Go构建验证（推荐先做）

### 3. 快速构建Go二进制
在项目根目录 `d:\projects\susanAssist\susanPlatform\Susan` 打开PowerShell：

```powershell
# 进入项目目录
cd d:\projects\susanAssist\susanPlatform\Susan

# 快速构建（仅Go代码，不包含C++原生代码）
go build -o susan.exe .

# 验证构建产物
dir susan.exe
# 应该看到susan.exe文件
```

### 4. 测试基本功能
```powershell
# 测试版本信息
.\susan.exe --version
# 应该显示版本号

# 测试帮助信息
.\susan.exe --help
# 应该显示帮助文本，检查是否还有"ollama"残留

# 测试serve命令（需要先有原生payload）
# .\susan.exe serve
```

## 完整CMake构建验证

### 5. 清理旧的构建产物
```powershell
# 清理之前的构建
if (Test-Path build) { Remove-Item -Recurse -Force build }
if (Test-Path dist) { Remove-Item -Recurse -Force dist }
```

### 6. 配置CMake项目
```powershell
# 基础配置（CPU only）
cmake -B build -G Ninja .

# 如果有NVIDIA GPU，使用CUDA后端
# cmake -B build -G Ninja -DOLLAMA_LLAMA_BACKENDS=cuda_v13 -DCMAKE_CUDA_ARCHITECTURES=native

# 如果有AMD GPU，使用ROCm后端
# cmake -B build -G Ninja -DOLLAMA_LLAMA_BACKENDS=rocm_v7_2
```

### 7. 构建项目
```powershell
# 并行构建（使用8个线程）
cmake --build build --parallel 8

# 预计时间：CPU构建10-30分钟，GPU构建30-60分钟
```

### 8. 验证构建产物
```powershell
# 检查Go二进制
dir susan.exe

# 检查原生库
dir build\lib\ollama

# 应该看到llama-server.exe和相关库文件
```

## 运行时验证

### 9. 处理本地Ollama服务冲突
如果本地已安装Ollama，需要先停止其服务或使用不同端口：

**方法1：停止本地Ollama服务（推荐）**
```powershell
# 停止Ollama服务
taskkill /IM ollama.exe /F
taskkill /IM "ollama app.exe" /F

# 或者通过服务管理器停止
# 打开"服务"管理器，找到Ollama相关服务并停止

# 验证端口已释放
netstat -ano | findstr :11434
# 应该没有输出或显示不同的进程
```

**方法2：使用不同端口启动Susan服务**
```powershell
# 设置环境变量指定不同端口
$env:OLLAMA_HOST = "localhost:11435"

# 启动Susan服务
.\susan.exe serve

# 服务将在 http://localhost:11435 运行
```

### 10. 启动Susan服务
```powershell
# 启动Susan服务（使用默认端口11434）
.\susan.exe serve

# 或者使用不同端口（如果方法2）
$env:OLLAMA_HOST = "localhost:11435"
.\susan.exe serve

# 服务将在后台运行，保持此PowerShell窗口打开
```

### 11. 在新的PowerShell窗口测试API
打开新的PowerShell窗口：

**如果使用默认端口11434：**
```powershell
# 测试健康检查
curl http://localhost:11434/api/tags

# 测试模型列表（应该为空或显示已安装的模型）
curl http://localhost:11434/api/list

# 如果有模型，测试聊天接口
curl http://localhost:11434/api/chat -d '{
  "model": "gemma4",
  "messages": [{"role": "user", "content": "Hello"}],
  "stream": false
}'
```

**如果使用端口11435：**
```powershell
# 测试健康检查
curl http://localhost:11435/api/tags

# 测试模型列表
curl http://localhost:11435/api/list

# 测试聊天接口
curl http://localhost:11435/api/chat -d '{
  "model": "gemma4",
  "messages": [{"role": "user", "content": "Hello"}],
  "stream": false
}'
```

### 12. 验证确实是Susan服务而不是Ollama
```powershell
# 检查进程名
tasklist | findstr susan
# 应该看到susan.exe进程

# 检查响应头中的品牌信息
curl -I http://localhost:11434/api/tags
# 查看响应头，确认是Susan服务

# 或者检查版本信息
curl http://localhost:11434/api/version
# 确认版本信息正确
```

### 11. 测试CLI命令
```powershell
# 测试模型运行（需要先下载模型）
.\susan.exe run gemma4 "Hello Susan"

# 测试模型列表
.\susan.exe list

# 测试模型信息
.\susan.exe show gemma4
```

## 品牌化验证检查

### 12. 检查二进制文件名
```powershell
# 确认没有ollama.exe残留
dir ollama.exe
# 应该显示"找不到文件"

# 确认susan.exe存在
dir susan.exe
# 应该显示文件信息
```

### 13. 检查配置目录
```powershell
# 检查用户目录
dir $env:USERPROFILE\.susan

# 检查旧目录是否自动迁移
dir $env:USERPROFILE\.ollama

# 检查日志文件（如果运行过服务）
dir $env:USERPROFILE\.susan\logs
```

### 14. 检查环境变量兼容性
```powershell
# 测试OLLAMA_MODELS是否仍然有效
$env:OLLAMA_MODELS = "D:\MyModels"
.\susan.exe serve

# 测试SUSAN_MODELS是否有效
$env:SUSAN_MODELS = "D:\MyModels"
.\susan.exe serve
```

### 15. 检查用户可见输出
```powershell
# 检查帮助信息中的品牌名称
.\susan.exe --help | Select-String "ollama"
# 应该没有匹配结果

# 检查错误消息中的品牌名称
.\susan.exe nonexistent-command
# 错误消息中应该显示"Susan"而不是"Ollama"
```

## 故障排除

### 常见问题1: CMake配置失败
```powershell
# 确保Visual Studio环境变量已设置
# 在"Developer Command Prompt for VS 2022"中运行构建

# 或者在PowerShell中手动设置
$env:PATH = "C:\Program Files\Microsoft Visual Studio\2022\Community\VC\Tools\MSVC\*\bin\Hostx64\x64;$env:PATH"
```

### 常见问题2: Go构建失败
```powershell
# 清理Go缓存
go clean -cache

# 重新下载依赖
go mod download

# 重新构建
go build -o susan.exe .
```

### 常见问题3: 原生代码构建失败
```powershell
# 检查C++编译器
where cl

# 如果没有，安装Visual Studio 2022 C++构建工具
# 或使用Visual Studio Installer安装"使用C++的桌面开发"工作负载
```

### 常见问题4: 服务启动失败
```powershell
# 检查端口占用
netstat -ano | findstr :11434

# 如果被占用，终止进程
taskkill /PID <进程ID> /F

# 检查日志
type $env:USERPROFILE\.susan\logs\app.log
```

## 验证清单

完成以下检查项确认品牌化成功：

- [ ] susan.exe文件存在且可运行
- [ ] ollama.exe文件不存在
- [ ] `.\susan.exe --version` 显示正确版本
- [ ] `.\susan.exe --help` 中无"ollama"残留
- [ ] `.\susan.exe serve` 成功启动
- [ ] API响应正常（curl测试通过）
- [ ] 配置目录为 ~/.susan
- [ ] 环境变量OLLAMA_*仍然兼容
- [ ] 错误消息中显示"Susan"品牌
- [ ] 日志文件路径正确

## 下一步

验证通过后，可以继续：
1. 阶段1.2: 修改路径检测逻辑
2. 阶段2: 修改用户可见品牌名称
3. 阶段3: 修改配置目录和环境变量
4. 阶段4: Logo和视觉资源替换

如果遇到构建失败，请记录错误信息并检查上述故障排除步骤。





V1--------------------------------------------------------------------------------------
# Susan品牌化构建验证保姆级步骤

本指南提供Windows环境下Susan品牌化项目的完整构建和验证步骤。

## 前置条件检查

### 1. 检查必需工具
打开PowerShell，依次检查以下工具是否安装：

```powershell
# 检查Go版本
go version
# 应该显示: go version go1.26.x windows/amd64

# 检查CMake版本
cmake --version
# 应该显示: cmake version 3.24.x 或更高

# 检查Visual Studio构建工具
where cl
# 应该显示cl.exe路径，如果没有需要安装Visual Studio 2022 C++构建工具

# 检查Ninja（推荐）
where ninja
# 应该显示ninja路径，如果没有可以从GitHub下载
```

### 2. 如果缺少工具，安装指南
- **Go**: 访问 https://go.dev/dl 下载Windows安装包
- **CMake**: 访问 https://cmake.org/download/ 下载Windows安装包
- **Visual Studio**: 安装Visual Studio 2022 Community，选择"使用C++的桌面开发"工作负载
- **Ninja**: 从 https://github.com/ninja-build/ninja/releases 下载ninja.exe，放到PATH路径中

## 快速Go构建验证（推荐先做）

### 3. 快速构建Go二进制
在项目根目录 `d:\projects\susanAssist\susanPlatform\Susan` 打开PowerShell：

```powershell
# 进入项目目录
cd d:\projects\susanAssist\susanPlatform\Susan

# 快速构建（仅Go代码，不包含C++原生代码）
go build -o susan.exe .

# 验证构建产物
dir susan.exe
# 应该看到susan.exe文件
```

### 4. 测试基本功能
```powershell
# 测试版本信息
.\susan.exe --version
# 应该显示版本号

# 测试帮助信息
.\susan.exe --help
# 应该显示帮助文本，检查是否还有"ollama"残留

# 测试serve命令（需要先有原生payload）
# .\susan.exe serve
```

## 完整CMake构建验证

### 5. 清理旧的构建产物
```powershell
# 清理之前的构建
if (Test-Path build) { Remove-Item -Recurse -Force build }
if (Test-Path dist) { Remove-Item -Recurse -Force dist }
```

### 6. 配置CMake项目
```powershell
# 基础配置（CPU only）
cmake -B build -G Ninja .

# 如果有NVIDIA GPU，使用CUDA后端
# cmake -B build -G Ninja -DOLLAMA_LLAMA_BACKENDS=cuda_v13 -DCMAKE_CUDA_ARCHITECTURES=native

# 如果有AMD GPU，使用ROCm后端
# cmake -B build -G Ninja -DOLLAMA_LLAMA_BACKENDS=rocm_v7_2
```

### 7. 构建项目
```powershell
# 并行构建（使用8个线程）
cmake --build build --parallel 8

# 预计时间：CPU构建10-30分钟，GPU构建30-60分钟
```

### 8. 验证构建产物
```powershell
# 检查Go二进制
dir susan.exe

# 检查原生库
dir build\lib\ollama

# 应该看到llama-server.exe和相关库文件
```

## 运行时验证

### 9. 启动服务
```powershell
# 启动Susan服务
.\susan.exe serve

# 服务将在后台运行，监听 http://localhost:11434
# 保持此PowerShell窗口打开
```

### 10. 在新的PowerShell窗口测试API
打开新的PowerShell窗口：

```powershell
# 测试健康检查
curl http://localhost:11434/api/tags

# 测试模型列表（应该为空或显示已安装的模型）
curl http://localhost:11434/api/list

# 如果有模型，测试聊天接口
curl http://localhost:11434/api/chat -d '{
  "model": "gemma4",
  "messages": [{"role": "user", "content": "Hello"}],
  "stream": false
}'
```

### 11. 测试CLI命令
```powershell
# 测试模型运行（需要先下载模型）
.\susan.exe run gemma4 "Hello Susan"

# 测试模型列表
.\susan.exe list

# 测试模型信息
.\susan.exe show gemma4
```

## 品牌化验证检查

### 12. 检查二进制文件名
```powershell
# 确认没有ollama.exe残留
dir ollama.exe
# 应该显示"找不到文件"

# 确认susan.exe存在
dir susan.exe
# 应该显示文件信息
```

### 13. 检查配置目录
```powershell
# 检查用户目录
dir $env:USERPROFILE\.susan

# 检查旧目录是否自动迁移
dir $env:USERPROFILE\.ollama

# 检查日志文件（如果运行过服务）
dir $env:USERPROFILE\.susan\logs
```

### 14. 检查环境变量兼容性
```powershell
# 测试OLLAMA_MODELS是否仍然有效
$env:OLLAMA_MODELS = "D:\MyModels"
.\susan.exe serve

# 测试SUSAN_MODELS是否有效
$env:SUSAN_MODELS = "D:\MyModels"
.\susan.exe serve
```

### 15. 检查用户可见输出
```powershell
# 检查帮助信息中的品牌名称
.\susan.exe --help | Select-String "ollama"
# 应该没有匹配结果

# 检查错误消息中的品牌名称
.\susan.exe nonexistent-command
# 错误消息中应该显示"Susan"而不是"Ollama"
```

## 故障排除

### 常见问题1: CMake配置失败
```powershell
# 确保Visual Studio环境变量已设置
# 在"Developer Command Prompt for VS 2022"中运行构建

# 或者在PowerShell中手动设置
$env:PATH = "C:\Program Files\Microsoft Visual Studio\2022\Community\VC\Tools\MSVC\*\bin\Hostx64\x64;$env:PATH"
```

### 常见问题2: Go构建失败
```powershell
# 清理Go缓存
go clean -cache

# 重新下载依赖
go mod download

# 重新构建
go build -o susan.exe .
```

### 常见问题3: 原生代码构建失败
```powershell
# 检查C++编译器
where cl

# 如果没有，安装Visual Studio 2022 C++构建工具
# 或使用Visual Studio Installer安装"使用C++的桌面开发"工作负载
```

### 常见问题4: 服务启动失败
```powershell
# 检查端口占用
netstat -ano | findstr :11434

# 如果被占用，终止进程
taskkill /PID <进程ID> /F

# 检查日志
type $env:USERPROFILE\.susan\logs\app.log
```

## 验证清单

完成以下检查项确认品牌化成功：

- [ ] susan.exe文件存在且可运行
- [ ] ollama.exe文件不存在
- [ ] `.\susan.exe --version` 显示正确版本
- [ ] `.\susan.exe --help` 中无"ollama"残留
- [ ] `.\susan.exe serve` 成功启动
- [ ] API响应正常（curl测试通过）
- [ ] 配置目录为 ~/.susan
- [ ] 环境变量OLLAMA_*仍然兼容
- [ ] 错误消息中显示"Susan"品牌
- [ ] 日志文件路径正确

## 下一步

验证通过后，可以继续：
1. 阶段1.2: 修改路径检测逻辑
2. 阶段2: 修改用户可见品牌名称
3. 阶段3: 修改配置目录和环境变量
4. 阶段4: Logo和视觉资源替换

如果遇到构建失败，请记录错误信息并检查上述故障排除步骤。
