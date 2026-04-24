package applets

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func init() {
	Register("syslogd", AppletFunc(syslogdMain))
	Register("klogd", AppletFunc(klogdMain))
	Register("logread", AppletFunc(logreadMain))
	Register("crond", AppletFunc(crondMain))
	Register("crontab", AppletFunc(crontabMain))
	Register("watchdog", AppletFunc(watchdogMain))
	Register("rdate", AppletFunc(rdateMain))
	Register("acpid", AppletFunc(acpidMain))
	Register("adjtimex", AppletFunc(adjtimexMain))
	Register("sysctl", AppletFunc(sysctlMain))
	Register("fsfreeze", AppletFunc(fsfreezeMain))
	Register("fstrim", AppletFunc(fstrimMain))
	Register("fsync", AppletFunc(fsyncMain))
	Register("blkdiscard", AppletFunc(blkdiscardMain))
}

func syslogdMain(args []string) int {
	logFile := "/var/log/messages"
	small := false

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-R":
			if i+1 < len(args) {
				i++ // remote host
			}
		case "-L":
			small = true
		case "-s":
			if i+1 < len(args) {
				i++ // size
			}
		case "-b":
			if i+1 < len(args) {
				i++ // buffer size
			}
		case "-O":
			if i+1 < len(args) {
				logFile = args[i+1]
				i++
			}
		case "-S":
			small = true
		}
	}

	// Remove existing socket
	os.Remove("/dev/log")

	// Create Unix domain socket
	addr, err := net.ResolveUnixAddr("unixgram", "/dev/log")
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: syslogd: %v\n", err)
		return 1
	}
	conn, err := net.ListenUnixgram("unixgram", addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: syslogd: %v\n", err)
		return 1
	}
	defer conn.Close()
	os.Chmod("/dev/log", 0666)

	f, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: syslogd: %v\n", err)
		return 1
	}
	defer f.Close()

	fmt.Fprintf(os.Stderr, "gobox: syslogd: started, logging to %s\n", logFile)

	buf := make([]byte, 4096)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		msg := strings.TrimRight(string(buf[:n]), "\x00\n\r")
		if msg == "" {
			continue
		}
		timestamp := time.Now().Format("Jan  2 15:04:05")
		logLine := fmt.Sprintf("%s %s\n", timestamp, msg)
		f.WriteString(logLine)
		f.Sync()

		if !small {
			os.Stderr.WriteString(logLine)
		}
	}
}

func klogdMain(args []string) int {
	f, err := os.Open("/proc/kmsg")
	if err != nil {
		fmt.Fprintln(os.Stderr, "gobox: klogd: cannot open /proc/kmsg")
		return 1
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fmt.Fprintln(os.Stderr, scanner.Text())
	}
	return 0
}

func logreadMain(args []string) int {
	data, err := os.ReadFile("/var/log/messages")
	if err != nil {
		// Try other locations
		data, err = os.ReadFile("/var/log/syslog")
		if err != nil {
			fmt.Fprintln(os.Stderr, "gobox: logread: no log file found")
			return 1
		}
	}
	os.Stdout.Write(data)
	return 0
}

func crondMain(args []string) int {
	cronDir := "/etc/crontabs"
	foreground := true

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-d":
			if i+1 < len(args) {
				i++ // debug level
			}
		case "-l":
			if i+1 < len(args) {
				i++ // log level
			}
		case "-c":
			if i+1 < len(args) {
				cronDir = args[i+1]
				i++
			}
		case "-b":
			foreground = false
		case "-f":
			foreground = true
		}
	}

	fmt.Fprintf(os.Stderr, "gobox: crond: starting, cron dir: %s\n", cronDir)

	if !foreground {
		// Background mode: fork
		return 0
	}

	// Main loop
	for {
		now := time.Now()
		min := now.Minute()
		hour := now.Hour()
		day := now.Day()
		mon := int(now.Month())
		week := int(now.Weekday())

		// Check crontabs
		entries, err := os.ReadDir(cronDir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				cronFile := filepath.Join(cronDir, entry.Name())
				data, err := os.ReadFile(cronFile)
				if err != nil {
					continue
				}
				for _, line := range strings.Split(string(data), "\n") {
					line = strings.TrimSpace(line)
					if line == "" || strings.HasPrefix(line, "#") {
						continue
					}
					// Parse: min hour day mon week command
					fields := strings.Fields(line)
					if len(fields) < 6 {
						continue
					}
					cronMin := fields[0]
					cronHour := fields[1]
					cronDay := fields[2]
					cronMon := fields[3]
					cronWeek := fields[4]
					cmd := strings.Join(fields[5:], " ")

					if matchCron(cronMin, min) && matchCron(cronHour, hour) &&
						matchCron(cronDay, day) && matchCron(cronMon, mon) &&
						matchCron(cronWeek, week) {
						fmt.Fprintf(os.Stderr, "gobox: crond: running: %s\n", cmd)
						go func(cmdStr string) {
							parts := strings.Fields(cmdStr)
							if len(parts) > 0 {
								exec.Command(parts[0], parts[1:]...).Start()
							}
						}(cmd)
					}
				}
			}
		}

		// Sleep until next minute
		next := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute()+1, 0, 0, now.Location())
		time.Sleep(time.Until(next))
	}
}

func matchCron(pattern string, value int) bool {
	if pattern == "*" {
		return true
	}
	// Handle comma-separated lists
	for _, part := range strings.Split(pattern, ",") {
		part = strings.TrimSpace(part)
		if strings.Contains(part, "*/") {
			step := 0
			fmt.Sscanf(part, "*/%d", &step)
			if step > 0 && value%step == 0 {
				return true
			}
		} else if strings.Contains(part, "-") {
			var start, end int
			fmt.Sscanf(part, "%d-%d", &start, &end)
			if value >= start && value <= end {
				return true
			}
		} else {
			n, _ := strconv.Atoi(part)
			if n == value {
				return true
			}
		}
	}
	return false
}

func crontabMain(args []string) int {
	if len(args) < 2 {
		data, err := os.ReadFile("/etc/crontab")
		if err != nil {
			fmt.Fprintln(os.Stderr, "gobox: crontab: no crontab")
			return 1
		}
		os.Stdout.Write(data)
		return 0
	}

	// Parse simple crontab operations
	switch args[1] {
	case "-l":
		data, err := os.ReadFile("/etc/crontab")
		if err != nil {
			return 1
		}
		os.Stdout.Write(data)
	case "-e":
		fmt.Println("gobox: crontab: editing not supported")
	case "-r":
		os.Remove("/etc/crontab")
	default:
		fmt.Fprintln(os.Stderr, "gobox: crontab: unknown option")
		return 1
	}
	return 0
}

func watchdogMain(args []string) int {
	device := "/dev/watchdog"
	if len(args) > 1 {
		device = args[1]
	}

	f, err := os.OpenFile(device, os.O_WRONLY, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: watchdog: %s: %v\n", device, err)
		return 1
	}
	defer f.Close()

	// Pet the watchdog periodically
	for {
		f.Write([]byte("W"))
		time.Sleep(10 * time.Second)
	}
}

func rdateMain(args []string) int {
	fmt.Fprintln(os.Stderr, "gobox: rdate: not fully implemented")
	return 1
}


// acpidMain - ACPI daemon
func acpidMain(args []string) int {
	return execTool("acpid", args[1:])
}

// adjtimexMain - adjust kernel time variables
func adjtimexMain(args []string) int {
	return execTool("adjtimex", args[1:])
}

// sysctlMain - configure kernel parameters
func sysctlMain(args []string) int {
	return execTool("sysctl", args[1:])
}

// fsfreezeMain - suspend/resume access to a filesystem
func fsfreezeMain(args []string) int {
	return execTool("fsfreeze", args[1:])
}

// fstrimMain - discard unused blocks on a filesystem
func fstrimMain(args []string) int {
	return execTool("fstrim", args[1:])
}

// fsyncMain - synchronize a file's data to disk
func fsyncMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: fsync: missing file")
		return 1
	}
	exitCode := 0
	for _, path := range args[1:] {
		f, err := os.Open(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: fsync: %s: %v\n", path, err)
			exitCode = 1
			continue
		}
		if err := f.Sync(); err != nil {
			fmt.Fprintf(os.Stderr, "gobox: fsync: %s: %v\n", path, err)
			exitCode = 1
		}
		f.Close()
	}
	return exitCode
}

// blkdiscardMain - discard sectors on a device
func blkdiscardMain(args []string) int {
	return execTool("blkdiscard", args[1:])
}
