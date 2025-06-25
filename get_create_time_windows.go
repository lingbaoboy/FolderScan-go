//go:build windows

package main

import (
	"io/fs"
	"syscall"
	"time"
)

// getCreateTimePlatform 在Windows系统下获取文件的创建时间。
func getCreateTimePlatform(fi fs.FileInfo) time.Time {
	if stat, ok := fi.Sys().(*syscall.Win32FileAttributeData); ok {
		return time.Unix(0, stat.CreationTime.Nanoseconds())
	}
	return fi.ModTime()
}
