# FolederScan-go

：

# Go环境安装与使用指南

## 1. 安装Go环境
请根据您的操作系统下载并安装Go语言环境。可以访问[Go官方网站](https://golang.org/dl/)获取最新版本。

## 2. 安装Fyne
在终端中运行以下命令以安装Fyne工具：
```bash
go install fyne.io/tools/cmd/fyne@latest
```

## 3. 打包应用
使用Fyne打包您的应用程序。根据目标操作系统运行以下命令：

### 打包为Windows应用
```bash
fyne package -os windows -release
```

### 打包为macOS应用
```bash
fyne package -os darwin -release
```

## 4. 压缩可执行文件
使用UPX压缩生成的可执行文件，以减小文件大小：
```bash
upx --best FolderScan.exe
```

## 5. Go常用命令
以下是一些常用的Go命令：

- 查看Go版本：
  ```bash
  go version
  ```

- 初始化Go模块：
  ```bash
  go mod init my-go-app
  ```

- 清理模块依赖：
  ```bash
  go mod tidy
  ```

通过以上步骤，您可以成功安装Go环境、使用Fyne进行应用打包，并掌握一些常用的Go命令。
