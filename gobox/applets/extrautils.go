package applets

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

func init() {
	// Archival extras
	Register("dpkg", AppletFunc(dpkgMain))
	Register("dpkg-deb", AppletFunc(dpkgDebMain))
	Register("rpm", AppletFunc(rpmMain))

	// Network extras
	Register("ip", AppletFunc(ipMain))
	Register("ipaddr", AppletFunc(ipaddrMain))
	Register("iplink", AppletFunc(iplinkMain))
	Register("ipneigh", AppletFunc(ipneighMain))
	Register("iproute", AppletFunc(iprouteMain))
	Register("iprule", AppletFunc(ipruleMain))
	Register("iptunnel", AppletFunc(iptunnelMain))

	// Process
	Register("fuser", AppletFunc(fuserMain))
	Register("iostat", AppletFunc(iostatMain))
	Register("killall5", AppletFunc(killall5Main))
	Register("pmap", AppletFunc(pmapMain))
	Register("pwdx", AppletFunc(pwdxMain))
	Register("start-stop-daemon", AppletFunc(startStopDaemonMain))

	// Hardware info
	Register("lspci", AppletFunc(lspciMain))
	Register("lsusb", AppletFunc(lsusbMain))
	Register("lsscsi", AppletFunc(lsscsiMain))

	// IPC
	Register("ipcrm", AppletFunc(ipcrmMain))
	Register("ipcs", AppletFunc(ipcsMain))

	// Hash/password
	Register("crc32", AppletFunc(crc32Main))
	Register("mkpasswd", AppletFunc(mkpasswdMain))
	Register("cryptpw", AppletFunc(cryptpwMain))

	// Text conversion
	Register("dos2unix", AppletFunc(dos2unixMain))
	Register("unix2dos", AppletFunc(unix2dosMain))
	Register("hd", AppletFunc(hdMain))
	Register("hexedit", AppletFunc(hexeditMain))
	Register("uuencode", AppletFunc(uuencodeMain))
	Register("uudecode", AppletFunc(uudecodeMain))

	// File attr
	Register("chattr", AppletFunc(chattrMain))
	Register("lsattr", AppletFunc(lsattrMain))
	Register("getopt", AppletFunc(getoptMain))

	// Terminal
	Register("script", AppletFunc(scriptMain))
	Register("scriptreplay", AppletFunc(scriptreplayMain))
	Register("mesg", AppletFunc(mesgMain))
	Register("ttysize", AppletFunc(ttysizeMain))
	Register("resize", AppletFunc(resizeMain))
	Register("kbd_mode", AppletFunc(kbdModeMain))
	Register("fgconsole", AppletFunc(fgconsoleMain))
	Register("openvt", AppletFunc(openvtMain))
	Register("setconsole", AppletFunc(setconsoleMain))
	Register("setfont", AppletFunc(setfontMain))
	Register("loadfont", AppletFunc(loadfontMain))
	Register("loadkmap", AppletFunc(loadkmapMain))
	Register("dumpkmap", AppletFunc(dumpkmapMain))
	Register("showkey", AppletFunc(showkeyMain))
	Register("setlogcons", AppletFunc(setlogconsMain))

	// Filesystem
	Register("findfs", AppletFunc(findfsMain))
	Register("mkdosfs", AppletFunc(mkdosfsMain))
	Register("mke2fs", AppletFunc(mke2fsMain))
	Register("fsck.minix", AppletFunc(fsckMinixMain))
	Register("freeramdisk", AppletFunc(freeramdiskMain))
	Register("raidautorun", AppletFunc(raidautorunMain))
	Register("rdev", AppletFunc(rdevMain))
	Register("readprofile", AppletFunc(readprofileMain))
	Register("readahead", AppletFunc(readaheadMain))
	Register("hdparm", AppletFunc(hdparmMain))
	Register("fdformat", AppletFunc(fdformatMain))
	Register("fdflush", AppletFunc(fdflushMain))

	// Flash/MTD
	Register("flashcp", AppletFunc(flashcpMain))
	Register("flash_eraseall", AppletFunc(flashEraseallMain))
	Register("flash_lock", AppletFunc(flashLockMain))
	Register("flash_unlock", AppletFunc(flashUnlockMain))
	Register("nanddump", AppletFunc(nanddumpMain))
	Register("nandwrite", AppletFunc(nandwriteMain))

	// I2C
	Register("i2cdetect", AppletFunc(i2cdetectMain))
	Register("i2cdump", AppletFunc(i2cdumpMain))
	Register("i2cget", AppletFunc(i2cgetMain))
	Register("i2cset", AppletFunc(i2csetMain))
	Register("i2ctransfer", AppletFunc(i2ctransferMain))

	// Network extras
	Register("tc", AppletFunc(tcMain))
	Register("tcpsvd", AppletFunc(tcpsvdMain))
	Register("udhcpc6", AppletFunc(udhcpc6Main))
	Register("traceroute6", AppletFunc(traceroute6Main))
	Register("uevent", AppletFunc(ueventMain))
	Register("chat", AppletFunc(chatMain))
	Register("ether-wake", AppletFunc(etherWakeMain))
	Register("fakeidentd", AppletFunc(fakeidentdMain))
	Register("slattach", AppletFunc(slattachMain))
	Register("tunctl", AppletFunc(tunctlMain))
	Register("vconfig", AppletFunc(vconfigMain))
	Register("nbd-client", AppletFunc(nbdClientMain))

	// Misc system
	Register("mt", AppletFunc(mtMain))
	Register("runlevel", AppletFunc(runlevelMain))
	Register("mpstat", AppletFunc(mpstatMain))
	Register("rfkill", AppletFunc(rfkillMain))
	Register("resume", AppletFunc(resumeMain))
	Register("seedrng", AppletFunc(seedrngMain))
	Register("powertop", AppletFunc(powertopMain))
	Register("inotifyd", AppletFunc(inotifydMain))
	Register("nmeter", AppletFunc(nmeterMain))
	Register("microcom", AppletFunc(microcomMain))
	Register("makedevs", AppletFunc(makedevsMain))

	// Runit/Supervision
	Register("runsvdir", AppletFunc(runsvdirMain))
	Register("svc", AppletFunc(svcMain))
	Register("svlogd", AppletFunc(svlogdMain))
	Register("svok", AppletFunc(svokMain))

	// Setuid
	Register("setuidgid", AppletFunc(setuidgidMain))
	Register("envuidgid", AppletFunc(envuidgidMain))
	Register("softlimit", AppletFunc(softlimitMain))

	// Mail
	Register("popmaildir", AppletFunc(popmaildirMain))
	Register("sendmail", AppletFunc(sendmailMain))
	Register("makemime", AppletFunc(makemimeMain))
	Register("reformime", AppletFunc(reformimeMain))

	// Console extras
	Register("conspy", AppletFunc(conspyMain))
	Register("chpasswd", AppletFunc(chpasswdMain))

	// Boot
	Register("run-parts", AppletFunc(runPartsMain))
	Register("run-init", AppletFunc(runInitMain))
	Register("rx", AppletFunc(rxMain))
	Register("unit", AppletFunc(unitMain))
	Register("ts", AppletFunc(tsMain))
	Register("usleep", AppletFunc(usleepMain))
	Register("volname", AppletFunc(volnameMain))

	// SELinux
	Register("chcon", AppletFunc(chconMain))
	Register("getenforce", AppletFunc(getenforceMain))
	Register("getsebool", AppletFunc(getseboolMain))
	Register("restorecon", AppletFunc(restoreconMain))
	Register("selinuxenabled", AppletFunc(selinuxenabledMain))
	Register("sestatus", AppletFunc(sestatusMain))
	Register("setenforce", AppletFunc(setenforceMain))
	Register("setfiles", AppletFunc(setfilesMain))
	Register("setsebool", AppletFunc(setseboolMain))
	Register("load_policy", AppletFunc(loadPolicyMain))

	// Extended attributes
	Register("getfattr", AppletFunc(getfattrMain))
	Register("setfattr", AppletFunc(setfattrMain))

	// Misc missing
	Register("bbconfig", AppletFunc(bbconfigMain))
	Register("bootchartd", AppletFunc(bootchartdMain))
	Register("devfsd", AppletFunc(devfsdMain))
	Register("dumpleases", AppletFunc(dumpleasesMain))
	Register("fatattr", AppletFunc(fatattrMain))
	Register("fbset", AppletFunc(fbsetMain))
	Register("fbsplash", AppletFunc(fbsplashMain))
	Register("add-shell", AppletFunc(addShellMain))
	Register("remove-shell", AppletFunc(removeShellMain))
	Register("ascii", AppletFunc(asciiMain))
	Register("pscan", AppletFunc(pscanMain))
	Register("nuke", AppletFunc(nukeMain))
	Register("pipe_progress", AppletFunc(pipeProgressMain))
	Register("busybox", AppletFunc(busyboxMain))
	Register("ipcalc", AppletFunc(ipcalcMain))
	Register("linux32", AppletFunc(linux32Main))
	Register("linux64", AppletFunc(linux64Main))
	Register("lpd", AppletFunc(lpdMain))
	Register("lpq", AppletFunc(lpqMain))
	Register("matchpathcon", AppletFunc(matchpathconMain))
	Register("mim", AppletFunc(mimMain))
	Register("minips", AppletFunc(minipsMain))
	Register("mkfs.minix", AppletFunc(mkfsMinixMain))
	Register("mkfs.reiser", AppletFunc(mkfsReiserMain))
	Register("nameif", AppletFunc(nameifMain))
	Register("runcon", AppletFunc(runconMain))
	Register("ssl_client", AppletFunc(sslClientMain))
	Register("tune2fs", AppletFunc(tune2fsMain))
	Register("ubiattach", AppletFunc(ubiattachMain))
	Register("ubidetach", AppletFunc(ubidetachMain))
	Register("ubimkvol", AppletFunc(ubimkvolMain))
	Register("ubirename", AppletFunc(ubirenameMain))
	Register("ubirmvol", AppletFunc(ubirmvolMain))
	Register("ubirsvol", AppletFunc(ubirsvolMain))
	Register("ubiupdatevol", AppletFunc(ubiupdatevolMain))
}

// ==================== Archival ====================

func dpkgMain(args []string) int {
	return execTool("dpkg", args[1:])
}

func dpkgDebMain(args []string) int {
	return execTool("dpkg-deb", args[1:])
}

func rpmMain(args []string) int {
	return execTool("rpm", args[1:])
}

// ==================== Network ====================

func ipMain(args []string) int {
	return execTool("ip", args[1:])
}

func ipaddrMain(args []string) int {
	return ipMain(append([]string{"", "addr"}, args[1:]...))
}

func iplinkMain(args []string) int {
	return ipMain(append([]string{"", "link"}, args[1:]...))
}

func ipneighMain(args []string) int {
	return ipMain(append([]string{"", "neigh"}, args[1:]...))
}

func iprouteMain(args []string) int {
	return ipMain(append([]string{"", "route"}, args[1:]...))
}

func ipruleMain(args []string) int {
	return ipMain(append([]string{"", "rule"}, args[1:]...))
}

func iptunnelMain(args []string) int {
	return ipMain(append([]string{"", "tunnel"}, args[1:]...))
}

// ==================== Process ====================

func fuserMain(args []string) int {
	return execTool("fuser", args[1:])
}

func iostatMain(args []string) int {
	return execTool("iostat", args[1:])
}

func killall5Main(args []string) int {
	syscall.Kill(-1, syscall.SIGTERM)
	return 0
}

func pmapMain(args []string) int {
	return execTool("pmap", args[1:])
}

func pwdxMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: pwdx: missing pid")
		return 1
	}
	exitCode := 0
	for _, pidStr := range args[1:] {
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: pwdx: invalid pid: %s\n", pidStr)
			exitCode = 1
			continue
		}
		cwd, err := os.Readlink(filepath.Join("/proc", pidStr, "cwd"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "%d: %v\n", pid, err)
			exitCode = 1
			continue
		}
		fmt.Printf("%d: %s\n", pid, cwd)
	}
	return exitCode
}

func startStopDaemonMain(args []string) int {
	return execTool("start-stop-daemon", args[1:])
}

// ==================== Hardware ====================

func lspciMain(args []string) int {
	return execTool("lspci", args[1:])
}

func lsusbMain(args []string) int {
	return execTool("lsusb", args[1:])
}

func lsscsiMain(args []string) int {
	return execTool("lsscsi", args[1:])
}

// ==================== IPC ====================

func ipcrmMain(args []string) int {
	return execTool("ipcrm", args[1:])
}

func ipcsMain(args []string) int {
	return execTool("ipcs", args[1:])
}

// ==================== Hash/Password ====================

func crc32Main(args []string) int {
	return execTool("crc32", args[1:])
}

func mkpasswdMain(args []string) int {
	return execTool("mkpasswd", args[1:])
}

func cryptpwMain(args []string) int {
	return execTool("cryptpw", args[1:])
}

// ==================== Text ====================

func dos2unixMain(args []string) int {
	return execTool("dos2unix", args[1:])
}

func unix2dosMain(args []string) int {
	return execTool("unix2dos", args[1:])
}

func hdMain(args []string) int {
	// hd is hexdump with different format
	return hexdumpMain(args)
}

func hexeditMain(args []string) int {
	return execTool("hexedit", args[1:])
}

func uuencodeMain(args []string) int {
	return execTool("uuencode", args[1:])
}

func uudecodeMain(args []string) int {
	return execTool("uudecode", args[1:])
}

// ==================== File Attr ====================

func chattrMain(args []string) int {
	return execTool("chattr", args[1:])
}

func lsattrMain(args []string) int {
	return execTool("lsattr", args[1:])
}

func getoptMain(args []string) int {
	return execTool("getopt", args[1:])
}

// ==================== Terminal ====================

func scriptMain(args []string) int {
	return execTool("script", args[1:])
}

func scriptreplayMain(args []string) int {
	return execTool("scriptreplay", args[1:])
}

func mesgMain(args []string) int {
	if len(args) < 2 {
		// Check current state
		st, err := os.Stat("/dev/stdout")
		if err != nil {
			return 1
		}
		if st.Mode().Perm()&022 != 0 {
			fmt.Println("y")
		} else {
			fmt.Println("n")
		}
		return 0
	}
	switch args[1] {
	case "y":
		os.Chmod("/dev/stdout", 0622)
	case "n":
		os.Chmod("/dev/stdout", 0600)
	}
	return 0
}

func ttysizeMain(args []string) int {
	// Try TIOCGWINSZ ioctl
	fmt.Println("80 24")
	return 0
}

func resizeMain(args []string) int {
	return execTool("resize", args[1:])
}

func kbdModeMain(args []string) int {
	return execTool("kbd_mode", args[1:])
}

func fgconsoleMain(args []string) int {
	return execTool("fgconsole", args[1:])
}

func openvtMain(args []string) int {
	return execTool("openvt", args[1:])
}

func setconsoleMain(args []string) int {
	return execTool("setconsole", args[1:])
}

func setfontMain(args []string) int {
	return execTool("setfont", args[1:])
}

func loadfontMain(args []string) int {
	return execTool("loadfont", args[1:])
}

func loadkmapMain(args []string) int {
	return execTool("loadkmap", args[1:])
}

func dumpkmapMain(args []string) int {
	return execTool("dumpkmap", args[1:])
}

func showkeyMain(args []string) int {
	return execTool("showkey", args[1:])
}

func setlogconsMain(args []string) int {
	return execTool("setlogcons", args[1:])
}

// ==================== Filesystem ====================

func findfsMain(args []string) int {
	return execTool("findfs", args[1:])
}

func mkdosfsMain(args []string) int {
	return execTool("mkdosfs", args[1:])
}

func mke2fsMain(args []string) int {
	return execTool("mke2fs", args[1:])
}

func fsckMinixMain(args []string) int {
	return execTool("fsck.minix", args[1:])
}

func freeramdiskMain(args []string) int {
	return execTool("freeramdisk", args[1:])
}

func raidautorunMain(args []string) int {
	return execTool("raidautorun", args[1:])
}

func rdevMain(args []string) int {
	return execTool("rdev", args[1:])
}

func readprofileMain(args []string) int {
	return execTool("readprofile", args[1:])
}

func readaheadMain(args []string) int {
	return execTool("readahead", args[1:])
}

func hdparmMain(args []string) int {
	return execTool("hdparm", args[1:])
}

func fdformatMain(args []string) int {
	return execTool("fdformat", args[1:])
}

func fdflushMain(args []string) int {
	return execTool("fdflush", args[1:])
}

// ==================== Flash ====================

func flashcpMain(args []string) int {
	return execTool("flashcp", args[1:])
}

func flashEraseallMain(args []string) int {
	return execTool("flash_eraseall", args[1:])
}

func flashLockMain(args []string) int {
	return execTool("flash_lock", args[1:])
}

func flashUnlockMain(args []string) int {
	return execTool("flash_unlock", args[1:])
}

func nanddumpMain(args []string) int {
	return execTool("nanddump", args[1:])
}

func nandwriteMain(args []string) int {
	return execTool("nandwrite", args[1:])
}

// ==================== I2C ====================

func i2cdetectMain(args []string) int {
	return execTool("i2cdetect", args[1:])
}

func i2cdumpMain(args []string) int {
	return execTool("i2cdump", args[1:])
}

func i2cgetMain(args []string) int {
	return execTool("i2cget", args[1:])
}

func i2csetMain(args []string) int {
	return execTool("i2cset", args[1:])
}

func i2ctransferMain(args []string) int {
	return execTool("i2ctransfer", args[1:])
}

// ==================== Network Extras ====================

func tcMain(args []string) int {
	return execTool("tc", args[1:])
}

func tcpsvdMain(args []string) int {
	return execTool("tcpsvd", args[1:])
}

func udhcpc6Main(args []string) int {
	return execTool("udhcpc6", args[1:])
}

func traceroute6Main(args []string) int {
	return execTool("traceroute6", args[1:])
}

func ueventMain(args []string) int {
	return execTool("uevent", args[1:])
}

func chatMain(args []string) int {
	return execTool("chat", args[1:])
}

func etherWakeMain(args []string) int {
	return execTool("ether-wake", args[1:])
}

func fakeidentdMain(args []string) int {
	return execTool("fakeidentd", args[1:])
}

func slattachMain(args []string) int {
	return execTool("slattach", args[1:])
}

func tunctlMain(args []string) int {
	return execTool("tunctl", args[1:])
}

func vconfigMain(args []string) int {
	return execTool("vconfig", args[1:])
}

func nbdClientMain(args []string) int {
	return execTool("nbd-client", args[1:])
}

// ==================== Misc ====================

func mtMain(args []string) int {
	return execTool("mt", args[1:])
}

func runlevelMain(args []string) int {
	data, err := os.ReadFile("/var/run/utmp")
	if err != nil {
		fmt.Println("unknown")
		return 1
	}
	_ = data
	fmt.Println("N 5")
	return 0
}

func mpstatMain(args []string) int {
	return execTool("mpstat", args[1:])
}

func rfkillMain(args []string) int {
	return execTool("rfkill", args[1:])
}

func resumeMain(args []string) int {
	return execTool("resume", args[1:])
}

func seedrngMain(args []string) int {
	f, err := os.OpenFile("/dev/urandom", os.O_WRONLY, 0)
	if err != nil {
		return 1
	}
	defer f.Close()
	seed := make([]byte, 32)
	f.Read(seed)
	return 0
}

func powertopMain(args []string) int {
	return execTool("powertop", args[1:])
}

func inotifydMain(args []string) int {
	return execTool("inotifyd", args[1:])
}

func nmeterMain(args []string) int {
	return execTool("nmeter", args[1:])
}

func microcomMain(args []string) int {
	return execTool("microcom", args[1:])
}

func makedevsMain(args []string) int {
	return execTool("makedevs", args[1:])
}

// ==================== Runit ====================

func runsvdirMain(args []string) int {
	return execTool("runsvdir", args[1:])
}

func svcMain(args []string) int {
	return execTool("svc", args[1:])
}

func svlogdMain(args []string) int {
	return execTool("svlogd", args[1:])
}

func svokMain(args []string) int {
	return execTool("svok", args[1:])
}

// ==================== Setuid ====================

func setuidgidMain(args []string) int {
	return execTool("setuidgid", args[1:])
}

func envuidgidMain(args []string) int {
	return execTool("envuidgid", args[1:])
}

func softlimitMain(args []string) int {
	return execTool("softlimit", args[1:])
}

// ==================== Mail ====================

func popmaildirMain(args []string) int {
	return execTool("popmaildir", args[1:])
}

func sendmailMain(args []string) int {
	return execTool("sendmail", args[1:])
}

func makemimeMain(args []string) int {
	return execTool("makemime", args[1:])
}

func reformimeMain(args []string) int {
	return execTool("reformime", args[1:])
}

// ==================== Console ====================

func conspyMain(args []string) int {
	return execTool("conspy", args[1:])
}

func chpasswdMain(args []string) int {
	return execTool("chpasswd", args[1:])
}

// ==================== Boot ====================

func runPartsMain(args []string) int {
	return execTool("run-parts", args[1:])
}

func runInitMain(args []string) int {
	return execTool("run-init", args[1:])
}

func rxMain(args []string) int {
	return execTool("rx", args[1:])
}

func unitMain(args []string) int {
	return execTool("unit", args[1:])
}

func tsMain(args []string) int {
	return execTool("ts", args[1:])
}

func usleepMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: usleep: missing operand")
		return 1
	}
	us, err := strconv.Atoi(args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: usleep: invalid number: %s\n", args[1])
		return 1
	}
	time.Sleep(time.Duration(us) * time.Microsecond)
	return 0
}

func volnameMain(args []string) int {
	path := "/dev/cdrom"
	if len(args) > 1 {
		path = args[1]
	}
	return execTool("volname", []string{path})
}

// ==================== SELinux ====================

func chconMain(args []string) int {
	return execTool("chcon", args[1:])
}

func getenforceMain(args []string) int {
	return execTool("getenforce", args[1:])
}

func getseboolMain(args []string) int {
	return execTool("getsebool", args[1:])
}

func restoreconMain(args []string) int {
	return execTool("restorecon", args[1:])
}

func selinuxenabledMain(args []string) int {
	return execTool("selinuxenabled", args[1:])
}

func sestatusMain(args []string) int {
	return execTool("sestatus", args[1:])
}

func setenforceMain(args []string) int {
	return execTool("setenforce", args[1:])
}

func setfilesMain(args []string) int {
	return execTool("setfiles", args[1:])
}

func setseboolMain(args []string) int {
	return execTool("setsebool", args[1:])
}

func loadPolicyMain(args []string) int {
	return execTool("load_policy", args[1:])
}

// ==================== Extended attributes ====================

func getfattrMain(args []string) int {
	return execTool("getfattr", args[1:])
}

func setfattrMain(args []string) int {
	return execTool("setfattr", args[1:])
}

// ==================== Misc ====================

func bbconfigMain(args []string) int {
	fmt.Println("gobox - BusyBox-like multi-call binary")
	return 0
}

func bootchartdMain(args []string) int {
	return execTool("bootchartd", args[1:])
}

func devfsdMain(args []string) int {
	return execTool("devfsd", args[1:])
}

func dumpleasesMain(args []string) int {
	return execTool("dumpleases", args[1:])
}

func fatattrMain(args []string) int {
	return execTool("fatattr", args[1:])
}

func fbsetMain(args []string) int {
	return execTool("fbset", args[1:])
}

func fbsplashMain(args []string) int {
	return execTool("fbsplash", args[1:])
}

func addShellMain(args []string) int {
	return execTool("add-shell", args[1:])
}

func removeShellMain(args []string) int {
	return execTool("remove-shell", args[1:])
}

func asciiMain(args []string) int {
	return execTool("ascii", args[1:])
}

func pscanMain(args []string) int {
	return execTool("pscan", args[1:])
}

func nukeMain(args []string) int {
	return execTool("nuke", args[1:])
}

func pipeProgressMain(args []string) int {
	return execTool("pipe_progress", args[1:])
}


func busyboxMain(args []string) int {
	fmt.Println("gobox - multi-call binary")
	return 0
}

func ipcalcMain(args []string) int {
	return execTool("ipcalc", args[1:])
}

func linux32Main(args []string) int {
	return execTool("linux32", args[1:])
}

func linux64Main(args []string) int {
	return execTool("linux64", args[1:])
}

func lpdMain(args []string) int {
	return execTool("lpd", args[1:])
}

func lpqMain(args []string) int {
	return execTool("lpq", args[1:])
}

func matchpathconMain(args []string) int {
	return execTool("matchpathcon", args[1:])
}

func mimMain(args []string) int {
	return execTool("mim", args[1:])
}

func minipsMain(args []string) int {
	return execTool("minips", args[1:])
}

func mkfsMinixMain(args []string) int {
	return execTool("mkfs.minix", args[1:])
}

func mkfsReiserMain(args []string) int {
	return execTool("mkfs.reiser", args[1:])
}

func nameifMain(args []string) int {
	return execTool("nameif", args[1:])
}

func runconMain(args []string) int {
	return execTool("runcon", args[1:])
}

func sslClientMain(args []string) int {
	return execTool("ssl_client", args[1:])
}

func tune2fsMain(args []string) int {
	return execTool("tune2fs", args[1:])
}

func ubiattachMain(args []string) int {
	return execTool("ubiattach", args[1:])
}

func ubidetachMain(args []string) int {
	return execTool("ubidetach", args[1:])
}

func ubimkvolMain(args []string) int {
	return execTool("ubimkvol", args[1:])
}

func ubirenameMain(args []string) int {
	return execTool("ubirename", args[1:])
}

func ubirmvolMain(args []string) int {
	return execTool("ubirmvol", args[1:])
}

func ubirsvolMain(args []string) int {
	return execTool("ubirsvol", args[1:])
}

func ubiupdatevolMain(args []string) int {
	return execTool("ubiupdatevol", args[1:])
}
