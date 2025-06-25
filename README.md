# FolederScan-go

1、安装go环境
2、安装fyne
go install fyne.io/tools/cmd/fyne@latest
3、打包
fyne package -os windows -release
fyne package -os darwin -release

压缩
upx --best FolderScan.exe

go常用命令
go version
go mod init my-go-app
go mod tidy
