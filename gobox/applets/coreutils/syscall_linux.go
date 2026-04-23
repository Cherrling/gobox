//go:build linux

package coreutils

import "syscall"

func syscallSync() {
	syscall.Sync()
}
