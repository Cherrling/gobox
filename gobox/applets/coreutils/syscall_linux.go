//go:build linux

package coreutils

import "syscall"

func syscallSync() {
	syscall.Sync()
}

func lchown(path string, uid, gid int) error {
	return syscall.Lchown(path, uid, gid)
}
