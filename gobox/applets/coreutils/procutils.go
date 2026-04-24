package coreutils

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gobox/applets"
)

func init() {
	applets.Register("pidof", applets.AppletFunc(pidofMain))
	applets.Register("pgrep", applets.AppletFunc(pgrepMain))
	applets.Register("pkill", applets.AppletFunc(pkillMain))
}

func pidofMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: pidof: missing operand")
		return 1
	}

	single := false
	omitPIDs := map[int]bool{}
	name := ""
	pidArgs := args[1:]

	for len(pidArgs) > 0 {
		arg := pidArgs[0]
		if arg == "-s" {
			single = true
			pidArgs = pidArgs[1:]
			continue
		}
		if arg == "-o" && len(pidArgs) > 1 {
			omit, _ := strconv.Atoi(pidArgs[1])
			if omit > 0 {
				omitPIDs[omit] = true
			}
			pidArgs = pidArgs[2:]
			continue
		}
		if !strings.HasPrefix(arg, "-") {
			name = arg
			pidArgs = pidArgs[1:]
			break
		}
		pidArgs = pidArgs[1:]
	}

	if name == "" {
		fmt.Fprintln(os.Stderr, "gobox: pidof: missing operand")
		return 1
	}

	pids := []string{}
	myPid := os.Getpid()

	entries, _ := os.ReadDir("/proc")
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		if omitPIDs[pid] {
			continue
		}

		matched := false

		// Check /proc/PID/comm for process name
		status, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "comm"))
		if err == nil {
			commName := strings.TrimSpace(string(status))
			if commName == name {
				matched = true
			}
		}

		// Also check /proc/PID/cmdline for script names (all argv fields)
		if !matched {
			cmdline, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "cmdline"))
			if err == nil {
				fields := strings.Split(string(cmdline), "\x00")
				for fi, f := range fields {
					if len(f) > 0 && filepath.Base(f) == name {
						// Skip self when match is from non-argv[0] cmdline args
						if fi == 0 || pid != myPid {
							matched = true
							break
						}
					}
				}
			}
		}

		if matched {
			pids = append(pids, strconv.Itoa(pid))
			if single {
				break
			}
		}
	}

	if len(pids) > 0 {
		fmt.Println(strings.Join(pids, " "))
		return 0
	}
	return 1
}

func pgrepMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: pgrep: missing operand")
		return 1
	}

	pattern := args[1]
	pids := []string{}

	entries, _ := os.ReadDir("/proc")
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		status, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "comm"))
		if err != nil {
			continue
		}
		commName := strings.TrimSpace(string(status))
		if strings.Contains(commName, pattern) {
			pids = append(pids, strconv.Itoa(pid))
		}
	}

	if len(pids) > 0 {
		fmt.Println(strings.Join(pids, "\n"))
		return 0
	}
	return 1
}

func pkillMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: pkill: missing operand")
		return 1
	}

	pattern := args[1]
	exitCode := 1

	entries, _ := os.ReadDir("/proc")
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		status, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "comm"))
		if err != nil {
			continue
		}
		commName := strings.TrimSpace(string(status))
		if strings.Contains(commName, pattern) {
			process, err := os.FindProcess(pid)
			if err == nil {
				process.Signal(os.Kill)
				exitCode = 0
			}
		}
	}
	return exitCode
}
