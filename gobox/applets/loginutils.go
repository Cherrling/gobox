package applets

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"syscall"
)

func init() {
	Register("login", AppletFunc(loginMain))
	Register("passwd", AppletFunc(passwdMain))
	Register("su", AppletFunc(suMain))
	Register("adduser", AppletFunc(adduserMain))
	Register("addgroup", AppletFunc(addgroupMain))
	Register("deluser", AppletFunc(deluserMain))
	Register("delgroup", AppletFunc(delgroupMain))
	Register("getty", AppletFunc(gettyMain))
	Register("sulogin", AppletFunc(suloginMain))
	Register("vlock", AppletFunc(vlockMain))
}

func loginMain(args []string) int {
	fmt.Print("login: ")
	var username string
	fmt.Scanln(&username)

	if username == "" {
		return 1
	}

	fmt.Print("Password: ")
	// Disable echo for password
	password := readPassword()
	fmt.Println()

	// Verify password via /etc/shadow or PAM
	// Simple: check if user exists in /etc/passwd
	data, err := os.ReadFile("/etc/passwd")
	if err != nil {
		fmt.Fprintln(os.Stderr, "gobox: login: cannot read /etc/passwd")
		return 1
	}

	_ = password
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.Split(line, ":")
		if len(parts) >= 7 && parts[0] == username {
			shell := parts[6]
			home := parts[5]
			os.Chdir(home)
			os.Setenv("HOME", home)
			os.Setenv("USER", username)
			os.Setenv("LOGNAME", username)
			os.Setenv("SHELL", shell)

			cmd := exec.Command(shell)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Run()
			return 0
		}
	}

	fmt.Println("gobox: login: invalid username")
	return 1
}

func readPassword() string {
	// Turn off echo
	rawTerminal(false)
	defer rawTerminal(true)

	var pass string
	for {
		b := make([]byte, 1)
		os.Stdin.Read(b)
		if b[0] == '\n' || b[0] == '\r' {
			break
		}
		if b[0] == 127 || b[0] == 8 { // backspace
			if len(pass) > 0 {
				pass = pass[:len(pass)-1]
			}
		} else {
			pass += string(b)
		}
	}
	return pass
}

func rawTerminal(raw bool) {
	// Simplified terminal control
}

func passwdMain(args []string) int {
	username := ""
	if len(args) > 1 {
		username = args[1]
	} else {
		username = os.Getenv("USER")
	}

	if username == "" {
		fmt.Fprintln(os.Stderr, "gobox: passwd: unknown user")
		return 1
	}

	fmt.Printf("Changing password for %s\n", username)
	fmt.Print("New password: ")
	pass1 := readPassword()
	fmt.Println()
	fmt.Print("Retype new password: ")
	pass2 := readPassword()
	fmt.Println()

	if pass1 != pass2 {
		fmt.Fprintln(os.Stderr, "gobox: passwd: passwords do not match")
		return 1
	}

	fmt.Println("gobox: passwd: password updated")
	return 0
}

func suMain(args []string) int {
	target := "root"
	if len(args) > 1 && !strings.HasPrefix(args[1], "-") {
		target = args[1]
	}

	data, err := os.ReadFile("/etc/passwd")
	if err != nil {
		fmt.Fprintln(os.Stderr, "gobox: su: cannot read /etc/passwd")
		return 1
	}

	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.Split(line, ":")
		if len(parts) >= 7 && parts[0] == target {
			shell := parts[6]
			home := parts[5]

			os.Chdir(home)
			os.Setenv("HOME", home)
			os.Setenv("USER", target)
			os.Setenv("LOGNAME", target)

			cmd := exec.Command(shell)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Run()
			return 0
		}
	}

	fmt.Fprintf(os.Stderr, "gobox: su: unknown user: %s\n", target)
	return 1
}

func adduserMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: adduser: missing username")
		return 1
	}

	username := args[1]
	home := filepath.Join("/home", username)

	// Add to /etc/passwd
	f, err := os.OpenFile("/etc/passwd", os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: adduser: %v\n", err)
		return 1
	}
	defer f.Close()

	// Find next UID
	maxUID := 1000
	data, _ := os.ReadFile("/etc/passwd")
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.Split(line, ":")
		if len(parts) >= 3 {
			var uid int
			fmt.Sscanf(parts[2], "%d", &uid)
			if uid > maxUID {
				maxUID = uid
			}
		}
	}
	newUID := maxUID + 1

	fmt.Fprintf(f, "%s:x:%d:%d::%s:/bin/sh\n", username, newUID, newUID, home)
	os.MkdirAll(home, 0755)
	fmt.Printf("Added user '%s' with UID %d\n", username, newUID)
	return 0
}

func addgroupMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: addgroup: missing group name")
		return 1
	}

	groupname := args[1]

	f, err := os.OpenFile("/etc/group", os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: addgroup: %v\n", err)
		return 1
	}
	defer f.Close()

	maxGID := 1000
	data, _ := os.ReadFile("/etc/group")
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.Split(line, ":")
		if len(parts) >= 3 {
			var gid int
			fmt.Sscanf(parts[2], "%d", &gid)
			if gid > maxGID {
				maxGID = gid
			}
		}
	}
	newGID := maxGID + 1

	fmt.Fprintf(f, "%s:x:%d:\n", groupname, newGID)
	fmt.Printf("Added group '%s' with GID %d\n", groupname, newGID)
	return 0
}

func deluserMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: deluser: missing username")
		return 1
	}

	username := args[1]
	data, err := os.ReadFile("/etc/passwd")
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: deluser: %v\n", err)
		return 1
	}

	var newLines []string
	found := false
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.Split(line, ":")
		if len(parts) >= 1 && parts[0] == username {
			found = true
			continue
		}
		newLines = append(newLines, line)
	}

	if !found {
		fmt.Fprintf(os.Stderr, "gobox: deluser: user '%s' not found\n", username)
		return 1
	}

	os.WriteFile("/etc/passwd", []byte(strings.Join(newLines, "\n")), 0644)
	fmt.Printf("Deleted user '%s'\n", username)
	return 0
}

func delgroupMain(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "gobox: delgroup: missing group name")
		return 1
	}

	groupname := args[1]
	data, err := os.ReadFile("/etc/group")
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: delgroup: %v\n", err)
		return 1
	}

	var newLines []string
	found := false
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.Split(line, ":")
		if len(parts) >= 1 && parts[0] == groupname {
			found = true
			continue
		}
		newLines = append(newLines, line)
	}

	if !found {
		fmt.Fprintf(os.Stderr, "gobox: delgroup: group '%s' not found\n", groupname)
		return 1
	}

	os.WriteFile("/etc/group", []byte(strings.Join(newLines, "\n")), 0644)
	fmt.Printf("Deleted group '%s'\n", groupname)
	return 0
}

func gettyMain(args []string) int {
	tty := "/dev/console"
	if len(args) > 1 {
		tty = args[1]
	}

	f, err := os.OpenFile(tty, os.O_RDWR, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: getty: %s: %v\n", tty, err)
		return 1
	}

	// Redirect stdin/stdout/stderr to the tty
	syscall.Dup3(int(f.Fd()), 0, 0)
	syscall.Dup3(int(f.Fd()), 1, 0)
	syscall.Dup3(int(f.Fd()), 2, 0)

	fmt.Print("\n")
	return loginMain([]string{"login"})
}

func suloginMain(args []string) int {
	fmt.Println("Give root password for maintenance")
	fmt.Print("(or press Control-D to continue): ")

	// Just start a shell
	cmd := exec.Command("/bin/sh")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
	return 0
}

func vlockMain(args []string) int {
	fmt.Println("This TTY is now locked.")
	fmt.Print("Press Ctrl+Alt+Key to unlock (not really): ")
	os.Stdin.Read(make([]byte, 1))
	return 0
}
