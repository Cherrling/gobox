package coreutils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"gobox/applets"
	"syscall"
)

func init() {
	applets.Register("ps", applets.AppletFunc(psMain))
	applets.Register("kill", applets.AppletFunc(killMain))
	applets.Register("killall", applets.AppletFunc(killallMain))
	applets.Register("pidof", applets.AppletFunc(pidofMain))
	applets.Register("pgrep", applets.AppletFunc(pgrepMain))
	applets.Register("pkill", applets.AppletFunc(pkillMain))
	applets.Register("free", applets.AppletFunc(freeMain))
	applets.Register("dmesg", applets.AppletFunc(dmesgMain))
	applets.Register("top", applets.AppletFunc(topMain))
	applets.Register("pstree", applets.AppletFunc(pstreeMain))
}

func psMain(args []string) int {
	// Read process info from /proc
	entries, err := os.ReadDir("/proc")
	if err != nil {
		fmt.Fprintln(os.Stderr, "gobox: ps: cannot read /proc")
		return 1
	}

	fmt.Printf("%-7s %-7s %-7s %-7s %s\n", "PID", "PPID", "UID", "TIME", "COMMAND")

	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		status, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "status"))
		if err != nil {
			continue
		}

		var name string
		var ppid int
		var uid int

		for _, line := range strings.Split(string(status), "\n") {
			if strings.HasPrefix(line, "Name:") {
				name = strings.TrimSpace(line[5:])
			}
			if strings.HasPrefix(line, "PPid:") {
				fmt.Sscanf(line[5:], "%d", &ppid)
			}
			if strings.HasPrefix(line, "Uid:") {
				fmt.Sscanf(line[4:], "%d", &uid)
			}
		}

		// Get process time from stat
		statData, _ := os.ReadFile(filepath.Join("/proc", entry.Name(), "stat"))
		_ = statData

		fmt.Printf("%-7d %-7d %-7d %-7s %s\n", pid, ppid, uid, "00:00", name)
	}
	return 0
}

func killMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: kill: missing operand")
		return 1
	}

	sig := syscall.SIGTERM
	start := 1

	if strings.HasPrefix(args[1], "-") {
		sigName := args[1][1:]
		switch sigName {
		case "9":
			sig = syscall.SIGKILL
		case "TERM":
			sig = syscall.SIGTERM
		case "INT":
			sig = syscall.SIGINT
		case "HUP":
			sig = syscall.SIGHUP
		case "USR1":
			sig = syscall.SIGUSR1
		case "USR2":
			sig = syscall.SIGUSR2
		case "STOP":
			sig = syscall.SIGSTOP
		case "CONT":
			sig = syscall.SIGCONT
		}
		start = 2
	}

	exitCode := 0
	for _, arg := range args[start:] {
		pid, err := strconv.Atoi(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: kill: invalid pid: %s\n", arg)
			exitCode = 1
			continue
		}
		proc, err := os.FindProcess(pid)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: kill: (%d): %v\n", pid, err)
			exitCode = 1
			continue
		}
		if err := proc.Signal(sig); err != nil {
			fmt.Fprintf(os.Stderr, "gobox: kill: (%d): %v\n", pid, err)
			exitCode = 1
		}
	}
	return exitCode
}

func killallMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: killall: missing operand")
		return 1
	}

	sig := syscall.SIGTERM
	start := 1

	if strings.HasPrefix(args[1], "-") {
		sigName := args[1][1:]
		switch sigName {
		case "9":
			sig = syscall.SIGKILL
		case "TERM":
			sig = syscall.SIGTERM
		}
		start = 2
	}

	name := args[start]
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
		if commName == name {
			proc, _ := os.FindProcess(pid)
			proc.Signal(sig)
			exitCode = 0
		}
	}
	return exitCode
}

func pidofMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: pidof: missing operand")
		return 1
	}

	name := args[1]
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
		if commName == name {
			pids = append(pids, strconv.Itoa(pid))
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
			proc, _ := os.FindProcess(pid)
			proc.Signal(syscall.SIGTERM)
			exitCode = 0
		}
	}
	return exitCode
}

func freeMain(args []string) int {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		fmt.Fprintln(os.Stderr, "gobox: free: cannot read /proc/meminfo")
		return 1
	}

	var total, free, avail, buffers, cached, swapTotal, swapFree int64

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			fmt.Sscanf(line, "MemTotal: %d kB", &total)
		} else if strings.HasPrefix(line, "MemFree:") {
			fmt.Sscanf(line, "MemFree: %d kB", &free)
		} else if strings.HasPrefix(line, "MemAvailable:") {
			fmt.Sscanf(line, "MemAvailable: %d kB", &avail)
		} else if strings.HasPrefix(line, "Buffers:") {
			fmt.Sscanf(line, "Buffers: %d kB", &buffers)
		} else if strings.HasPrefix(line, "Cached:") {
			fmt.Sscanf(line, "Cached: %d kB", &cached)
		} else if strings.HasPrefix(line, "SwapTotal:") {
			fmt.Sscanf(line, "SwapTotal: %d kB", &swapTotal)
		} else if strings.HasPrefix(line, "SwapFree:") {
			fmt.Sscanf(line, "SwapFree: %d kB", &swapFree)
		}
	}

	used := total - free - buffers - cached
	swapUsed := swapTotal - swapFree

	fmt.Printf("%7s %12s %12s %12s %12s %12s\n", "", "total", "used", "free", "shared", "buff/cache")
	fmt.Printf("%7s %12d %12d %12d %12d %12d\n", "Mem:", total/1024, used/1024, free/1024, 0, (buffers+cached)/1024)
	fmt.Printf("%7s %12d %12d %12d\n", "Swap:", swapTotal/1024, swapUsed/1024, swapFree/1024)
	return 0
}

func dmesgMain(args []string) int {
	data, err := os.ReadFile("/var/log/dmesg")
	if err != nil {
		// Try syslog
		data, err = os.ReadFile("/proc/kmsg")
		if err != nil {
			// Try dmesg command via syslog
			fmt.Fprintln(os.Stderr, "gobox: dmesg: cannot read kernel log")
			return 1
		}
	}

	os.Stdout.Write(data)
	return 0
}

func topMain(args []string) int {
	batch := false
	delay := 5
	nIter := 0

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-b":
			batch = true
		case "-d":
			if i+1 < len(args) {
				delay, _ = strconv.Atoi(args[i+1])
				i++
			}
		case "-n":
			if i+1 < len(args) {
				nIter, _ = strconv.Atoi(args[i+1])
				i++
			}
		case "-H":
			// Threads mode
		}
	}

	iter := 0
	for {
		if nIter > 0 && iter >= nIter {
			break
		}
		iter++

		memTotal, memFree, _ := getMemInfo()
		loadData, _ := os.ReadFile("/proc/loadavg")
		loadAvg := "0.00 0.00 0.00"
		if loadData != nil {
			parts := strings.Fields(string(loadData))
			if len(parts) >= 3 {
				loadAvg = strings.Join(parts[:3], " ")
			}
		}

		if !batch {
			fmt.Print("\033[H\033[J")
			fmt.Printf("top - %s up %s, load average: %s\n",
				time.Now().Format("15:04:05"),
				getUptime(), loadAvg)
			fmt.Printf("Mem: %6dK total, %6dK free\n", memTotal, memFree)
			fmt.Println("  PID  PPID  VSZ     RSS     S  COMMAND")
		}

		entries, _ := os.ReadDir("/proc")
		type procEntry struct {
			pid  int
			ppid int
			vsz  int64
			rss  int64
			stat byte
			cmd  string
		}
		var procs []procEntry

		for _, entry := range entries {
			pid, err := strconv.Atoi(entry.Name())
			if err != nil {
				continue
			}
			status, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "status"))
			if err != nil {
				continue
			}
			statData, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "stat"))
			if err != nil {
				continue
			}

			var pe procEntry
			pe.pid = pid

			for _, line := range strings.Split(string(status), "\n") {
				if strings.HasPrefix(line, "Name:") {
					pe.cmd = strings.TrimSpace(strings.TrimPrefix(line, "Name:"))
				}
				if strings.HasPrefix(line, "PPid:") {
					fmt.Sscanf(strings.TrimPrefix(line, "PPid:"), "%d", &pe.ppid)
				}
				if strings.HasPrefix(line, "VmSize:") {
					fmt.Sscanf(strings.TrimPrefix(line, "VmSize:"), "%d", &pe.vsz)
				}
				if strings.HasPrefix(line, "VmRSS:") {
					fmt.Sscanf(strings.TrimPrefix(line, "VmRSS:"), "%d", &pe.rss)
				}
			}

			parenEnd := strings.LastIndex(string(statData), ")")
			if parenEnd > 0 && parenEnd+2 < len(statData) {
				pe.stat = statData[parenEnd+2]
			}

			procs = append(procs, pe)
		}

		sort.Slice(procs, func(i, j int) bool {
			return procs[i].pid < procs[j].pid
		})

		for _, p := range procs {
			if p.cmd == "" {
				continue
			}
			fmt.Printf("%5d  %5d  %6d  %6d  %c  %s\n",
				p.pid, p.ppid, p.vsz, p.rss, p.stat, p.cmd)
		}

		if batch {
			break
		}
		time.Sleep(time.Duration(delay) * time.Second)
	}
	return 0
}

func getUptime() string {
	data, _ := os.ReadFile("/proc/uptime")
	if data == nil {
		return "?"
	}
	var secs float64
	fmt.Sscanf(string(data), "%f", &secs)
	days := int(secs / 86400)
	hours := int(secs/3600) % 24
	mins := int(secs/60) % 60
	if days > 0 {
		return fmt.Sprintf("%d days, %02d:%02d", days, hours, mins)
	}
	return fmt.Sprintf("%02d:%02d", hours, mins)
}

func getMemInfo() (total, free int64, _ error) {
	data, _ := os.ReadFile("/proc/meminfo")
	if data == nil {
		return 0, 0, fmt.Errorf("cannot read /proc/meminfo")
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			fmt.Sscanf(line, "MemTotal: %d kB", &total)
		}
		if strings.HasPrefix(line, "MemFree:") {
			fmt.Sscanf(line, "MemFree: %d kB", &free)
		}
	}
	return
}

type procInfo struct {
	pid   int
	ppid  int
	name  string
}

func pstreeMain(args []string) int {
	// Simple pstree: read /proc and build tree
	entries, _ := os.ReadDir("/proc")

	var procs []procInfo

	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		status, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "status"))
		if err != nil {
			continue
		}

		var name string
		var ppid int

		for _, line := range strings.Split(string(status), "\n") {
			if strings.HasPrefix(line, "Name:") {
				name = strings.TrimSpace(line[5:])
			}
			if strings.HasPrefix(line, "PPid:") {
				fmt.Sscanf(line[5:], "%d", &ppid)
			}
		}

		procs = append(procs, procInfo{pid, ppid, name})
	}

	// Find init (PID 1)
	for _, p := range procs {
		if p.pid == 1 {
			fmt.Printf("init(%d)\n", p.pid)
			printChildren(procs, p.pid, 1)
			break
		}
	}
	return 0
}

func printChildren(procs []procInfo, parentPid int, depth int) {
	for _, p := range procs {
		if p.ppid == parentPid {
			prefix := strings.Repeat("  ", depth)
			fmt.Printf("%s`-%s(%d)\n", prefix, p.name, p.pid)
			printChildren(procs, p.pid, depth+1)
		}
	}
}

// Silence unused import warning
var _ = io.ReadAll
