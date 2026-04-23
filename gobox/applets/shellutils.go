package applets

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"syscall"
)

func init() {
	Register("sh", AppletFunc(shMain))
	Register("ash", AppletFunc(shMain))
	Register("bash", AppletFunc(shMain))
	Register("hush", AppletFunc(shMain))
	Register("linuxrc", AppletFunc(shMain))
	Register("cttyhack", AppletFunc(cttyhackMain))
}

func shMain(args []string) int {
	// Find a shell to run
	shells := []string{"/bin/sh", "/bin/bash", "/bin/ash", "/bin/dash", "/bin/busybox"}

	cmd := ""
	cmdArgs := []string{}

	if len(args) > 1 && !strings.HasPrefix(args[1], "-") {
		// Run a script file
		cmd = args[1]
		cmdArgs = args[2:]
	} else {
		// Find an available shell
		for _, s := range shells {
			if _, err := os.Stat(s); err == nil {
				// Check if it's gobox/busybox
				cmd = s
				break
			}
		}
		if cmd == "" {
			// Try to use /proc/self/exe
			cmd = "/proc/self/exe"
		}
	}

	if len(args) > 1 && args[1] == "-c" && len(args) > 2 {
		// Run command
		cmdArgs = []string{"-c", args[2]}
	}

	if cmd == "" {
		fmt.Fprintln(os.Stderr, "gobox: sh: no shell found")
		return 1
	}

	// If the shell is ourselves, we need to exec with the right applet
	if cmd == "/proc/self/exe" || cmd == filepath.Base(cmd) {
		fmt.Fprintln(os.Stderr, "gobox: sh: cannot find a shell")
		return 1
	}

	execPath, err := exec.LookPath(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: sh: %s: not found\n", cmd)
		return 1
	}

	allArgs := append([]string{execPath}, cmdArgs...)
	return runExec(execPath, allArgs)
}

func cttyhackMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: cttyhack: missing command")
		return 1
	}
	// Try to get a controlling terminal
	f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		// Try /dev/console
		f, err = os.OpenFile("/dev/console", os.O_RDWR, 0)
		if err != nil {
			fmt.Fprintln(os.Stderr, "gobox: cttyhack: cannot open tty")
			return 1
		}
	}
	defer f.Close()

	// Set controlling terminal
	syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), syscall.TIOCSCTTY, 0)

	// Run the command
	cmd := exec.Command(args[1], args[2:]...)
	cmd.Stdin = f
	cmd.Stdout = f
	cmd.Stderr = f
	return runCommandExit(cmd)
}

func runExec(path string, args []string) int {
	if err := syscall.Exec(path, args, os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "gobox: exec: %v\n", err)
		return 1
	}
	return 0
}
