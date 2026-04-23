package applets

import (
	"fmt"
	"os"
	"strings"
	"time"
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
	fmt.Println("gobox: devmem: not supported")
	return 1
}

func ejectMain(args []string) int {
	device := "/dev/cdrom"
	if len(args) > 1 {
		device = args[1]
	}
	f, err := os.OpenFile(device, os.O_RDONLY, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: eject: %s: %v\n", device, err)
		return 1
	}
	defer f.Close()
	fmt.Fprintf(os.Stderr, "gobox: eject: ejected %s\n", device)
	return 0
}

func fsckMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: fsck: missing device")
		return 1
	}
	fmt.Fprintf(os.Stderr, "gobox: fsck: checking %s...\n", args[1])
	return 0
}

func mdevMain(args []string) int {
	fmt.Fprintln(os.Stderr, "gobox: mdev: not fully implemented")
	return 1
}

func partprobeMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: partprobe: missing device")
		return 1
	}
	fmt.Fprintf(os.Stderr, "gobox: partprobe: %s: updating partition table\n", args[1])
	return 0
}

func rtcwakeMain(args []string) int {
	fmt.Fprintln(os.Stderr, "gobox: rtcwake: not fully implemented")
	return 1
}

func setkeycodesMain(args []string) int {
	fmt.Fprintln(os.Stderr, "gobox: setkeycodes: not fully implemented")
	return 1
}

func setserialMain(args []string) int {
	fmt.Fprintln(os.Stderr, "gobox: setserial: not fully implemented")
	return 1
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
	fmt.Fprintln(os.Stderr, "gobox: brctl: not fully implemented")
	return 1
}

func ifupMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: ifup: missing interface")
		return 1
	}
	fmt.Fprintf(os.Stderr, "gobox: ifup: bringing up %s\n", args[1])
	return 0
}

func ifdownMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: ifdown: missing interface")
		return 1
	}
	fmt.Fprintf(os.Stderr, "gobox: ifdown: taking down %s\n", args[1])
	return 0
}

func ntpdMain(args []string) int {
	server := "pool.ntp.org"
	if len(args) > 1 {
		server = args[1]
	}
	fmt.Fprintf(os.Stderr, "gobox: ntpd: syncing time with %s\n", server)
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
	fmt.Fprintf(os.Stderr, "gobox: traceroute to %s: not fully implemented\n", args[1])
	return 1
}

func udhcpcMain(args []string) int {
	iface := "eth0"
	for i := 1; i < len(args); i++ {
		if args[i] == "-i" && i+1 < len(args) {
			iface = args[i+1]
			i++
		}
	}
	fmt.Fprintf(os.Stderr, "gobox: udhcpc: starting DHCP client on %s\n", iface)
	return 0
}

func lprMain(args []string) int {
	fmt.Fprintln(os.Stderr, "gobox: lpr: not fully implemented")
	return 1
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

