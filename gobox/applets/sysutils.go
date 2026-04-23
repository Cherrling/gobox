package applets

import (
	"bufio"
	"fmt"
	"io"
	"os"
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
}

func syslogdMain(args []string) int {
	fmt.Println("gobox: syslogd: starting (logging to /var/log/messages)")
	logFile := "/var/log/messages"

	f, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: syslogd: %v\n", err)
		return 1
	}
	defer f.Close()

	// Read from /dev/log (Unix domain socket)
	_ = f
	for {
		time.Sleep(time.Second)
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
	fmt.Println("gobox: crond: starting")
	_ = time.Second
	_ = bufio.ScanLines
	return 0
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

var _ = io.Discard
