# Windows和macOS平台Susan项目保姆级构建指南

本指南为首次构建者提供Windows和macOS平台下Susan项目的完整构建步骤，包含工具安装、构建配置、运行测试和故障排除。

## Windows平台构建指南

### 第一阶段：前置工具安装和检查

#### 1.1 检查系统环境
打开PowerShell（以管理员身份运行），执行以下命令检查系统基本信息：

```powershell
# 检查系统架构
$env:PROCESSOR_ARCHITECTURE
# 应该显示 AMD64（Intel/AMD）或 ARM64

# 检查Windows版本
systeminfo | findstr /B /C:"OS Name" /C:"OS Version"
```

#### 1.2 安装Go语言
1. 访问 https://go.dev/dl/
2. 下载Windows安装包（go1.x.x.windows-amd64.msi）
3. 双击安装包，按默认设置完成安装
4. 验证安装：

```powershell
# 打开新的PowerShell窗口
go version
# 应该显示：go version go1.x.x windows/amd64
```

#### 1.3 安装CMake
1. 访问 https://cmake.org/download/
2. 下载Windows x64 Installer（cmake-3.2x.x-windows-x86_64.msi）
3. 运行安装程序，**重要**：勾选"Add CMake to the system PATH"
4. 验证安装：

```powershell
# 打开新的PowerShell窗口
cmake --version
# 应该显示：cmake version 3.2x.x
```

#### 1.4 安装Visual Studio 2022
1. 访问 https://visualstudio.microsoft.com/downloads/
2. 下载Visual Studio 2022 Community（免费）
3. 运行安装程序，选择"使用C++的桌面开发"工作负载
4. 确保以下组件被选中：
   - MSVC v143 - VS 2022 C++ x64/x86生成工具
   - Windows 10 SDK（或Windows 11 SDK）
5. 完成安装（可能需要30分钟-1小时）
6. 验证安装：

```powershell
# 搜索并打开"Developer Command Prompt for VS 2022"
# 在其中执行：
cl
# 应该显示Microsoft C/C++编译器版本信息
```

#### 1.5 安装Ninja（推荐但非必需）
1. 访问 https://github.com/ninja-build/ninja/releases
2. 下载ninja-win.zip
3. 解压到某个目录，例如：C:\ninja
4. 将C:\ninja添加到系统PATH：
   - 右键"此电脑" → 属性 → 高级系统设置 → 环境变量
   - 在"系统变量"中找到Path，点击编辑
   - 新建条目，添加：C:\ninja
5. 验证安装：

```powershell
# 打开新的PowerShell窗口
ninja --version
# 应该显示ninja版本号
```

#### 1.6 可选：GPU加速支持
**如果需要NVIDIA GPU支持：**
1. 访问 https://developer.nvidia.com/cuda-downloads
2. 下载CUDA 13.0或更高版本
3. 运行安装程序，按默认设置安装
4. 验证安装：

```powershell
nvcc --version
# 应该显示CUDA版本信息
```

**如果需要AMD GPU支持：**
1. 访问 https://rocm.docs.amd.com/en/latest/
2. 下载并安装ROCm 7.x
3. 设置环境变量：

```powershell
$env:HIP_PATH = "C:\Program Files\AMD\ROCm\7.x"
```

**如果需要Vulkan支持（适用于AMD/Intel GPU）：**
1. 访问 https://vulkan.lunarg.com/sdk/home
2. 下载Vulkan SDK
3. 安装后设置环境变量：

```powershell
$env:VULKAN_SDK = "C:\VulkanSDK\x.x.x"
```

### 第二阶段：项目构建

#### 2.1 准备项目目录
```powershell
# 进入项目目录
cd d:\projects\susanAssist\susanPlatform\Susan

# 检查项目结构
dir
# 应该看到CMakeLists.txt、go.mod等文件
```

#### 2.2 快速Go构建（推荐首次尝试）
这是最简单的构建方式，仅编译Go代码，不包含C++原生代码：

```powershell
# 清理旧的构建产物
if (Test-Path susan.exe) { Remove-Item susan.exe }

# 快速构建
go build -o susan.exe .

# 验证构建产物
dir susan.exe
# 应该看到susan.exe文件

# 测试基本功能
.\susan.exe --version
.\susan.exe --help
```

#### 2.3 完整CMake构建（包含原生代码）
**方式1：使用Developer Command Prompt（推荐）**

1. 搜索并打开"Developer Command Prompt for VS 2022"
2. 导航到项目目录：

```cmd
cd /d d:\projects\susanAssist\susanPlatform\Susan
```

3. 配置CMake项目：

```cmd
# CPU only构建（最简单）
cmake -B build -G Ninja .

# 如果有NVIDIA GPU，使用CUDA后端
cmake -B build -G Ninja -DOLLAMA_LLAMA_BACKENDS=cuda_v13 -DCMAKE_CUDA_ARCHITECTURES=native

# 如果有AMD GPU，使用ROCm后端
cmake -B build -G Ninja -DOLLAMA_LLAMA_BACKENDS=rocm_v7_2

# 如果有Vulkan支持
cmake -B build -G Ninja -DOLLAMA_LLAMA_BACKENDS=vulkan
```

4. 构建项目：

```cmd
cmake --build build --parallel 8
```

**方式2：使用PowerShell**

```powershell
# 进入项目目录
cd d:\projects\susanAssist\susanPlatform\Susan

# 导入Visual Studio环境模块
$vsInstall = "C:\Program Files\Microsoft Visual Studio\2022\Community"
if (Test-Path "$vsInstall\Common7\Tools\Microsoft.VisualStudio.DevShell.dll") {
    Import-Module "$vsInstall\Common7\Tools\Microsoft.VisualStudio.DevShell.dll"
    Enter-VsDevShell -VsInstallPath $vsInstall -SkipAutomaticLocation -DevCmdArguments "-arch=x64 -host_arch=x64"
}

# 配置和构建
cmake -B build -G Ninja .
cmake --build build --parallel 8
```

#### 2.4 使用官方构建脚本（最全面）
项目提供了官方的Windows构建脚本，可以自动检测环境并构建多个变体：

```powershell
# 在项目目录执行
.\scripts\build_windows.ps1

# 脚本会自动：
# 1. 检测Visual Studio安装
# 2. 检测CUDA/ROCm/Vulkan
# 3. 构建CPU版本
# 4. 构建GPU版本（如果检测到）
# 5. 生成安装包
```

### 第三阶段：运行和测试

#### 3.1 启动Susan服务
```powershell
# 基本启动
.\susan.exe serve

# 使用不同端口（避免与已安装的Ollama冲突）
$env:OLLAMA_HOST = "localhost:11435"
.\susan.exe serve
```

#### 3.2 测试API接口
打开新的PowerShell窗口：

```powershell
# 测试版本信息
Invoke-WebRequest -Uri "http://localhost:11434/api/version" -Method GET

# 测试模型列表
Invoke-WebRequest -Uri "http://localhost:11434/api/tags" -Method GET

# 测试健康检查
Invoke-WebRequest -Uri "http://localhost:11434/api/ps" -Method GET
```

#### 3.3 运行单元测试
```powershell
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./api
go test ./server
```

### 第四阶段：故障排除

#### 常见问题1：CMake配置失败
**症状**：CMake报错找不到编译器
**解决**：
```powershell
# 确保在Developer Command Prompt中运行
# 或手动设置环境变量
$env:PATH = "C:\Program Files\Microsoft Visual Studio\2022\Community\VC\Tools\MSVC\*\bin\Hostx64\x64;$env:PATH"
```

#### 常见问题2：Go构建失败
**症状**：Go编译错误或依赖问题
**解决**：
```powershell
# 清理Go缓存
go clean -cache

# 重新下载依赖
go mod download

# 重新构建
go build -o susan.exe .
```

#### 常见问题3：端口冲突
**症状**：启动服务时提示端口被占用
**解决**：
```powershell
# 查找占用端口的进程
netstat -ano | findstr :11434

# 终止进程
taskkill /PID <进程ID> /F

# 或使用不同端口
$env:OLLAMA_HOST = "localhost:11435"
```

## macOS平台构建指南

### 第一阶段：前置工具安装和检查

#### 1.1 检查系统环境
打开Terminal，执行以下命令：

```bash
# 检查macOS版本
sw_vers
# 检查系统架构
uname -m
# 应该显示 arm64（Apple Silicon）或 x86_64（Intel）
```

#### 1.2 安装Xcode
1. 从App Store安装Xcode（约10GB）
2. 安装完成后，打开Xcode，接受许可协议
3. 安装命令行工具：

```bash
xcode-select --install
```

4. 验证安装：

```bash
xcodebuild -version
# 应该显示Xcode版本信息
```

#### 1.3 安装Metal工具链（Apple Silicon必需）
```bash
# 下载Metal工具链
xcodebuild -downloadComponent MetalToolchain
```

#### 1.4 安装Homebrew（包管理器）
```bash
# 安装Homebrew
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# 将Homebrew添加到PATH（按照安装完成后的提示操作）
# 通常是：
echo 'eval "$(/opt/homebrew/bin/brew shellenv)"' >> ~/.zprofile
eval "$(/opt/homebrew/bin/brew shellenv)"
```

#### 1.5 安装Go语言
```bash
# 使用Homebrew安装
brew install go

# 验证安装
go version
# 应该显示：go version go1.x.x darwin/arm64 或 darwin/amd64
```

#### 1.6 安装CMake
```bash
# 使用Homebrew安装
brew install cmake

# 验证安装
cmake --version
# 应该显示：cmake version 3.2x.x
```

#### 1.7 安装Node.js和npm（构建macOS应用需要）
```bash
# 使用Homebrew安装
brew install node

# 验证安装
node --version
npm --version
```

### 第二阶段：项目构建

#### 2.1 准备项目目录
```bash
# 进入项目目录
cd /path/to/Susan

# 检查项目结构
ls
# 应该看到CMakeLists.txt、go.mod等文件
```

#### 2.2 快速Go构建
```bash
# 快速构建
go build -o susan .

# 验证构建产物
ls -lh susan
# 应该看到susan可执行文件

# 测试基本功能
./susan --version
./susan --help
```

#### 2.3 完整CMake构建
```bash
# 配置项目（Apple Silicon自动使用Metal加速）
cmake -B build .

# 构建项目
cmake --build build --parallel 8

# 验证构建产物
ls -lh build/lib/ollama/
# 应该看到llama-server等原生库
```

#### 2.4 使用官方构建脚本
项目提供了官方的macOS构建脚本：

```bash
# 基本构建（仅命令行工具）
./scripts/build_darwin.sh build

# 完整构建（包括macOS应用）
./scripts/build_darwin.sh

# 仅打包
./scripts/build_darwin.sh package

# 仅代码签名（需要Apple开发者账号）
./scripts/build_darwin.sh sign

# 仅构建macOS应用
./scripts/build_darwin.sh app
```

### 第三阶段：运行和测试

#### 3.1 启动Susan服务
```bash
# 基本启动
./susan serve

# 使用不同端口
OLLAMA_HOST=localhost:11435 ./susan serve
```

#### 3.2 测试API接口
打开新的Terminal窗口：

```bash
# 测试版本信息
curl http://localhost:11434/api/version

# 测试模型列表
curl http://localhost:11434/api/tags

# 测试健康检查
curl http://localhost:11434/api/ps
```

#### 3.3 运行单元测试
```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./api
go test ./server
```

#### 3.4 测试macOS应用
如果构建了macOS应用：

```bash
# 打开应用
open dist/Susan.app

# 或从Finder中双击Susan.app
```

### 第四阶段：故障排除

#### 常见问题1：Xcode命令行工具未安装
**症状**：xcodebuild命令找不到
**解决**：
```bash
xcode-select --install
# 按照提示完成安装
```

#### 常见问题2：Metal工具链未安装
**症状**：Metal相关编译错误
**解决**：
```bash
xcodebuild -downloadComponent MetalToolchain
```

#### 常见问题3：权限问题
**症状**：无法写入某些目录
**解决**：
```bash
# 修复文件权限
sudo chown -R $(whoami) /path/to/Susan
```

#### 常见问题4：Node.js依赖安装失败
**症状**：npm install失败
**解决**：
```bash
# 清理npm缓存
npm cache clean --force

# 重新安装
cd app/ui/app
npm install
```

## 通用验证清单

完成构建后，使用以下清单验证构建成功：

### Windows验证清单
- [ ] susan.exe文件存在且可运行
- [ ] `.\susan.exe --version` 显示正确版本
- [ ] `.\susan.exe --help` 显示帮助信息
- [ ] `.\susan.exe serve` 成功启动服务
- [ ] API接口响应正常
- [ ] 单元测试通过（`go test ./...`）

### macOS验证清单
- [ ] susan可执行文件存在且可运行
- [ ] `./susan --version` 显示正确版本
- [ ] `./susan --help` 显示帮助信息
- [ ] `./susan serve` 成功启动服务
- [ ] API接口响应正常
- [ ] 单元测试通过（`go test ./...`）
- [ ] 如果构建了macOS应用，Susan.app可以正常启动

## 下一步建议

1. **首次构建建议**：先尝试快速Go构建，确保环境配置正确
2. **完整构建**：快速构建成功后，再尝试完整CMake构建
3. **GPU支持**：只有在需要GPU加速时才配置CUDA/ROCm/Vulkan
4. **测试驱动**：每完成一个阶段就进行测试，确保问题及时发现
5. **文档记录**：记录遇到的问题和解决方案，方便后续参考

## 获取帮助

如果遇到问题：
1. 查看项目文档：`docs/development.md`
2. 查看构建脚本注释：`scripts/build_windows.ps1` 和 `scripts/build_darwin.sh`
3. 检查CMake配置：`build/CMakeCache.txt`
4. 查看构建日志：重新构建并保存输出日志
