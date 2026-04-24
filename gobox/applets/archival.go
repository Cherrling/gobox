package applets

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func init() {
	Register("tar", AppletFunc(tarMain))
	Register("gzip", AppletFunc(gzipMain))
	Register("gunzip", AppletFunc(gunzipMain))
	Register("zcat", AppletFunc(zcatMain))
	Register("bunzip2", AppletFunc(bunzip2Main))
	Register("bzcat", AppletFunc(bzcatMain))
	Register("bzip2", AppletFunc(bzip2Main))
	Register("unlzma", AppletFunc(unlzmaMain))
	Register("lzcat", AppletFunc(lzcatMain))
	Register("lzma", AppletFunc(lzmaMain))
	Register("unxz", AppletFunc(unxzMain))
	Register("xzcat", AppletFunc(xzcatMain))
	Register("xz", AppletFunc(xzMain))
	Register("uncompress", AppletFunc(uncompressMain))
	Register("cpio", AppletFunc(cpioMain))
	Register("ar", AppletFunc(arMain))
	Register("unzip", AppletFunc(unzipMain))
	Register("lzop", AppletFunc(lzopMain))
	Register("unlzop", AppletFunc(unlzopMain))
	Register("lzopcat", AppletFunc(lzopcatMain))
	Register("rpm2cpio", AppletFunc(rpm2cpioMain))
}

func tarMain(args []string) int {
	create := false
	extract := false
	list := false
	compress := false
	verbose := false
	file := ""
	dir := ""
	paths := []string{}

	for i := 1; i < len(args); i++ {
		arg := args[i]
		flags := arg
		if strings.HasPrefix(flags, "-") {
			flags = flags[1:]
		}
		if len(flags) > 0 && strings.IndexFunc(flags, func(r rune) bool {
			return r != 'c' && r != 'x' && r != 't' && r != 'z' && r != 'v' && r != 'f' && r != 'C'
		}) == -1 {
			for _, c := range flags {
				switch c {
				case 'c':
					create = true
				case 'x':
					extract = true
				case 't':
					list = true
				case 'z':
					compress = true
				case 'v':
					verbose = true
				case 'f':
					if i+1 < len(args) {
						file = args[i+1]
						i++
					}
				case 'C':
					if i+1 < len(args) {
						dir = args[i+1]
						i++
					}
				}
			}
		} else {
			paths = append(paths, arg)
		}
	}

	if create {
		return tarCreate(file, paths, compress)
	}
	if extract {
		return tarExtract(file, dir, compress, verbose)
	}
	if list {
		return tarList(file, compress)
	}

	fmt.Fprintln(os.Stderr, "tar: need one of -c, -x, -t")
	return 1
}

func tarCreate(tarFile string, paths []string, compress bool) int {
	if len(paths) == 0 {
		fmt.Fprintln(os.Stderr, "tar: no files to archive")
		return 1
	}

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	type inodeKey struct {
		dev uint64
		ino uint64
	}
	seenInodes := make(map[inodeKey]string)

	for _, path := range paths {
		cleanPath, prefix := sanitizeTarPath(path)
		if prefix != "" {
			fmt.Fprintf(os.Stderr, "tar: removing leading '%s' from member names\n", prefix)
		}
		path = cleanPath

		err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			relPath := filePath
			relPath = strings.TrimPrefix(relPath, "./")

			var linkTarget string
			var header *tar.Header

			isHardlink := false
			if !info.IsDir() {
				if stat, ok := info.Sys().(*syscall.Stat_t); ok && stat.Nlink > 1 {
					key := inodeKey{dev: stat.Dev, ino: stat.Ino}
					if firstPath, seen := seenInodes[key]; seen {
						isHardlink = true
						header, err = tar.FileInfoHeader(info, "")
						if err != nil {
							return err
						}
						header.Typeflag = tar.TypeLink
						header.Linkname = firstPath
						header.Size = 0
					} else {
						seenInodes[key] = relPath
					}
				}
			}

			if !isHardlink {
				if info.Mode()&os.ModeSymlink != 0 {
					linkTarget, err = os.Readlink(filePath)
					if err != nil {
						return err
					}
					header, err = tar.FileInfoHeader(info, linkTarget)
					if err != nil {
						return err
					}
				} else if info.Mode().IsRegular() {
					header, err = tar.FileInfoHeader(info, "")
					if err != nil {
						return err
					}
				} else {
					header, err = tar.FileInfoHeader(info, "")
					if err != nil {
						return err
					}
				}
			}

			header.Name = relPath

			if info.IsDir() && !strings.HasSuffix(header.Name, "/") {
				header.Name += "/"
			}

			if err := tw.WriteHeader(header); err != nil {
				return err
			}

			if info.Mode().IsRegular() && header.Typeflag != tar.TypeLink {
				f, err := os.Open(filePath)
				if err != nil {
					return err
				}
				defer f.Close()
				io.Copy(tw, f)
			}
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "tar: %v\n", err)
			return 1
		}
	}
	tw.Close()

	if tarFile != "" && tarFile != "-" {
		mode := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
		f, err := os.OpenFile(tarFile, mode, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "tar: %v\n", err)
			return 1
		}
		defer f.Close()
		f.Write(buf.Bytes())
	} else {
		os.Stdout.Write(buf.Bytes())
	}

	return 0
}

func sanitizeTarPath(path string) (string, string) {
	orig := path
	path = strings.TrimPrefix(path, "./")

	lastIdx := -1
	for i := 0; i <= len(path)-2; i++ {
		if path[i] == '.' && path[i+1] == '.' {
			if i == 0 && (len(path) == 2 || path[i+2] == '/') {
				lastIdx = i
			} else if i > 0 && path[i-1] == '/' && (i+2 == len(path) || path[i+2] == '/') {
				lastIdx = i - 1
			}
		}
	}

	if lastIdx < 0 {
		return path, ""
	}

	end := lastIdx + 3
	if lastIdx > 0 && path[lastIdx] == '/' {
		end = lastIdx + 4
	}
	prefix := path[:end]
	path = path[end:]

	path = strings.TrimPrefix(path, "./")
	if path == "" {
		path = "."
	}

	prefixFromOrig := ""
	if idx := strings.Index(orig, prefix); idx >= 0 {
		prefixFromOrig = orig[:idx+len(prefix)]
	}
	return path, prefixFromOrig
}

func peekAndRead(r io.Reader) (io.Reader, error) {
	buf := make([]byte, 1024)
	n, err := io.ReadFull(r, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return nil, err
	}
	if n == 0 {
		return nil, fmt.Errorf("short read")
	}
	return io.MultiReader(bytes.NewReader(buf[:n]), r), nil
}

func tarExtract(tarFile string, dir string, compress bool, verbose bool) int {
	var r io.Reader

	if tarFile != "" && tarFile != "-" {
		f, err := os.Open(tarFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "tar: %v\n", err)
			return 1
		}
		defer f.Close()
		r = f
	} else {
		r = os.Stdin
	}

	peeked, err := peekAndRead(r)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tar: short read")
		return 1
	}
	r = peeked

	if compress {
		gr, err := gzip.NewReader(r)
		if err != nil {
			fmt.Fprintf(os.Stderr, "tar: %v\n", err)
			return 1
		}
		defer gr.Close()
		r = gr
	}

	tr := tar.NewReader(r)

	type dirPerm struct {
		path string
		mode os.FileMode
	}
	var dirs []dirPerm

	header, err := tr.Next()
	if err == io.EOF {
		return 0
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "tar: %v\n", err)
		return 1
	}

	for {
		target := header.Name
		if dir != "" {
			target = filepath.Join(dir, target)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(target, 0755)
			dirs = append(dirs, dirPerm{target, os.FileMode(header.Mode)})
			if verbose {
				fmt.Println(header.Name)
			}
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(target), 0755)
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				fmt.Fprintf(os.Stderr, "tar: %v\n", err)
				return 1
			}
			io.Copy(f, tr)
			f.Close()
			os.Chmod(target, os.FileMode(header.Mode))
			if verbose {
				fmt.Println(header.Name)
			}
		case tar.TypeSymlink:
			os.MkdirAll(filepath.Dir(target), 0755)
			os.Remove(target)
			if err := os.Symlink(header.Linkname, target); err != nil {
				fmt.Fprintf(os.Stderr, "tar: %v\n", err)
				return 1
			}
			if verbose {
				fmt.Println(header.Name)
			}
		case tar.TypeLink:
			linkTarget := header.Linkname
			if dir != "" {
				linkTarget = filepath.Join(dir, linkTarget)
			}
			if linkTarget == target {
				if verbose {
					fmt.Println(header.Name)
				}
				break
			}
			os.MkdirAll(filepath.Dir(target), 0755)
			os.Remove(target)
			if err := os.Link(linkTarget, target); err != nil {
				fmt.Fprintf(os.Stderr, "tar: %v\n", err)
				return 1
			}
			if verbose {
				fmt.Println(header.Name)
			}
		}

		header, err = tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "tar: %v\n", err)
			return 1
		}
	}

	for i := len(dirs) - 1; i >= 0; i-- {
		os.Chmod(dirs[i].path, dirs[i].mode)
	}
	return 0
}

func tarList(tarFile string, compress bool) int {
	var r io.Reader

	if tarFile != "" {
		f, err := os.Open(tarFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "tar: %v\n", err)
			return 1
		}
		defer f.Close()
		r = f
	} else {
		r = os.Stdin
	}

	peeked, err := peekAndRead(r)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tar: short read")
		return 1
	}
	r = peeked

	if compress {
		gr, err := gzip.NewReader(r)
		if err != nil {
			fmt.Fprintf(os.Stderr, "tar: %v\n", err)
			return 1
		}
		defer gr.Close()
		r = gr
	}

	tr := tar.NewReader(r)

	header, err := tr.Next()
	if err == io.EOF {
		return 0
	}
	if err != nil {
		return 1
	}

	for {
		name := header.Name
		switch header.Typeflag {
		case tar.TypeSymlink:
			fmt.Printf("%s -> %s\n", name, header.Linkname)
		case tar.TypeLink:
			fmt.Printf("%s -> %s\n", name, header.Linkname)
		default:
			fmt.Println(name)
		}

		header, err = tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 1
		}
	}
	return 0
}

func gzipMain(args []string) int {
	paths := args[1:]
	if len(paths) == 0 {
		gw := gzip.NewWriter(os.Stdout)
		io.Copy(gw, os.Stdin)
		gw.Close()
		return 0
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: gzip: %s: %v\n", path, err)
			return 1
		}

		var buf bytes.Buffer
		gw := gzip.NewWriter(&buf)
		gw.Write(data)
		gw.Close()

		outPath := path + ".gz"
		if err := os.WriteFile(outPath, buf.Bytes(), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "gobox: gzip: %s: %v\n", outPath, err)
			return 1
		}
	}
	return 0
}

func gunzipMain(args []string) int {
	paths := args[1:]
	if len(paths) == 0 {
		gr, err := gzip.NewReader(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, "gobox: gunzip: not in gzip format")
			return 1
		}
		defer gr.Close()
		io.Copy(os.Stdout, gr)
		return 0
	}

	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: gunzip: %s: %v\n", path, err)
			return 1
		}

		gr, err := gzip.NewReader(f)
		if err != nil {
			f.Close()
			fmt.Fprintf(os.Stderr, "gobox: gunzip: %s: not in gzip format\n", path)
			return 1
		}

		outPath := strings.TrimSuffix(path, ".gz")
		outPath = strings.TrimSuffix(outPath, ".tgz")
		if outPath == path {
			outPath = strings.TrimSuffix(path, ".Z")
		}

		outFile, err := os.Create(outPath)
		if err != nil {
			gr.Close()
			f.Close()
			fmt.Fprintf(os.Stderr, "gobox: gunzip: %v\n", err)
			return 1
		}

		io.Copy(outFile, gr)
		gr.Close()
		f.Close()
		outFile.Close()
	}
	return 0
}

func zcatMain(args []string) int {
	paths := args[1:]
	if len(paths) == 0 {
		gr, err := gzip.NewReader(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, "gobox: zcat: not in gzip format")
			return 1
		}
		defer gr.Close()
		io.Copy(os.Stdout, gr)
		return 0
	}

	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: zcat: %s: %v\n", path, err)
			return 1
		}

		gr, err := gzip.NewReader(f)
		if err != nil {
			f.Close()
			fmt.Fprintf(os.Stderr, "gobox: zcat: %s: not in gzip format\n", path)
			return 1
		}

		io.Copy(os.Stdout, gr)
		gr.Close()
		f.Close()
	}
	return 0
}

func bunzip2Main(args []string) int {
	return execTool("bunzip2", args[1:])
}

func bzcatMain(args []string) int {
	return execTool("bzcat", args[1:])
}

func bzip2Main(args []string) int {
	return execTool("bzip2", args[1:])
}

func unlzmaMain(args []string) int {
	return execTool("unlzma", args[1:])
}

func lzcatMain(args []string) int {
	return execTool("lzcat", args[1:])
}

func lzmaMain(args []string) int {
	return execTool("lzma", args[1:])
}

func unxzMain(args []string) int {
	return execTool("unxz", args[1:])
}

func xzcatMain(args []string) int {
	return execTool("xzcat", args[1:])
}

func xzMain(args []string) int {
	return execTool("xz", args[1:])
}

func uncompressMain(args []string) int {
	return execTool("uncompress", args[1:])
}

func cpioMain(args []string) int {
	return execTool("cpio", args[1:])
}

func arMain(args []string) int {
	return execTool("ar", args[1:])
}

func unzipMain(args []string) int {
	return execTool("unzip", args[1:])
}

func lzopMain(args []string) int {
	return execTool("lzop", args[1:])
}

func unlzopMain(args []string) int {
	return execTool("unlzop", args[1:])
}

func lzopcatMain(args []string) int {
	return execTool("lzopcat", args[1:])
}

func rpm2cpioMain(args []string) int {
	return execTool("rpm2cpio", args[1:])
}

func execTool(tool string, args []string) int {
	cmd := exec.Command(tool, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				return status.ExitStatus()
			}
		}
		return 1
	}
	return 0
}
