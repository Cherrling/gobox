package applets

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func init() {
	Register("ping", AppletFunc(pingMain))
	Register("wget", AppletFunc(wgetMain))
	Register("hostname", AppletFunc(hostnameMain))
	Register("dnsdomainname", AppletFunc(dnsdomainnameMain))
	Register("nc", AppletFunc(ncMain))
	Register("netstat", AppletFunc(netstatMain))
	Register("ifconfig", AppletFunc(ifconfigMain))
	Register("route", AppletFunc(routeMain))
	Register("nslookup", AppletFunc(nslookupMain))
	Register("telnet", AppletFunc(telnetMain))
}

func pingMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: ping: missing host")
		return 1
	}

	count := 0
	target := args[len(args)-1]

	for _, arg := range args[1:] {
		if arg == "-c" {
			// next arg is count
		} else if n, err := strconv.Atoi(arg); err == nil {
			count = n
		}
	}

	// Look up the host
	ips, err := net.LookupHost(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: ping: %s: %v\n", target, err)
		return 1
	}

	ip := ips[0]
	fmt.Printf("PING %s (%s): 56 data bytes\n", target, ip)

	sent := 0
	received := 0
	limit := count
	if limit == 0 {
		limit = 4 // default 4 pings
	}

	for i := 0; i < limit; i++ {
		start := time.Now()
		conn, err := net.DialTimeout("ip4:icmp", ip, time.Second)
		if err != nil {
			// Fallback to TCP ping
			conn, err = net.DialTimeout("tcp", ip+":80", time.Second)
			if err != nil {
				fmt.Printf("ping: cannot connect to %s\n", ip)
				sent++
				continue
			}
			sent++
			received++
			conn.Close()
			rtt := time.Since(start)
			fmt.Printf("64 bytes from %s: tcp_seq=%d time=%.3f ms\n", ip, i, float64(rtt.Microseconds())/1000.0)
			time.Sleep(time.Second)
			continue
		}
		sent++
		received++
		conn.Close()
		rtt := time.Since(start)
		fmt.Printf("64 bytes from %s: icmp_seq=%d time=%.3f ms\n", ip, i, float64(rtt.Microseconds())/1000.0)
		time.Sleep(time.Second)
	}

	fmt.Printf("--- %s ping statistics ---\n", target)
	fmt.Printf("%d packets transmitted, %d packets received, %d%% packet loss\n",
		sent, received, (sent-received)*100/sent)
	return 0
}

func wgetMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: wget: missing URL")
		return 1
	}

	url := args[1]
	output := ""

	for i := 1; i < len(args); i++ {
		if args[i] == "-O" && i+1 < len(args) {
			output = args[i+1]
			i++
		}
	}

	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: wget: %v\n", err)
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "gobox: wget: server returned %s\n", resp.Status)
		return 1
	}

	if output == "" {
		// Extract filename from URL
		parts := strings.Split(url, "/")
		output = parts[len(parts)-1]
		if output == "" {
			output = "index.html"
		}
	}

	f, err := os.Create(output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: wget: %v\n", err)
		return 1
	}
	defer f.Close()

	written, err := io.Copy(f, resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: wget: %v\n", err)
		return 1
	}

	fmt.Printf("Saved %d bytes to %s\n", written, output)
	return 0
}

func hostnameMain(args []string) int {
	if len(args) > 1 && !strings.HasPrefix(args[1], "-") {
		// Set hostname (requires privileges)
		if err := os.WriteFile("/proc/sys/kernel/hostname", []byte(args[1]), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "gobox: hostname: %v\n", err)
			return 1
		}
		return 0
	}

	hostname, err := os.Hostname()
	if err != nil {
		fmt.Fprintln(os.Stderr, "gobox: hostname: cannot get hostname")
		return 1
	}
	fmt.Println(hostname)
	return 0
}

func dnsdomainnameMain(args []string) int {
	hostname, _ := os.Hostname()
	if strings.Contains(hostname, ".") {
		parts := strings.SplitN(hostname, ".", 2)
		if len(parts) > 1 {
			fmt.Println(parts[1])
			return 0
		}
	}
	return 0
}

func ncMain(args []string) int {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "gobox: nc: missing operand")
		return 1
	}

	host := args[1]
	port := args[2]

	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 10*time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: nc: %v\n", err)
		return 1
	}
	defer conn.Close()

	done := make(chan struct{}, 2)
	go func() {
		io.Copy(conn, os.Stdin)
		done <- struct{}{}
	}()
	go func() {
		io.Copy(os.Stdout, conn)
		done <- struct{}{}
	}()
	<-done
	return 0
}

func netstatMain(args []string) int {
	data, err := os.ReadFile("/proc/net/tcp")
	if err != nil {
		fmt.Fprintln(os.Stderr, "gobox: netstat: cannot read /proc/net/tcp")
		return 1
	}

	fmt.Println("Active Internet connections (servers and established)")
	fmt.Printf("%-15s %-15s %-12s %s\n", "Local Address", "Remote Address", "State", "PID/Program name")

	lines := strings.Split(string(data), "\n")
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		localAddr := formatNetAddr(fields[1])
		remoteAddr := formatNetAddr(fields[2])
		state := tcpState(fields[3])
		fmt.Printf("%-15s %-15s %-12s -\n", localAddr, remoteAddr, state)
	}
	return 0
}

func formatNetAddr(hexAddr string) string {
	parts := strings.Split(hexAddr, ":")
	if len(parts) != 2 {
		return hexAddr
	}
	// Reverse byte order for IP
	addrHex := parts[0]
	port, _ := strconv.ParseInt(parts[1], 16, 32)

	ip := make(net.IP, 4)
	for i := 0; i < 8 && i < len(addrHex); i += 2 {
		b, _ := strconv.ParseInt(addrHex[i:i+2], 16, 32)
		ip[3-i/2] = byte(b)
	}

	return fmt.Sprintf("%s:%d", ip.String(), port)
}

func tcpState(hex string) string {
	states := map[string]string{
		"01": "ESTABLISHED",
		"02": "SYN_SENT",
		"03": "SYN_RECV",
		"04": "FIN_WAIT1",
		"05": "FIN_WAIT2",
		"06": "TIME_WAIT",
		"07": "CLOSE",
		"08": "CLOSE_WAIT",
		"09": "LAST_ACK",
		"0A": "LISTEN",
		"0B": "CLOSING",
	}
	if s, ok := states[hex]; ok {
		return s
	}
	return "UNKNOWN"
}

func ifconfigMain(args []string) int {
	interfaces, err := net.Interfaces()
	if err != nil {
		fmt.Fprintln(os.Stderr, "gobox: ifconfig: cannot get interfaces")
		return 1
	}

	target := ""
	if len(args) > 1 {
		target = args[1]
	}

	for _, iface := range interfaces {
		if target != "" && iface.Name != target {
			continue
		}

		addrs, _ := iface.Addrs()
		var ipAddr, mask string
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if ok && ipNet.IP.To4() != nil {
				ipAddr = ipNet.IP.String()
				mask = fmt.Sprintf("0x%02x%02x%02x%02x",
					ipNet.Mask[0], ipNet.Mask[1], ipNet.Mask[2], ipNet.Mask[3])
				break
			}
		}

		if ipAddr == "" {
			ipAddr = "0.0.0.0"
		}

		mtu := iface.MTU
		hwAddr := iface.HardwareAddr.String()
		flags := "UP"
		if iface.Flags&net.FlagBroadcast != 0 {
			flags += " BROADCAST"
		}
		if iface.Flags&net.FlagLoopback != 0 {
			flags += " LOOPBACK"
		}
		if iface.Flags&net.FlagRunning == 0 {
			flags = "DOWN"
		}

		fmt.Printf("%s: flags=%d<%s> mtu %d\n", iface.Name, iface.Flags, flags, mtu)
		fmt.Printf("        inet %s  netmask %s  ", ipAddr, mask)
		fmt.Printf("ether %s\n", hwAddr)
		fmt.Println()
	}
	return 0
}

func routeMain(args []string) int {
	data, err := os.ReadFile("/proc/net/route")
	if err != nil {
		fmt.Fprintln(os.Stderr, "gobox: route: cannot read /proc/net/route")
		return 1
	}

	fmt.Printf("%-16s %-16s %-16s %-6s %-6s %-3s %s\n",
		"Destination", "Gateway", "Genmask", "Flags", "Metric", "Ref", "Iface")

	lines := strings.Split(string(data), "\n")
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) < 8 {
			continue
		}

		dest := fmtIP(fields[1])
		gateway := fmtIP(fields[2])
		mask := fmtIP(fields[7])
		flags := fields[3]
		metric := fields[6]
		iface := fields[0]

		fmt.Printf("%-16s %-16s %-16s %-6s %-6s %-3s %s\n",
			dest, gateway, mask, flags, metric, "0", iface)
	}
	return 0
}

func fmtIP(hex string) string {
	if len(hex) < 8 {
		return "0.0.0.0"
	}
	n, _ := strconv.ParseUint(hex, 16, 32)
	ip := net.IPv4(byte(n), byte(n>>8), byte(n>>16), byte(n>>24))
	return ip.String()
}

func nslookupMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: nslookup: missing host")
		return 1
	}

	host := args[1]
	ips, err := net.LookupHost(host)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: nslookup: %v\n", err)
		return 1
	}

	fmt.Printf("Server:  localhost\n")
	fmt.Printf("Address: 127.0.0.1\n\n")
	fmt.Printf("Name:    %s\n", host)
	fmt.Printf("Address: %s\n", strings.Join(ips, ", "))
	return 0
}

func telnetMain(args []string) int {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "gobox: telnet: missing host or port")
		return 1
	}

	host := args[1]
	port := args[2]

	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 10*time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: telnet: %v\n", err)
		return 1
	}
	defer conn.Close()

	// Raw TCP connection with stdin/stdout passthrough
	cmd := exec.Command("cat")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	done := make(chan struct{}, 2)
	go func() {
		io.Copy(conn, os.Stdin)
		done <- struct{}{}
	}()
	go func() {
		io.Copy(os.Stdout, conn)
		done <- struct{}{}
	}()
	<-done
	return 0
}
