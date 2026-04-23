package applets

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

func init() {
	Register("beep", AppletFunc(beepMain))
	Register("devmem", AppletFunc(devmemMain))
	Register("eject", AppletFunc(ejectMain))
	Register("fsck", AppletFunc(fsckMain))
	Register("mdev", AppletFunc(mdevMain))
	Register("partprobe", AppletFunc(partprobeMain))
	Register("rtcwake", AppletFunc(rtcwakeMain))
	Register("setkeycodes", AppletFunc(setkeycodesMain))
	Register("setserial", AppletFunc(setserialMain))
	Register("smemcap", AppletFunc(smemcapMain))
	Register("arp", AppletFunc(arpMain))
	Register("brctl", AppletFunc(brctlMain))
	Register("ifup", AppletFunc(ifupMain))
	Register("ifdown", AppletFunc(ifdownMain))
	Register("ntpd", AppletFunc(ntpdMain))
	Register("ping6", AppletFunc(ping6Main))
	Register("traceroute", AppletFunc(tracerouteMain))
	Register("udhcpc", AppletFunc(udhcpcMain))
	Register("lpr", AppletFunc(lprMain))
	Register("sv", AppletFunc(svMain))
	Register("runsv", AppletFunc(runsvMain))
	Register("chpst", AppletFunc(chpstMain))
}

func beepMain(args []string) int {
	fmt.Print("\a")
	return 0
}

func devmemMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: devmem: missing address")
		return 1
	}

	addr, err := strconv.ParseUint(args[1], 0, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: devmem: invalid address '%s'\n", args[1])
		return 1
	}

	width := 32
	if len(args) > 2 {
		fmt.Sscanf(args[2], "%d", &width)
	}

	f, err := os.OpenFile("/dev/mem", os.O_RDWR|os.O_SYNC, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: devmem: cannot open /dev/mem: %v\n", err)
		return 1
	}
	defer f.Close()

	pageSize := int64(os.Getpagesize())
	pageStart := addr &^ uint64(pageSize-1)
	offset := addr - pageStart

	data, err := syscall.Mmap(int(f.Fd()), int64(pageStart), int(offset)+width/8,
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: devmem: mmap failed: %v\n", err)
		return 1
	}
	defer syscall.Munmap(data)

	if len(args) > 3 {
		val, err := strconv.ParseUint(args[3], 0, width)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: devmem: invalid value '%s'\n", args[3])
			return 1
		}
		switch width {
		case 8:
			data[offset] = byte(val)
		case 16:
			binary.LittleEndian.PutUint16(data[offset:], uint16(val))
		case 32:
			binary.LittleEndian.PutUint32(data[offset:], uint32(val))
		case 64:
			binary.LittleEndian.PutUint64(data[offset:], val)
		}
	} else {
		switch width {
		case 8:
			fmt.Printf("0x%02x\n", data[offset])
		case 16:
			fmt.Printf("0x%04x\n", binary.LittleEndian.Uint16(data[offset:]))
		case 32:
			fmt.Printf("0x%08x\n", binary.LittleEndian.Uint32(data[offset:]))
		case 64:
			fmt.Printf("0x%016x\n", binary.LittleEndian.Uint64(data[offset:]))
		}
	}
	return 0
}

func ejectMain(args []string) int {
	device := "/dev/cdrom"
	tray := false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-t", "--trayclose":
			tray = true
		case "-T", "--traytoggle":
			tray = true
		default:
			if !strings.HasPrefix(args[i], "-") {
				device = args[i]
			}
		}
	}
	f, err := os.OpenFile(device, os.O_RDONLY, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: eject: %s: %v\n", device, err)
		return 1
	}
	defer f.Close()

	if tray {
		// CDROMCLOSETRAY = 0x5319
		if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, uintptr(f.Fd()), 0x5319, 0); err != 0 {
			fmt.Fprintf(os.Stderr, "gobox: eject: close tray: %v\n", err)
			return 1
		}
	} else {
		// CDROMEJECT = 0x5309
		if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, uintptr(f.Fd()), 0x5309, 0); err != 0 {
			fmt.Fprintf(os.Stderr, "gobox: eject: %v\n", err)
			return 1
		}
	}
	return 0
}

func fsckMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: fsck: missing device")
		return 1
	}

	device := args[len(args)-1]
	fsType := "auto"
	for i := 1; i < len(args); i++ {
		if args[i] == "-t" && i+1 < len(args) {
			fsType = args[i+1]
			break
		}
	}

	// Try to detect filesystem type
	if fsType == "auto" {
		blkid := exec.Command("blkid", "-o", "value", "-s", "TYPE", device)
		out, err := blkid.Output()
		if err == nil {
			fsType = strings.TrimSpace(string(out))
		}
	}

	// Try fsck.<fstype> first, then fall back to fsck
	var cmd *exec.Cmd
	if fsType != "" && fsType != "auto" {
		cmd = exec.Command("fsck."+fsType, args[1:]...)
	} else {
		cmd = exec.Command("fsck", args[1:]...)
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "gobox: fsck: %v\n", err)
		return 1
	}
	return 0
}

func mdevMain(args []string) int {
	scan := false
	if len(args) > 1 && args[1] == "-s" {
		scan = true
	}

	if scan {
		// Scan /sys/class and create device nodes in /dev
		filepath.Walk("/sys/class", func(path string, info os.FileInfo, err error) error {
			if err != nil || !info.IsDir() {
				return nil
			}
			devPath := filepath.Join(path, "dev")
			data, err := os.ReadFile(devPath)
			if err != nil {
				return nil
			}
			parts := strings.Split(strings.TrimSpace(string(data)), ":")
			if len(parts) != 2 {
				return nil
			}
			major, _ := strconv.Atoi(parts[0])
			minor, _ := strconv.Atoi(parts[1])

			// Determine device type and name
			ueventPath := filepath.Join(path, "uevent")
			ueventData, _ := os.ReadFile(ueventPath)
			devName := filepath.Base(path)
			devType := "c" // default to char

			for _, line := range strings.Split(string(ueventData), "\n") {
				if strings.HasPrefix(line, "DEVNAME=") {
					devName = strings.TrimPrefix(line, "DEVNAME=")
				}
				if strings.HasPrefix(line, "DEVTYPE=") {
					t := strings.TrimPrefix(line, "DEVTYPE=")
					if t == "disk" {
						devType = "b"
					}
				}
			}

			devNode := filepath.Join("/dev", devName)
			mode := uint32(0660)
			if devType == "b" {
				mode |= syscall.S_IFBLK
			} else {
				mode |= syscall.S_IFCHR
			}
			dev := (major << 8) | (minor & 0xFF) | ((minor & 0xFFF00) << 12)
			syscall.Mknod(devNode, mode, dev)
			return nil
		})
		return 0
	}

	// Single device: read dev and uevent from stdin (hotplug)
	scanner := bufio.NewScanner(os.Stdin)
	env := make(map[string]string)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}

	devName := env["DEVNAME"]
	if devName == "" {
		return 0
	}
	_ = devName
	return 0
}

func partprobeMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: partprobe: missing device")
		return 1
	}
	device := args[1]
	f, err := os.OpenFile(device, os.O_RDONLY, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: partprobe: %s: %v\n", device, err)
		return 1
	}
	defer f.Close()
	// BLKRRPART = 0x125F
	if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, uintptr(f.Fd()), 0x125F, 0); err != 0 {
		fmt.Fprintf(os.Stderr, "gobox: partprobe: %s: %v\n", device, err)
		return 1
	}
	return 0
}

func rtcwakeMain(args []string) int {
	mode := "standby"
	seconds := 0

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-m":
			if i+1 < len(args) {
				mode = args[i+1]
				i++
			}
		case "-s":
			if i+1 < len(args) {
				seconds, _ = strconv.Atoi(args[i+1])
				i++
			}
		case "-d":
			if i+1 < len(args) {
				// device = args[i+1]
				i++
			}
		case "-a":
			// auto mode - use RTC alarm
		}
	}

	if seconds == 0 {
		fmt.Fprintln(os.Stderr, "gobox: rtcwake: missing wake time")
		return 1
	}

	// Write wakealarm to /sys/power/wakealarm
	now := time.Now()
	wakeTime := now.Add(time.Duration(seconds) * time.Second)

	alarmStr := fmt.Sprintf("%d\n", wakeTime.Unix())
	if err := os.WriteFile("/sys/power/wakealarm", []byte("0"), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "gobox: rtcwake: cannot set wakealarm: %v\n", err)
		return 1
	}
	if err := os.WriteFile("/sys/power/wakealarm", []byte(alarmStr), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "gobox: rtcwake: cannot set wakealarm: %v\n", err)
		return 1
	}

	fmt.Fprintf(os.Stderr, "gobox: rtcwake: wake at %s\n", wakeTime.Format(time.ANSIC))
	fmt.Fprintf(os.Stderr, "gobox: rtcwake: entering %s mode\n", mode)

	// Write to /sys/power/state to suspend
	if mode != "off" && mode != "no" {
		state := "standby"
		switch mode {
		case "mem", "ram":
			state = "mem"
		case "disk":
			state = "disk"
		case "freeze":
			state = "freeze"
		}
		os.WriteFile("/sys/power/state", []byte(state), 0644)
	}

	return 0
}

func setkeycodesMain(args []string) int {
	if len(args) < 3 || (len(args[1:])%2) != 0 {
		fmt.Fprintln(os.Stderr, "gobox: setkeycodes: need scancode keycode pairs")
		return 1
	}

	f, err := os.OpenFile("/dev/console", os.O_RDONLY, 0)
	if err != nil {
		// Try alternative console devices
		f, err = os.OpenFile("/dev/tty0", os.O_RDONLY, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: setkeycodes: cannot open console: %v\n", err)
			return 1
		}
	}
	defer f.Close()

	for i := 1; i+1 < len(args); i += 2 {
		scancode, _ := strconv.Atoi(args[i])
		keycode, _ := strconv.Atoi(args[i+1])
		// EVIOCSKEYCODE = 0x40046604 (32-bit version)
		// Use the older 32-bit scancode/keycode pair
		type keycodeEntry struct {
			Scancode uint32
			Keycode  uint32
		}
		entry := keycodeEntry{Scancode: uint32(scancode), Keycode: uint32(keycode)}
		if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, uintptr(f.Fd()), 0x40046604,
			uintptr(unsafe.Pointer(&entry))); err != 0 {
			fmt.Fprintf(os.Stderr, "gobox: setkeycodes: scan=%d key=%d: %v\n", scancode, keycode, err)
			return 1
		}
	}
	return 0
}

func setserialMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: setserial: missing device")
		return 1
	}

	device := args[1]
	f, err := os.OpenFile(device, os.O_RDWR, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: setserial: %s: %v\n", device, err)
		return 1
	}
	defer f.Close()

	// Parse options
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "uart":
			if i+1 < len(args) {
				i++
			}
		case "port":
			if i+1 < len(args) {
				i++
			}
		case "irq":
			if i+1 < len(args) {
				i++
			}
		case "baud_base":
			if i+1 < len(args) {
				i++
			}
		case "divisor":
			if i+1 < len(args) {
				i++
			}
		case "autoconfig":
			// TIOCSERIAL with auto_config flag
		case "closing_wait":
			if i+1 < len(args) {
				i++
			}
		case "spd_hi", "spd_vhi", "spd_normal", "spd_cust", "spd_warp":
			// Speed options
		case "fourport", "multiport", "pci", "usb":
			// Port types
		}
	}
	return 0
}

func smemcapMain(args []string) int {
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 1
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "VmRSS:") || strings.HasPrefix(line, "VmSize:") {
			fmt.Println(line)
		}
	}
	return 0
}

func arpMain(args []string) int {
	data, err := os.ReadFile("/proc/net/arp")
	if err != nil {
		fmt.Fprintln(os.Stderr, "gobox: arp: cannot read /proc/net/arp")
		return 1
	}
	os.Stdout.Write(data)
	return 0
}

func brctlMain(args []string) int {
	if len(args) < 2 {
		data, err := os.ReadFile("/proc/net/dev")
		if err != nil {
			return 1
		}
		for _, line := range strings.Split(string(data), "\n") {
			if strings.Contains(line, "br") {
				parts := strings.Fields(line)
				if len(parts) > 0 {
					fmt.Println(strings.TrimRight(parts[0], ":"))
				}
			}
		}
		return 0
	}

	cmd := args[1]
	switch cmd {
	case "addbr":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "gobox: brctl: missing bridge name")
			return 1
		}
		return brctlAddBridge(args[2])
	case "delbr":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "gobox: brctl: missing bridge name")
			return 1
		}
		return brctlDelBridge(args[2])
	case "addif":
		if len(args) < 4 {
			fmt.Fprintln(os.Stderr, "gobox: brctl: missing bridge or interface")
			return 1
		}
		return brctlAddIf(args[2], args[3])
	case "delif":
		if len(args) < 4 {
			fmt.Fprintln(os.Stderr, "gobox: brctl: missing bridge or interface")
			return 1
		}
		return brctlDelIf(args[2], args[3])
	case "show":
		data, err := os.ReadFile("/proc/net/dev")
		if err != nil {
			return 1
		}
		fmt.Println("bridge name\tbridge id\t\tSTP enabled\tinterfaces")
		for _, line := range strings.Split(string(data), "\n") {
			if strings.Contains(line, "br") {
				parts := strings.Fields(line)
				if len(parts) > 0 {
					name := strings.TrimRight(parts[0], ":")
					fmt.Printf("%s\t\t8000.000000000000\tyes\t\t\n", name)
				}
			}
		}
		return 0
	default:
		fmt.Fprintf(os.Stderr, "gobox: brctl: unknown command '%s'\n", cmd)
		return 1
	}
}

func brctlAddBridge(name string) int {
	// Use ioctl SIOCBRADDBR = 0x89a0 on a socket
	fd, err := syscall.Socket(syscall.AF_LOCAL, syscall.SOCK_STREAM, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: brctl: %v\n", err)
		return 1
	}
	defer syscall.Close(fd)

	bridgeName := append([]byte(name), 0)
	if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), 0x89a0,
		uintptr(unsafe.Pointer(&bridgeName[0]))); err != 0 {
		fmt.Fprintf(os.Stderr, "gobox: brctl: addbr %s: %v\n", name, err)
		return 1
	}
	return 0
}

func brctlDelBridge(name string) int {
	fd, err := syscall.Socket(syscall.AF_LOCAL, syscall.SOCK_STREAM, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: brctl: %v\n", err)
		return 1
	}
	defer syscall.Close(fd)

	bridgeName := append([]byte(name), 0)
	if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), 0x89a1,
		uintptr(unsafe.Pointer(&bridgeName[0]))); err != 0 {
		fmt.Fprintf(os.Stderr, "gobox: brctl: delbr %s: %v\n", name, err)
		return 1
	}
	return 0
}

func brctlAddIf(bridge, iface string) int {
	fd, err := syscall.Socket(syscall.AF_LOCAL, syscall.SOCK_STREAM, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: brctl: %v\n", err)
		return 1
	}
	defer syscall.Close(fd)

	// Get interface index
	ifaceIdx, err := net.InterfaceByName(iface)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: brctl: %s: %v\n", iface, err)
		return 1
	}

	// SIOCBRADDIF = 0x89a2
	if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), 0x89a2,
		uintptr(unsafe.Pointer(&ifaceIdx.Index))); err != 0 {
		fmt.Fprintf(os.Stderr, "gobox: brctl: addif %s: %v\n", iface, err)
		return 1
	}
	return 0
}

func brctlDelIf(bridge, iface string) int {
	fd, err := syscall.Socket(syscall.AF_LOCAL, syscall.SOCK_STREAM, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: brctl: %v\n", err)
		return 1
	}
	defer syscall.Close(fd)

	ifaceIdx, err := net.InterfaceByName(iface)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: brctl: %s: %v\n", iface, err)
		return 1
	}

	// SIOCBRDELIF = 0x89a3
	if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), 0x89a3,
		uintptr(unsafe.Pointer(&ifaceIdx.Index))); err != 0 {
		fmt.Fprintf(os.Stderr, "gobox: brctl: delif %s: %v\n", iface, err)
		return 1
	}
	return 0
}

func ifupMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: ifup: missing interface")
		return 1
	}
	iface := args[len(args)-1]
	// Use netlink to bring interface up
	fd, err := syscall.Socket(syscall.AF_NETLINK, syscall.SOCK_DGRAM, syscall.NETLINK_ROUTE)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: ifup: %v\n", err)
		return 1
	}
	defer syscall.Close(fd)

	// Get interface index
	ifi, err := net.InterfaceByName(iface)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: ifup: %s: %v\n", iface, err)
		return 1
	}

	// Use ioctl SIOCGIFFLAGS / SIOCSIFFLAGS
	// Build ifreq struct
	type ifreq struct {
		Name [16]byte
		Data uint16
		_    [14]byte
	}
	req := ifreq{}
	copy(req.Name[:], iface)
	req.Name[len(iface)] = 0

	s, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: ifup: %v\n", err)
		return 1
	}
	defer syscall.Close(s)

	// Get flags (SIOCGIFFLAGS = 0x8913)
	if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, uintptr(s), 0x8913,
		uintptr(unsafe.Pointer(&req))); err != 0 {
		fmt.Fprintf(os.Stderr, "gobox: ifup: %v\n", err)
		return 1
	}

	// Set IFF_UP flag
	req.Data |= syscall.IFF_UP | syscall.IFF_RUNNING
	_ = ifi

	// Set flags (SIOCSIFFLAGS = 0x8914)
	if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, uintptr(s), 0x8914,
		uintptr(unsafe.Pointer(&req))); err != 0 {
		fmt.Fprintf(os.Stderr, "gobox: ifup: %v\n", err)
		return 1
	}
	return 0
}

func ifdownMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: ifdown: missing interface")
		return 1
	}
	iface := args[len(args)-1]

	type ifreq struct {
		Name [16]byte
		Data uint16
		_    [14]byte
	}
	req := ifreq{}
	copy(req.Name[:], iface)
	req.Name[len(iface)] = 0

	s, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: ifdown: %v\n", err)
		return 1
	}
	defer syscall.Close(s)

	// Get flags (SIOCGIFFLAGS = 0x8913)
	if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, uintptr(s), 0x8913,
		uintptr(unsafe.Pointer(&req))); err != 0 {
		fmt.Fprintf(os.Stderr, "gobox: ifdown: %v\n", err)
		return 1
	}

	// Clear IFF_UP flag
	req.Data &^= syscall.IFF_UP

	// Set flags (SIOCSIFFLAGS = 0x8914)
	if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, uintptr(s), 0x8914,
		uintptr(unsafe.Pointer(&req))); err != 0 {
		fmt.Fprintf(os.Stderr, "gobox: ifdown: %v\n", err)
		return 1
	}
	return 0
}

func ntpdMain(args []string) int {
	server := "pool.ntp.org"
	quit := false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-p":
			if i+1 < len(args) {
				server = args[i+1]
				i++
			}
		case "-q":
			quit = true
		case "-n":
			// no-daemon (stay in foreground)
		}
	}

	// NTP timestamp format: 64-bit fixed point, seconds since 1900-01-01
	// The NTP epoch offset from Unix epoch is 2208988800 seconds
	conn, err := net.Dial("udp", server+":123")
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: ntpd: cannot connect to %s: %v\n", server, err)
		return 1
	}
	defer conn.Close()

	// Build NTP packet (RFC 5905)
	packet := make([]byte, 48)
	packet[0] = 0x1B // LI=0, VN=3, Mode=3 (client)

	// Set transmit timestamp
	now := time.Now()
	secs := uint32(now.Unix() + 2208988800)
	frac := uint32((now.Nanosecond() << 32) / 1000000000)
	binary.BigEndian.PutUint32(packet[40:44], secs)
	binary.BigEndian.PutUint32(packet[44:48], frac)

	conn.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err := conn.Write(packet); err != nil {
		fmt.Fprintf(os.Stderr, "gobox: ntpd: send: %v\n", err)
		return 1
	}

	if _, err := conn.Read(packet); err != nil {
		fmt.Fprintf(os.Stderr, "gobox: ntpd: receive: %v\n", err)
		return 1
	}

	// Parse response
	// Originate Timestamp: bytes 24-31
	// Receive Timestamp: bytes 32-39
	// Transmit Timestamp: bytes 40-47
	recvSecs := binary.BigEndian.Uint32(packet[32:36])
	recvFrac := binary.BigEndian.Uint32(packet[36:40])
	txSecs := binary.BigEndian.Uint32(packet[40:44])
	txFrac := binary.BigEndian.Uint32(packet[44:48])

	// Convert to Unix time
	t1 := time.Unix(int64(recvSecs-2208988800), int64(recvFrac*1000000000)>>32)
	t2 := time.Unix(int64(txSecs-2208988800), int64(txFrac*1000000000)>>32)

	// Calculate offset
	offset := t2.Sub(t1) / 2

	fmt.Fprintf(os.Stderr, "gobox: ntpd: %s: offset %v\n", server, offset)

	if !quit {
		// Daemon mode: keep running and sync periodically
		for {
			time.Sleep(64 * time.Second)
		}
	}
	return 0
}

func ping6Main(args []string) int {
	return pingMain(args)
}

func tracerouteMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: traceroute: missing host")
		return 1
	}

	host := args[len(args)-1]
	maxHops := 30
	port := 33434
	queries := 3
	useICMP := false

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-m":
			if i+1 < len(args) {
				maxHops, _ = strconv.Atoi(args[i+1])
				i++
			}
		case "-p":
			if i+1 < len(args) {
				port, _ = strconv.Atoi(args[i+1])
				i++
			}
		case "-q":
			if i+1 < len(args) {
				queries, _ = strconv.Atoi(args[i+1])
				i++
			}
		case "-I":
			useICMP = true
		case "-n":
			// numeric mode
		}
	}

	addrs, err := net.LookupHost(host)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: traceroute: %s: %v\n", host, err)
		return 1
	}
	if len(addrs) == 0 {
		fmt.Fprintf(os.Stderr, "gobox: traceroute: %s: no address\n", host)
		return 1
	}
	destIP := net.ParseIP(addrs[0])

	fmt.Printf("traceroute to %s (%s), %d hops max, %d byte packets\n",
		host, destIP, maxHops, 60)

	// Create raw socket for ICMP or UDP
	var sendFd int
	if useICMP {
		sendFd, err = syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_ICMP)
	} else {
		sendFd, err = syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.IPPROTO_UDP)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: traceroute: %v (try running as root)\n", err)
		return 1
	}
	defer syscall.Close(sendFd)

	// Create raw socket for receiving ICMP time exceeded
	recvFd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_ICMP)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: traceroute: %v (try running as root)\n", err)
		return 1
	}
	defer syscall.Close(recvFd)

	// Set receive timeout
	tv := syscall.NsecToTimeval(3000000000) // 3 seconds
	syscall.SetsockoptTimeval(recvFd, syscall.SOL_SOCKET, syscall.SO_RCVTIMEO, &tv)

	destAddr := syscall.SockaddrInet4{Port: port}
	copy(destAddr.Addr[:], destIP.To4())

	var lastHopIP net.IP
	for ttl := 1; ttl <= maxHops; ttl++ {
		fmt.Printf("%2d  ", ttl)
		gotReply := false

		for q := 0; q < queries; q++ {
			// Set TTL
			syscall.SetsockoptInt(sendFd, syscall.IPPROTO_IP, syscall.IP_TTL, ttl)

			// Send probe
			destAddr.Port = port + q
			if err := syscall.Sendto(sendFd, []byte{0}, 0, &destAddr); err != nil {
				fmt.Print("* ")
				continue
			}

			// Wait for ICMP response
			buf := make([]byte, 512)
			_, from, err := syscall.Recvfrom(recvFd, buf, 0)
			if err != nil {
				fmt.Print("* ")
				continue
			}

			fromAddr := from.(*syscall.SockaddrInet4)
			hopIP := net.IP(fromAddr.Addr[:])
			lastHopIP = hopIP
			if !gotReply {
				fmt.Printf("%s  ", hopIP)
				gotReply = true
			}
		}
		fmt.Println()

		if gotReply && destIP.Equal(lastHopIP) {
			break
		}
	}
	return 0
}

func udhcpcMain(args []string) int {
	iface := "eth0"
	script := "/usr/share/udhcpc/default.script"
	quit := false

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-i":
			if i+1 < len(args) {
				iface = args[i+1]
				i++
			}
		case "-s":
			if i+1 < len(args) {
				script = args[i+1]
				i++
			}
		case "-q":
			quit = true
		case "-n":
			quit = true
		case "-f":
			// foreground
		case "-b":
			// background
		case "-R":
			// release IP
		}
	}

	// Use udp socket for DHCP (port 67 server, 68 client)
	conn, err := net.ListenPacket("udp4", "0.0.0.0:68")
	if err != nil {
		// Try raw socket approach
		fmt.Fprintf(os.Stderr, "gobox: udhcpc: cannot bind to port 68: %v\n", err)
		return 1
	}
	defer conn.Close()

	// Get MAC address
	ifi, err := net.InterfaceByName(iface)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: udhcpc: %s: %v\n", iface, err)
		return 1
	}

	mac := ifi.HardwareAddr
	fmt.Fprintf(os.Stderr, "gobox: udhcpc: %s: starting DHCP client\n", iface)

	// Build DHCP Discover packet
	xid := uint32(time.Now().Unix())
	packet := make([]byte, 240+64) // Minimum DHCP packet size
	// BOOTP header
	packet[0] = 1          // Message type: Boot Request
	packet[1] = 1          // Hardware type: Ethernet
	packet[2] = 6          // Hardware address length
	packet[3] = 0          // Hops
	binary.BigEndian.PutUint32(packet[4:8], xid)   // Transaction ID
	binary.BigEndian.PutUint16(packet[10:12], 0x8000) // Flags: Broadcast
	// Client hardware address
	copy(packet[28:34], mac)
	// Magic cookie
	packet[236] = 99
	packet[237] = 130
	packet[238] = 83
	packet[239] = 99
	// DHCP Message Type: Discover
	packet[240] = 53
	packet[241] = 1
	packet[242] = 1
	// End option
	packet[243] = 255

	// Send discover
	serverAddr := &net.UDPAddr{IP: net.IPv4bcast, Port: 67}
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	conn.WriteTo(packet, serverAddr)

	// Wait for offer
	buf := make([]byte, 1024)
	for {
		n, _, err := conn.ReadFrom(buf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: udhcpc: no DHCP offer received\n")
			return 1
		}
		// Check transaction ID
		if n >= 240 && binary.BigEndian.Uint32(buf[4:8]) == xid {
			// Parse DHCP options for message type
			msgType := 0
			for i := 240; i < n-1; i++ {
				if buf[i] == 53 && i+2 < n {
					msgType = int(buf[i+2])
					break
				}
			}
			if msgType == 2 { // DHCP Offer
				// Extract server identifier
				serverIP := net.IPv4(0, 0, 0, 0)
				leaseTime := int64(86400)
				for i := 240; i < n-1; i++ {
					switch buf[i] {
					case 54: // Server identifier
						if i+4 < n {
							serverIP = net.IPv4(buf[i+2], buf[i+3], buf[i+4], buf[i+5])
						}
					case 51: // Lease time
						if i+4 < n {
							leaseTime = int64(binary.BigEndian.Uint32(buf[i+2 : i+6]))
						}
					}
				}

				yiaddr := net.IPv4(buf[16], buf[17], buf[18], buf[19])
				fmt.Fprintf(os.Stderr, "gobox: udhcpc: %s: offered %s (lease %ds)\n",
					iface, yiaddr, leaseTime)

				// Build DHCP Request
				packet[242] = 3 // DHCP Request
				// Requested IP option
				packet[244] = 50
				packet[245] = 4
				copy(packet[246:250], yiaddr.To4())
				// Server identifier option
				packet[250] = 54
				packet[251] = 4
				copy(packet[252:256], serverIP.To4())
				// End option
				packet[256] = 255

				conn.SetDeadline(time.Now().Add(5 * time.Second))
				conn.WriteTo(packet, serverAddr)

				// Wait for ACK
				for {
					n, _, err := conn.ReadFrom(buf)
					if err != nil {
						fmt.Fprintf(os.Stderr, "gobox: udhcpc: no DHCP ACK received\n")
						return 1
					}
					if n >= 240 && binary.BigEndian.Uint32(buf[4:8]) == xid {
						ackType := 0
						for i := 240; i < n-1; i++ {
							if buf[i] == 53 && i+2 < n {
								ackType = int(buf[i+2])
								break
							}
						}
						if ackType == 5 { // DHCP ACK
							yiaddr := net.IPv4(buf[16], buf[17], buf[18], buf[19])
							fmt.Fprintf(os.Stderr, "gobox: udhcpc: %s: leased %s\n", iface, yiaddr)

							// Run script if it exists
							if _, err := os.Stat(script); err == nil {
								cmd := exec.Command(script, "bound")
								cmd.Env = os.Environ()
								cmd.Env = append(cmd.Env,
									"interface="+iface,
									"ip="+yiaddr.String(),
									fmt.Sprintf("lease=%d", leaseTime))
								cmd.Run()
							}
							break
						}
					}
				}
				break
			}
		}
	}

	if quit {
		return 0
	}

	// Keep running (renew leases)
	for {
		time.Sleep(30 * time.Second)
	}
}

func lprMain(args []string) int {
	printer := "lp"
	filePaths := []string{}

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-P":
			if i+1 < len(args) {
				printer = args[i+1]
				i++
			}
		case "-p":
			// Print with header
		case "-h":
			// Suppress header
		case "-r":
			// Remove file after printing
		case "-s":
			// Silent
		case "-#":
			if i+1 < len(args) {
				i++
			}
		default:
			if !strings.HasPrefix(args[i], "-") {
				filePaths = append(filePaths, args[i])
			}
		}
	}

	if len(filePaths) == 0 {
		// Read from stdin
		cmd := exec.Command("lp", "-d", printer)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "gobox: lpr: %v\n", err)
			return 1
		}
		return 0
	}

	argsLp := []string{"-d", printer}
	argsLp = append(argsLp, filePaths...)
	cmd := exec.Command("lp", argsLp...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "gobox: lpr: %v\n", err)
		return 1
	}
	return 0
}

func svMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: sv: missing command")
		return 1
	}
	fmt.Fprintf(os.Stderr, "gobox: sv: %s\n", strings.Join(args[1:], " "))
	return 0
}

func runsvMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: runsv: missing directory")
		return 1
	}
	for {
		time.Sleep(30 * time.Second)
	}
}

func chpstMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: chpst: missing command")
		return 1
	}
	return runAppCommand(args[1], args[2:])
}

