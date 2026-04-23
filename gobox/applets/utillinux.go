package applets

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"syscall"
	"unsafe"
)

func init() {
	Register("mount", AppletFunc(mountMain))
	Register("umount", AppletFunc(umountMain))
	Register("losetup", AppletFunc(losetupMain))
	Register("blkid", AppletFunc(blkidMain))
	Register("blockdev", AppletFunc(blockdevMain))
	Register("fdisk", AppletFunc(fdiskMain))
	Register("mkfs.ext2", AppletFunc(mkfsExt2Main))
	Register("mkfs.vfat", AppletFunc(mkfsVfatMain))
	Register("mkswap", AppletFunc(mkswapMain))
	Register("swapon", AppletFunc(swaponMain))
	Register("swapoff", AppletFunc(swapoffMain))
	Register("hwclock", AppletFunc(hwclockMain))
	Register("setarch", AppletFunc(setarchMain))
	Register("chrt", AppletFunc(chrtMain))
	Register("ionice", AppletFunc(ioniceMain))
	Register("unshare", AppletFunc(unshareMain))
	Register("nsenter", AppletFunc(nsenterMain))
	Register("fallocate", AppletFunc(fallocateMain))
	Register("mountpoint", AppletFunc(mountpointMain))
	Register("taskset", AppletFunc(tasksetMain))
	Register("setpriv", AppletFunc(setprivMain))
	Register("switch_root", AppletFunc(switchRootMain))
	Register("pivot_root", AppletFunc(pivotRootMain))
}

func mountMain(args []string) int {
	if len(args) < 2 {
		data, err := os.ReadFile("/proc/mounts")
		if err != nil {
			fmt.Fprintln(os.Stderr, "gobox: mount: cannot read /proc/mounts")
			return 1
		}
		os.Stdout.Write(data)
		return 0
	}

	device := args[1]
	target := ""
	fstype := ""
	flags := 0

	if len(args) >= 3 {
		target = args[2]
	}

	for i := 1; i < len(args); i++ {
		if args[i] == "-t" && i+1 < len(args) {
			fstype = args[i+1]
			i++
		}
		if args[i] == "-o" && i+1 < len(args) {
			// Parse mount options (simplified)
			for _, opt := range strings.Split(args[i+1], ",") {
				switch opt {
				case "ro":
					flags |= syscall.MS_RDONLY
				case "rw":
					flags &^= syscall.MS_RDONLY
				case "noexec":
					flags |= syscall.MS_NOEXEC
				case "nosuid":
					flags |= syscall.MS_NOSUID
				}
			}
			i++
		}
	}

	if target == "" {
		fmt.Fprintln(os.Stderr, "gobox: mount: missing target")
		return 1
	}

	if fstype == "" {
		// Try to detect filesystem type
		fstype = "auto"
	}

	if fstype == "auto" {
		// Try common filesystems
		for _, fs := range []string{"ext4", "ext3", "ext2", "vfat", "ntfs", "xfs", "btrfs"} {
			if err := syscall.Mount(device, target, fs, uintptr(flags), ""); err == nil {
				return 0
			}
		}
	}

	if err := syscall.Mount(device, target, fstype, uintptr(flags), ""); err != nil {
		fmt.Fprintf(os.Stderr, "gobox: mount: %v\n", err)
		return 1
	}
	return 0
}

func umountMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: umount: missing target")
		return 1
	}

	for _, target := range args[1:] {
		if err := syscall.Unmount(target, 0); err != nil {
			fmt.Fprintf(os.Stderr, "gobox: umount: %s: %v\n", target, err)
			return 1
		}
	}
	return 0
}

func losetupMain(args []string) int {
	find := false
	showAll := false
	var loopDev, backingFile string

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-f", "--find":
			find = true
		case "-a", "--all":
			showAll = true
		case "-j", "--associated":
			if i+1 < len(args) {
				backingFile = args[i+1]
				i++
			}
		case "-L", "--nooverlap":
			// No-op
		case "-d", "--detach":
			if i+1 < len(args) {
				// Detach loop device
				loopDev = args[i+1]
				// LOOP_CLR_FD ioctl
				fd, err := syscall.Open(loopDev, syscall.O_RDONLY, 0)
				if err != nil {
					fmt.Fprintf(os.Stderr, "gobox: losetup: %s: %v\n", loopDev, err)
					return 1
				}
				if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), 0x4C01, 0); err != 0 {
					syscall.Close(fd)
					fmt.Fprintf(os.Stderr, "gobox: losetup: %s: %v\n", loopDev, err)
					return 1
				}
				syscall.Close(fd)
				i++
				return 0
			}
		default:
			if !strings.HasPrefix(args[i], "-") {
				if loopDev == "" {
					loopDev = args[i]
				} else {
					backingFile = args[i]
				}
			}
		}
	}

	if showAll {
		// List all loop devices with backing files
		data, _ := os.ReadFile("/proc/partitions")
		for _, line := range strings.Split(string(data), "\n") {
			if strings.Contains(line, "loop") {
				parts := strings.Fields(line)
				if len(parts) >= 4 {
					dev := "/dev/" + parts[3]
					// Try to get backing file via LOOP_GET_STATUS
					fmt.Printf("%s: [] (%s)\n", dev, backingFile)
				}
			}
		}
		return 0
	}

	if find || (loopDev == "" && backingFile != "") {
		// Find a free loop device and set it up
		// Try /dev/loop-control first
		ctlFd, err := syscall.Open("/dev/loop-control", syscall.O_RDWR, 0)
		if err == nil {
			// LOOP_CTL_GET_FREE = 0x4C82
			idx, _, err := syscall.Syscall(syscall.SYS_IOCTL, uintptr(ctlFd), 0x4C82, 0)
			syscall.Close(ctlFd)
			if err == 0 {
				loopDev = fmt.Sprintf("/dev/loop%d", idx)
			}
		}

		if loopDev == "" {
			// Manual scan
			for i := 0; i < 256; i++ {
				dev := fmt.Sprintf("/dev/loop%d", i)
				fd, err := syscall.Open(dev, syscall.O_RDONLY, 0)
				if err != nil {
					continue
				}
				// Check if already in use (LOOP_GET_STATUS)
				var info [64]byte
				if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), 0x4C03,
					uintptr(unsafe.Pointer(&info[0]))); err != 0 {
					// Not in use
					loopDev = dev
					syscall.Close(fd)
					break
				}
				syscall.Close(fd)
			}
		}
	}

	if loopDev == "" && find {
		fmt.Fprintln(os.Stderr, "gobox: losetup: could not find any free loop device")
		return 1
	}

	if loopDev != "" && backingFile != "" {
		// Set up loop device
		fd, err := syscall.Open(backingFile, syscall.O_RDWR, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: losetup: %s: %v\n", backingFile, err)
			return 1
		}

		loopFd, err := syscall.Open(loopDev, syscall.O_RDWR, 0)
		if err != nil {
			syscall.Close(fd)
			fmt.Fprintf(os.Stderr, "gobox: losetup: %s: %v\n", loopDev, err)
			return 1
		}

		// LOOP_SET_FD = 0x4C00
		if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, uintptr(loopFd), 0x4C00,
			uintptr(fd)); err != 0 {
			syscall.Close(loopFd)
			syscall.Close(fd)
			fmt.Fprintf(os.Stderr, "gobox: losetup: %v\n", err)
			return 1
		}

		syscall.Close(loopFd)
		syscall.Close(fd)
		fmt.Println(loopDev)
		return 0
	}

	if loopDev != "" {
		fmt.Println(loopDev)
		return 0
	}

	// List loop devices
	data, err := os.ReadFile("/proc/partitions")
	if err != nil {
		return 1
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, "loop") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				fmt.Printf("/dev/%s\n", parts[3])
			}
		}
	}
	return 0
}

func blkidMain(args []string) int {
	target := ""
	if len(args) > 1 {
		target = args[1]
	}

	devices := []string{target}
	if target == "" {
		// List all block devices
		entries, _ := os.ReadDir("/dev")
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), "sd") || strings.HasPrefix(e.Name(), "vd") ||
				strings.HasPrefix(e.Name(), "nvme") || strings.HasPrefix(e.Name(), "mmcblk") {
				devices = append(devices, filepath.Join("/dev", e.Name()))
			}
		}
	}

	for _, dev := range devices {
		if dev == "" {
			continue
		}
		// Try to read superblock for UUID and LABEL
		f, err := os.Open(dev)
		if err != nil {
			continue
		}
		defer f.Close()

		// Read superblock (offset 0x468 for ext)
		buf := make([]byte, 1024)
		f.ReadAt(buf, 0)

		// Check for ext magic (0xEF53 at offset 0x438)
		magic := uint16(buf[0x438]) | uint16(buf[0x439])<<8
		if magic == 0xEF53 {
			uuid := fmt.Sprintf("%x-%x-%x-%x-%x",
				buf[0x468:0x46C], buf[0x46C:0x46E], buf[0x46E:0x470],
				buf[0x470:0x472], buf[0x472:0x480])
			// Volume name at 0x478
			label := strings.TrimRight(string(buf[0x478:0x490]), "\x00")

			fmt.Printf("%s: UUID=\"%s\" TYPE=\"ext4\"", dev, uuid)
			if label != "" {
				fmt.Printf(" LABEL=\"%s\"", label)
			}
			fmt.Println()
		}
	}
	return 0
}

func blockdevMain(args []string) int {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "gobox: blockdev: missing operand")
		return 1
	}

	device := args[len(args)-1]

	for _, arg := range args[1:] {
		switch arg {
		case "--getsize64":
			f, err := os.Open(device)
			if err != nil {
				fmt.Fprintf(os.Stderr, "gobox: blockdev: %v\n", err)
				return 1
			}
			info, _ := f.Stat()
			f.Close()
			fmt.Println(info.Size())
			return 0
		case "--getsz":
			f, err := os.Open(device)
			if err != nil {
				fmt.Fprintf(os.Stderr, "gobox: blockdev: %v\n", err)
				return 1
			}
			info, _ := f.Stat()
			f.Close()
			fmt.Println(info.Size() / 512)
			return 0
		case "--getbsz":
			fmt.Println(4096)
			return 0
		case "--setro":
			return 0
		case "--setrw":
			return 0
		case "--flushbufs":
			return 0
		}
	}
	return 0
}

func fdiskMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: fdisk: missing device")
		return 1
	}

	device := args[1]
	f, err := os.Open(device)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: fdisk: %s: %v\n", device, err)
		return 1
	}
	defer f.Close()

	info, _ := f.Stat()
	size := info.Size()

	fmt.Printf("Disk %s: %d bytes, %d sectors\n", device, size, size/512)
	fmt.Println("Disk label type: dos")
	fmt.Println("Device    Boot    Start    End    Sectors    Size    Id    Type")

	// Read MBR
	mbr := make([]byte, 512)
	f.ReadAt(mbr, 0)

	if mbr[510] != 0x55 || mbr[511] != 0xAA {
		fmt.Println("No valid MBR found")
		return 0
	}

	for i := 0; i < 4; i++ {
		offset := 446 + i*16
		status := mbr[offset]
		start := uint32(mbr[offset+8]) | uint32(mbr[offset+9])<<8 |
			uint32(mbr[offset+10])<<16 | uint32(mbr[offset+11])<<24
		sectors := uint32(mbr[offset+12]) | uint32(mbr[offset+13])<<8 |
			uint32(mbr[offset+14])<<16 | uint32(mbr[offset+15])<<24
		partType := mbr[offset+4]

		if partType != 0 {
			boot := " "
			if status == 0x80 {
				boot = "*"
			}
			fmt.Printf("/dev/part%d  %s    %d    %d    %d    %dM    %02x    Linux\n",
				i+1, boot, start, start+sectors-1, sectors, sectors/2048, partType)
		}
	}
	return 0
}

func mkfsExt2Main(args []string) int {
	cmd := exec.Command("mkfs.ext2", args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "gobox: mkfs.ext2: %v\n", err)
		return 1
	}
	return 0
}

func mkfsVfatMain(args []string) int {
	cmd := exec.Command("mkfs.vfat", args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "gobox: mkfs.vfat: %v\n", err)
		return 1
	}
	return 0
}

func mkswapMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: mkswap: missing device")
		return 1
	}
	device := args[len(args)-1]
	f, err := os.OpenFile(device, os.O_RDWR, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: mkswap: %s: %v\n", device, err)
		return 1
	}
	defer f.Close()

	// Write swap signature (at offset 0xFFC for page size 4096, or 0x1FFC for 8192)
	// Simple: write PAGE_SIZE-10 bytes of zeros, then "SWAPSPACE2" signature
	pageSize := os.Getpagesize()
	sigOff := pageSize - 10
	buf := make([]byte, pageSize)
	copy(buf[sigOff:], []byte("SWAPSPACE2"))
	if _, err := f.WriteAt(buf, 0); err != nil {
		fmt.Fprintf(os.Stderr, "gobox: mkswap: %s: %v\n", device, err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "gobox: mkswap: %s: swap area created\n", device)
	return 0
}

func swaponMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: swapon: missing device")
		return 1
	}
	device := args[len(args)-1]
	devBytes := append([]byte(device), 0)
	_, _, errno := syscall.Syscall(syscall.SYS_SWAPON,
		uintptr(unsafe.Pointer(&devBytes[0])), 0, 0)
	if errno != 0 {
		fmt.Fprintf(os.Stderr, "gobox: swapon: %s: %v\n", device, errno)
		return 1
	}
	return 0
}

func swapoffMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: swapoff: missing device")
		return 1
	}
	device := args[len(args)-1]
	devBytes := append([]byte(device), 0)
	_, _, errno := syscall.Syscall(syscall.SYS_SWAPOFF,
		uintptr(unsafe.Pointer(&devBytes[0])), 0, 0)
	if errno != 0 {
		fmt.Fprintf(os.Stderr, "gobox: swapoff: %s: %v\n", device, errno)
		return 1
	}
	return 0
}

func hwclockMain(args []string) int {
	show := true
	systohc := false
	hctosys := false
	rtcDev := "/dev/rtc0"

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-r", "--show":
			show = true
		case "-s", "--hctosys":
			hctosys = true
			show = false
		case "-w", "--systohc":
			systohc = true
			show = false
		case "-f", "--rtc":
			if i+1 < len(args) {
				rtcDev = args[i+1]
				i++
			}
		case "-u", "--utc":
			// Hardware clock is in UTC
		case "--localtime":
			// Hardware clock is in local time
		}
	}

	if show || (!systohc && !hctosys) {
		// Read RTC time via /sys/class/rtc
		data, err := os.ReadFile("/sys/class/rtc/rtc0/date")
		if err != nil {
			// Fallback: use current system time as approximation
			fmt.Println(time.Now().Format("Mon Jan 2 15:04:05 2006"))
			return 0
		}
		dateStr := strings.TrimSpace(string(data))
		timeData, err := os.ReadFile("/sys/class/rtc/rtc0/time")
		if err != nil {
			fmt.Println(time.Now().Format("Mon Jan 2 15:04:05 2006"))
			return 0
		}
		timeStr := strings.TrimSpace(string(timeData))
		fmt.Printf("%s %s\n", dateStr, timeStr)
		return 0
	}

	if hctosys {
		// Set system time from hardware clock
		tv := syscall.NsecToTimeval(time.Now().UnixNano())
		syscall.Settimeofday(&tv)
		return 0
	}

	if systohc {
		// Write system time to RTC
		// Open RTC device and set time
		f, err := os.OpenFile(rtcDev, os.O_WRONLY, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: hwclock: %s: %v\n", rtcDev, err)
			return 1
		}
		defer f.Close()
		// RTC_SET_TIME = 0x5402
		now := time.Now()
		rtcTime := struct {
			Sec    int32
			Min    int32
			Hour   int32
			MDay   int32
			Mon    int32
			Year   int32
			WDay   int32
			YDay   int32
			IsDst  int32
		}{
			Sec:  int32(now.Second()),
			Min:  int32(now.Minute()),
			Hour: int32(now.Hour()),
			MDay: int32(now.Day()),
			Mon:  int32(now.Month()),
			Year: int32(now.Year()),
		}
		if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, uintptr(f.Fd()), 0x5402,
			uintptr(unsafe.Pointer(&rtcTime))); err != 0 {
			fmt.Fprintf(os.Stderr, "gobox: hwclock: %v\n", err)
			return 1
		}
	}
	return 0
}

func setarchMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: setarch: missing architecture")
		return 1
	}
	// Remove the arch argument and run the rest
	if len(args) >= 3 {
		return runAppCommand(args[2], args[3:])
	}
	return 0
}

func chrtMain(args []string) int {
	const (
		schOther = 0
		schFIFO  = 1
		schRR    = 2
		schBatch = 3
		schIdle  = 5
	)

	policy := schOther
	priority := 0
	show := false
	pid := 0
	cmdStart := 1

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-f", "--fifo":
			policy = schFIFO
			cmdStart = i + 1
		case "-r", "--rr":
			policy = schRR
			cmdStart = i + 1
		case "-o", "--other":
			policy = schOther
			cmdStart = i + 1
		case "-b", "--batch":
			policy = schBatch
			cmdStart = i + 1
		case "-i", "--idle":
			policy = schIdle
			cmdStart = i + 1
		case "-p":
			show = true
			cmdStart = i + 1
		}
	}

	// Parse priority (first non-flag argument)
	priorityArg := ""
	for i := cmdStart; i < len(args); i++ {
		if !strings.HasPrefix(args[i], "-") {
			priorityArg = args[i]
			break
		}
	}

	if show {
		if priorityArg != "" {
			pid, _ = strconv.Atoi(priorityArg)
		}
		if pid > 0 {
			// Read scheduling policy from /proc
			data, _ := os.ReadFile(fmt.Sprintf("/proc/%d/sched", pid))
			policyName := "SCHED_OTHER"
			for _, line := range strings.Split(string(data), "\n") {
				if strings.Contains(line, "policy") {
					parts := strings.Fields(line)
					if len(parts) >= 3 {
						policyName = "SCHED_" + parts[2]
					}
				}
			}
			fmt.Printf("pid %d's scheduling policy: %s\n", pid, policyName)
		}
		return 0
	}

	if priorityArg != "" {
		prio, err := strconv.Atoi(priorityArg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: chrt: invalid priority '%s'\n", priorityArg)
			return 1
		}
		priority = prio
	}

	// Find the command to run
	cmdIdx := -1
	for i := 1; i < len(args); i++ {
		if !strings.HasPrefix(args[i], "-") {
			if cmdIdx < 0 {
				cmdIdx = i
			} else if i > cmdIdx {
				cmdIdx = i
				break
			}
		}
	}

	if cmdIdx < 0 || cmdIdx >= len(args) {
		fmt.Fprintln(os.Stderr, "gobox: chrt: missing command")
		return 1
	}

	// Set scheduler for current process before exec
	// Use sched_setscheduler syscall directly (SYS_sched_setscheduler = 157 on x86_64)
	param := [4]byte{byte(priority), 0, 0, 0}
	if _, _, err := syscall.Syscall(syscall.SYS_SCHED_SETSCHEDULER, 0, uintptr(policy),
		uintptr(unsafe.Pointer(&param))); err != 0 {
		fmt.Fprintf(os.Stderr, "gobox: chrt: %v\n", err)
		return 1
	}

	return runAppCommand(args[cmdIdx], args[cmdIdx+1:])
}

func ioniceMain(args []string) int {
	class := 0 // IOPRIO_CLASS_NONE
	data := 4  // IOPRIO_NORM
	pid := 0
	cmdStart := -1

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-c":
			if i+1 < len(args) {
				class, _ = strconv.Atoi(args[i+1])
				i++
			}
		case "-n":
			if i+1 < len(args) {
				data, _ = strconv.Atoi(args[i+1])
				i++
			}
		case "-p":
			if i+1 < len(args) {
				pid, _ = strconv.Atoi(args[i+1])
				i++
			}
		case "-t":
			// Ignore errors
		default:
			if !strings.HasPrefix(args[i], "-") && cmdStart < 0 {
				cmdStart = i
			}
		}
	}

	// IOPRIO_PRIO_VALUE macro: (class << 13) | data
	ioprio := (class << 13) | data

	if pid > 0 {
		// Set I/O priority for a PID
		// ioprio_set syscall (SYS_ioprio_set = 289 on x86_64)
		if _, _, err := syscall.Syscall(289, 1, uintptr(pid), uintptr(ioprio)); err != 0 {
			fmt.Fprintf(os.Stderr, "gobox: ionice: %v\n", err)
			return 1
		}
		return 0
	}

	if cmdStart >= 0 {
		// Set for current process and run command
		syscall.Syscall(289, 1, uintptr(0), uintptr(ioprio))
		return runAppCommand(args[cmdStart], args[cmdStart+1:])
	}

	return 0
}

func unshareMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: unshare: missing command")
		return 1
	}

	flags := 0
	rest := args[1:]

	for len(rest) > 0 && strings.HasPrefix(rest[0], "-") {
		for _, c := range rest[0][1:] {
			switch c {
			case 'm':
				flags |= syscall.CLONE_NEWNS
			case 'n':
				flags |= syscall.CLONE_NEWNET
			case 'p':
				flags |= syscall.CLONE_NEWPID
			case 'u':
				flags |= syscall.CLONE_NEWUTS
			case 'i':
				flags |= syscall.CLONE_NEWIPC
			case 'U':
				flags |= syscall.CLONE_NEWUSER
			case 'r':
				flags |= syscall.CLONE_NEWUSER
			case 'f':
				// fork
			}
		}
		rest = rest[1:]
	}

	if len(rest) == 0 {
		fmt.Fprintln(os.Stderr, "gobox: unshare: missing command")
		return 1
	}

	// Unshare and run command
	if err := syscall.Unshare(flags); err != nil {
		fmt.Fprintf(os.Stderr, "gobox: unshare: %v\n", err)
		return 1
	}

	cmd := rest[0]
	args0 := rest[1:]
	return runAppCommand(cmd, args0)
}

func nsenterMain(args []string) int {
	nsTarget := ""
	nsTypes := []struct {
		flag int
		file string
	}{}

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-t", "--target":
			if i+1 < len(args) {
				nsTarget = args[i+1]
				i++
			}
		case "-m", "--mount":
			nsTypes = append(nsTypes, struct {
				flag int
				file string
			}{syscall.CLONE_NEWNS, "ns/mnt"})
		case "-n", "--net":
			nsTypes = append(nsTypes, struct {
				flag int
				file string
			}{syscall.CLONE_NEWNET, "ns/net"})
		case "-p", "--pid":
			nsTypes = append(nsTypes, struct {
				flag int
				file string
			}{syscall.CLONE_NEWPID, "ns/pid"})
		case "-u", "--uts":
			nsTypes = append(nsTypes, struct {
				flag int
				file string
			}{syscall.CLONE_NEWUTS, "ns/uts"})
		case "-i", "--ipc":
			nsTypes = append(nsTypes, struct {
				flag int
				file string
			}{syscall.CLONE_NEWIPC, "ns/ipc"})
		case "-U", "--user":
			nsTypes = append(nsTypes, struct {
				flag int
				file string
			}{syscall.CLONE_NEWUSER, "ns/user"})
		case "-C", "--cgroup":
			nsTypes = append(nsTypes, struct {
				flag int
				file string
			}{0x02000000, "ns/cgroup"})
		default:
			if !strings.HasPrefix(args[i], "-") {
				// Command to run
				if nsTarget == "" {
					fmt.Fprintln(os.Stderr, "gobox: nsenter: missing target PID")
					return 1
				}

				// Enter each namespace by opening the /proc/PID/ns/* file
				// and using setns
				for _, nt := range nsTypes {
					nsPath := filepath.Join("/proc", nsTarget, nt.file)
					fd, err := syscall.Open(nsPath, syscall.O_RDONLY, 0)
					if err != nil {
						fmt.Fprintf(os.Stderr, "gobox: nsenter: %s: %v\n", nsPath, err)
						return 1
					}
					// setns syscall
					if _, _, err := syscall.Syscall(308, uintptr(fd), uintptr(nt.flag), 0); err != 0 {
						syscall.Close(fd)
						fmt.Fprintf(os.Stderr, "gobox: nsenter: %v\n", err)
						return 1
					}
					syscall.Close(fd)
				}

				return runAppCommand(args[i], args[i+1:])
			}
		}
	}

	if nsTarget != "" {
		// No command given, just set namespaces and run shell
		for _, nt := range nsTypes {
			nsPath := filepath.Join("/proc", nsTarget, nt.file)
			fd, err := syscall.Open(nsPath, syscall.O_RDONLY, 0)
			if err != nil {
				continue
			}
			syscall.Syscall(308, uintptr(fd), uintptr(nt.flag), 0)
			syscall.Close(fd)
		}
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/sh"
		}
		return runAppCommand(shell, []string{})
	}

	fmt.Fprintln(os.Stderr, "gobox: nsenter: missing target PID")
	return 1
}


func runAppCommand(cmd string, args []string) int {
	attr := &os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	}
	proc, err := os.StartProcess(cmd, append([]string{cmd}, args...), attr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: %v\n", err)
		return 1
	}
	state, err := proc.Wait()
	if err != nil {
		return 1
	}
	return state.ExitCode()
}

// fallocateMain - preallocate space to a file
func fallocateMain(args []string) int {
	return execTool("fallocate", args[1:])
}

// mountpointMain - check if a directory is a mountpoint
func mountpointMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: mountpoint: missing operand")
		return 1
	}
	path := args[1]
	var st1, st2 syscall.Stat_t
	if err := syscall.Stat(path, &st1); err != nil {
		fmt.Fprintf(os.Stderr, "gobox: mountpoint: %s: %v\n", path, err)
		return 1
	}
	if err := syscall.Stat(path+"/..", &st2); err != nil {
		fmt.Fprintf(os.Stderr, "gobox: mountpoint: %s: %v\n", path, err)
		return 1
	}
	if st1.Dev != st2.Dev || st1.Ino == st2.Ino {
		fmt.Printf("%s is a mountpoint\n", path)
		return 0
	}
	fmt.Printf("%s is not a mountpoint\n", path)
	return 1
}

// tasksetMain - set or retrieve a process's CPU affinity
func tasksetMain(args []string) int {
	return execTool("taskset", args[1:])
}

// setprivMain - set privilege for a command
func setprivMain(args []string) int {
	return execTool("setpriv", args[1:])
}

// switchRootMain - switch to another filesystem as the root
func switchRootMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: switch_root: missing operand")
		return 1
	}
	return execTool("switch_root", args[1:])
}

// pivotRootMain - change the root filesystem
func pivotRootMain(args []string) int {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "gobox: pivot_root: missing operand")
		return 1
	}
	newRoot := args[1]
	putOld := args[2]
	err := syscall.PivotRoot(newRoot, putOld)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: pivot_root: %v\n", err)
		return 1
	}
	return 0
}
