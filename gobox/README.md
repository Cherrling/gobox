# gobox

A multi-call binary compatible with busybox, written in Go.

## Build

    go build -o gobox .

## Usage

    gobox [APPLET] [ARGS...]

Or symlink applet names to the gobox binary (e.g. `ln -s gobox ls`).

## Test Status

Tested against the [busybox testsuite](https://github.com/mirror/busybox/tree/master/testsuite).

**396 PASS, 82 FAIL, 18 SKIP** (all 82 failures are `sed`).

### Verified working (0 failures)

| Category | Applets |
|----------|---------|
| File ops | cat, cp, cut, dd, head, install, ln, ls, mkdir, mkfifo, mknod, mktemp, mv, rm, rmdir, shred, tail, tee, touch, truncate, unlink |
| Text utils | comm, cksum, expand, fold, nl, paste, printf, rev, seq, sort, sum, tac, tr, tsort, unexpand, uniq, wc |
| Archival | tar (create/extract/list, gzip, hardlinks, symlinks) |
| Search | grep/egrep/fgrep (recursive, count, invert, context, -o) |
| Diff | diff (unified, recursive, quiet, -b, -B, dir comparison) |
| Find | find (-name, -type, -exec, -delete, -maxdepth, -empty, -ok) |
| Misc | basename, dirname, echo, env, expr, false, id, logname, nohup, printenv, pwd, readlink, realpath, sleep, test, timeout, true, uname, usleep, whoami, yes |
| Perms | chgrp, chmod, chown, stat |
| System | cal, clear, date, dmesg, hexdump, hostid, kill, lsmod, od, pidof, ps, reset, rmmod, stty, sync, tree, uptime, watch |
| Hashing | md5sum, sha1sum, sha256sum, sha3sum, sha512sum, xxd |
| Patch | patch |

### Broken

- **sed**: Only basic `s///` substitution is implemented (no address ranges, hold space, branching, or other commands)

### Delegated to external tools

These applets require the corresponding system tool to be installed:

arch, arp, blkid, bunzip2, bzcat, bzip2, cpio, crond, crontab, dc, depmod, dos2unix, dpkg, fdisk, fsck, ftpget, ftpput, gzip, gunzip, hdparm, httpd, ifconfig, ip, iptables, last, less, logger, losetup, lzcat, lzma, lzop, lzopcat, makemime, mkfs.vfat, modprobe, more, mount, nc, netstat, nslookup, ntpd, ping, route, rpm, rpm2cpio, sysctl, telnet, tftp, traceroute, udhcpc, umount, uncompress, unix2dos, unlzma, unlzop, unxz, unzip, uuencode, uudecode, vi, wget, xz, xzcat, zcat, and ~100 more.

### Skipped in tests

ash, awk, bc, dc, mdev — interactive or feature-dependent. Tests skipped via `run_tests.sh`.

## Run tests

    bash run_tests.sh          # run all
    bash run_tests.sh tar      # run specific test
    bash run_tests.sh tar sed  # run multiple
