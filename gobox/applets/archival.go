package applets

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func init() {
	Register("tar", AppletFunc(tarMain))
	Register("gzip", AppletFunc(gzipMain))
	Register("gunzip", AppletFunc(gunzipMain))
	Register("zcat", AppletFunc(zcatMain))
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
		err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			header, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}
			header.Name = filePath

			if info.IsDir() {
				header.Name += "/"
			}

			if err := tw.WriteHeader(header); err != nil {
				return err
			}

			if !info.IsDir() {
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

	out := io.Writer(os.Stdout)
	if compress {
		gw := gzip.NewWriter(out.(io.WriteCloser))
		gw.Write(buf.Bytes())
		gw.Close()
	} else {
		os.Stdout.Write(buf.Bytes())
	}

	if tarFile != "" {
		mode := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
		f, err := os.OpenFile(tarFile, mode, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: tar: %v\n", err)
			return 1
		}
		defer f.Close()
		f.Write(buf.Bytes())
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
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobox: tar: %v\n", err)
			return 1
		}

		target := header.Name
		if dir != "" {
			target = filepath.Join(dir, target)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(target, os.FileMode(header.Mode))
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(target), 0755)
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				fmt.Fprintf(os.Stderr, "gobox: tar: %v\n", err)
				return 1
			}
			io.Copy(f, tr)
			f.Close()
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
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 1
		}
		fmt.Println(header.Name)
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
