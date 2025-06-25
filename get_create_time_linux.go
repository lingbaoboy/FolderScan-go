//go:build linux

package main

import (
	"io/fs"
	"syscall"
	"time"
)

// getCreateTimePlatform 在 Linux 系统下获取文件的创建时间。
func getCreateTimePlatform(fi fs.FileInfo) time.Time {
	if stat, ok := fi.Sys().(*syscall.Stat_t); ok {
		// 在 Linux 上，没有统一的创建时间字段。
		// 我们使用 st_ctim (状态改变时间) 作为创建时间的近似值。
		return time.Unix(stat.Ctim.Sec, stat.Ctim.Nsec)
	}
	// 如果无法获取，则回退到使用修改时间
	return fi.ModTime()
}
