package applets

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"syscall"
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
	if len(args) < 2 {
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

	fmt.Fprintln(os.Stderr, "gobox: losetup: not fully implemented")
	return 1
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
	fmt.Fprintln(os.Stderr, "gobox: mkfs.ext2: not implemented")
	return 1
}

func mkfsVfatMain(args []string) int {
	fmt.Fprintln(os.Stderr, "gobox: mkfs.vfat: not implemented")
	return 1
}

func mkswapMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: mkswap: missing device")
		return 1
	}
	fmt.Fprintf(os.Stderr, "gobox: mkswap: setting up swap on %s\n", args[1])
	return 0
}

func swaponMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: swapon: missing device")
		return 1
	}
	_, _, errno := syscall.Syscall(syscall.SYS_SWAPON, uintptr(0), 0, 0)
	if errno != 0 {
		fmt.Fprintf(os.Stderr, "gobox: swapon: %v\n", errno)
		return 1
	}
	return 0
}

func swapoffMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: swapoff: missing device")
		return 1
	}
	_, _, errno := syscall.Syscall(syscall.SYS_SWAPOFF, uintptr(0), 0, 0)
	if errno != 0 {
		fmt.Fprintf(os.Stderr, "gobox: swapoff: %v\n", errno)
		return 1
	}
	return 0
}

func hwclockMain(args []string) int {
	fmt.Fprintln(os.Stderr, "gobox: hwclock: not fully implemented")
	return 1
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
	fmt.Fprintln(os.Stderr, "gobox: chrt: not fully implemented")
	return 1
}

func ioniceMain(args []string) int {
	fmt.Fprintln(os.Stderr, "gobox: ionice: not fully implemented")
	return 1
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
	fmt.Fprintln(os.Stderr, "gobox: nsenter: not fully implemented")
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
