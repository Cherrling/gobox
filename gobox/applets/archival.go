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
	file := ""
	dir := ""
	paths := []string{}

	for i := 1; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			for _, c := range arg[1:] {
				switch c {
				case 'c':
					create = true
				case 'x':
					extract = true
				case 't':
					list = true
				case 'z':
					compress = true
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
		return tarExtract(file, dir, compress)
	}
	if list {
		return tarList(file, compress)
	}

	fmt.Fprintln(os.Stderr, "gobox: tar: need one of -c, -x, -t")
	return 1
}

func tarCreate(tarFile string, paths []string, compress bool) int {
	if len(paths) == 0 {
		fmt.Fprintln(os.Stderr, "gobox: tar: no files to archive")
		return 1
	}

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	for _, path := range paths {
		// Sanitize path: strip /../ components
		cleanPath := path
		sawDotDot := false
		for strings.Contains(cleanPath, "/../") || strings.HasPrefix(cleanPath, "../") {
			sawDotDot = true
			if strings.HasPrefix(cleanPath, "../") {
				cleanPath = cleanPath[3:]
			} else {
				idx := strings.Index(cleanPath, "/../")
				if idx >= 0 {
					// Remove the component before /../
					prevSlash := strings.LastIndex(cleanPath[:idx], "/")
					if prevSlash >= 0 {
						cleanPath = cleanPath[:prevSlash] + cleanPath[idx+3:]
					} else {
						cleanPath = cleanPath[idx+4:]
					}
				}
			}
		}
		if sawDotDot {
			fmt.Fprintf(os.Stderr, "tar: removing leading '%s' from member names\n", strings.Replace(path, cleanPath, "..", 1))
			// Actually, we need to extract the stripped prefix
			// The prefix is everything that was removed
			prefix := path
			if strings.HasPrefix(path, "./") {
				prefix = prefix[2:]
			}
			if strings.HasSuffix(cleanPath, "/") {
				cleanPath = cleanPath[:len(cleanPath)-1]
			}
			// Find the actual prefix that was removed
			if idx := strings.LastIndex(path, cleanPath); idx > 0 {
				prefix = path[:idx]
			} else {
				prefix = path
			}
			// Remove leading/trailing slashes for the warning
			prefix = strings.Trim(prefix, "/")
			if prefix != "" {
				fmt.Fprintf(os.Stderr, "tar: removing leading '%s/' from member names\n", prefix)
			}
		}
		path = cleanPath

		err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Sanitize relative path
			relPath := filePath
			relPath = strings.TrimPrefix(relPath, "./")

			var linkTarget string
			var header *tar.Header

			if info.Mode()&os.ModeSymlink != 0 {
				linkTarget, err = os.Readlink(filePath)
				if err != nil {
					return err
				}
				header, err = tar.FileInfoHeader(info, linkTarget)
				if err != nil {
					return err
				}
			} else {
				header, err = tar.FileInfoHeader(info, "")
				if err != nil {
					return err
				}
			}

			header.Name = relPath

			if info.IsDir() && !strings.HasSuffix(header.Name, "/") {
				header.Name += "/"
			}

			if err := tw.WriteHeader(header); err != nil {
				return err
			}

			if info.Mode().IsRegular() {
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
			fmt.Fprintf(os.Stderr, "gobox: tar: %v\n", err)
			return 1
		}
	}
	tw.Close()

	if tarFile != "" {
		mode := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
		f, err := os.OpenFile(tarFile, mode, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: tar: %v\n", err)
			return 1
		}
		defer f.Close()
		f.Write(buf.Bytes())
	} else {
		out := io.Writer(os.Stdout)
		if compress {
			gw := gzip.NewWriter(out.(io.WriteCloser))
			gw.Write(buf.Bytes())
			gw.Close()
		} else {
			os.Stdout.Write(buf.Bytes())
		}
	}

	return 0
}

func tarExtract(tarFile string, dir string, compress bool) int {
	var r io.Reader

	if tarFile != "" {
		f, err := os.Open(tarFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: tar: %v\n", err)
			return 1
		}
		defer f.Close()
		r = f
	} else {
		r = os.Stdin
	}

	if compress {
		gr, err := gzip.NewReader(r)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: tar: %v\n", err)
			return 1
		}
		defer gr.Close()
		r = gr
	}

	tr := tar.NewReader(r)

	// Check for empty input (not a tarball)
	header, err := tr.Next()
	if err == io.EOF {
		// Check if there's any data at all
		buf := make([]byte, 1)
		var n int
		if tarFile != "" {
			f, _ := os.Open(tarFile)
			if f != nil {
				n, _ = f.Read(buf)
				f.Close()
			}
		} else {
			// Can't easily peek at stdin, assume empty
			return 0
		}
		if n == 0 {
			fmt.Fprintln(os.Stderr, "tar: short read")
			return 1
		}
		return 0
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "gobox: tar: %v\n", err)
		return 1
	}

	// Process the first header and continue with remaining
	for {
		target := header.Name
		if dir != "" {
			target = filepath.Join(dir, target)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(target, os.FileMode(header.Mode))
			os.Chmod(target, os.FileMode(header.Mode))
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(target), 0755)
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				fmt.Fprintf(os.Stderr, "gobox: tar: %v\n", err)
				return 1
			}
			io.Copy(f, tr)
			f.Close()
			os.Chmod(target, os.FileMode(header.Mode))
		case tar.TypeSymlink:
			os.MkdirAll(filepath.Dir(target), 0755)
			os.Remove(target)
			if err := os.Symlink(header.Linkname, target); err != nil {
				fmt.Fprintf(os.Stderr, "gobox: tar: %v\n", err)
				return 1
			}
		case tar.TypeLink:
			linkTarget := header.Linkname
			if dir != "" {
				linkTarget = filepath.Join(dir, linkTarget)
			}
			os.MkdirAll(filepath.Dir(target), 0755)
			if err := os.Link(linkTarget, target); err != nil {
				fmt.Fprintf(os.Stderr, "gobox: tar: %v\n", err)
				return 1
			}
		}

		header, err = tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: tar: %v\n", err)
			return 1
		}
	}
	return 0
}

func tarList(tarFile string, compress bool) int {
	var r io.Reader

	if tarFile != "" {
		f, err := os.Open(tarFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: tar: %v\n", err)
			return 1
		}
		defer f.Close()
		r = f
	} else {
		r = os.Stdin
	}

	if compress {
		gr, err := gzip.NewReader(r)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: tar: %v\n", err)
			return 1
		}
		defer gr.Close()
		r = gr
	}

	tr := tar.NewReader(r)

	// Check for empty input
	header, err := tr.Next()
	if err == io.EOF {
		buf := make([]byte, 1)
		var n int
		if tarFile != "" {
			f, _ := os.Open(tarFile)
			if f != nil {
				n, _ = f.Read(buf)
				f.Close()
			}
		}
		if n == 0 {
			fmt.Fprintln(os.Stderr, "tar: short read")
			return 1
		}
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

// bunzip2Main - decompress .bz2 files
func bunzip2Main(args []string) int {
	return execTool("bunzip2", args[1:])
}

// bzcatMain - decompress .bz2 to stdout
func bzcatMain(args []string) int {
	return execTool("bzcat", args[1:])
}

// bzip2Main - compress files with bzip2
func bzip2Main(args []string) int {
	return execTool("bzip2", args[1:])
}

// unlzmaMain - decompress .lzma files
func unlzmaMain(args []string) int {
	return execTool("unlzma", args[1:])
}

// lzcatMain - decompress .lzma to stdout
func lzcatMain(args []string) int {
	return execTool("lzcat", args[1:])
}

// lzmaMain - compress with lzma
func lzmaMain(args []string) int {
	return execTool("lzma", args[1:])
}

// unxzMain - decompress .xz files
func unxzMain(args []string) int {
	return execTool("unxz", args[1:])
}

// xzcatMain - decompress .xz to stdout
func xzcatMain(args []string) int {
	return execTool("xzcat", args[1:])
}

// xzMain - compress with xz
func xzMain(args []string) int {
	return execTool("xz", args[1:])
}

// uncompressMain - decompress .Z files
func uncompressMain(args []string) int {
	return execTool("uncompress", args[1:])
}

// cpioMain - copy files in/out of cpio archives
func cpioMain(args []string) int {
	return execTool("cpio", args[1:])
}

// arMain - create/extract ar archives
func arMain(args []string) int {
	return execTool("ar", args[1:])
}

// unzipMain - list/test/extract zip archives
func unzipMain(args []string) int {
	return execTool("unzip", args[1:])
}

// lzopMain - compress with lzop
func lzopMain(args []string) int {
	return execTool("lzop", args[1:])
}

// unlzopMain - decompress .lzo files
func unlzopMain(args []string) int {
	return execTool("unlzop", args[1:])
}

// lzopcatMain - decompress .lzo to stdout
func lzopcatMain(args []string) int {
	return execTool("lzopcat", args[1:])
}

// rpm2cpioMain - extract cpio archive from RPM
func rpm2cpioMain(args []string) int {
	return execTool("rpm2cpio", args[1:])
}

// execTool runs an external tool with args, passing stdin/stdout/stderr
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
