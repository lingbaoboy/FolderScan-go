//go:build darwin

package main

import (
	"io/fs"
	"syscall"
	"time"
)

// getCreateTimePlatform 在 macOS 系统下获取文件的创建时间。
func getCreateTimePlatform(fi fs.FileInfo) time.Time {
	if stat, ok := fi.Sys().(*syscall.Stat_t); ok {
		// 在 macOS (darwin) 上，st_birthtimespec 直接提供了文件的创建时间
		return time.Unix(stat.Birthtimespec.Sec, stat.Birthtimespec.Nsec)
	}
	// 如果无法获取，则回退到使用修改时间
	return fi.ModTime()
}
