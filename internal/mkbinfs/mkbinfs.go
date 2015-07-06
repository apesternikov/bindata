package mkbinfs

import (
	"bufio"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var pkgname = flag.String("p", "main", "Optional name of the package to generate.")

type ByteWriter struct {
	io.Writer
	c int
}

func (w *ByteWriter) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return
	}
	var buf [5]byte
	for n = range p {
		if w.c%12 == 0 {
			w.Writer.Write([]byte{'\n'})
			w.c = 0
		}
		buf[0] = '0'
		buf[1] = 'x'
		hex.Encode(buf[2:], p[n:n+1])
		buf[4] = ','
		w.Writer.Write(buf[:])
		w.c++
	}
	n++
	return
}

func translate(input io.Reader, output io.Writer, pkgname, varname string, filename string, info os.FileInfo) {
	fmt.Fprintf(output, `package %s
//This file has been generated by bindata, DO NOT EDIT!
import "github.com/apesternikov/bindata"
import "time"

var %s = &bindata.Bindata{ []byte{
	`, pkgname, varname)

	io.Copy(&ByteWriter{Writer: output}, input)

	fmt.Fprintf(output, `},
	"%s", %d, time.Unix(%d, 0),
}`, filename, info.Mode(), info.ModTime().Unix())
}

func pathToPkg(in string) (pkg string) {
	pkg = strings.Replace(in, "-", "_", -1)
	pkg = strings.Replace(pkg, ".", "_", -1)
	return
}

func genfile(path string, info os.FileInfo) (varname string) {
	name := info.Name()
	fmt.Fprintf(os.Stderr, "binary file %s (%s)\n", path, name)
	pkgname := pathToPkg(filepath.Base(path))
	varname = strings.Replace(info.Name(), ".", "_", -1)
	varname = strings.Replace(varname, "-", "_", -1)
	varname = strings.Title(varname)
	fs, err := os.Open(path + "/" + name)
	if err != nil {
		panic(err)
	}
	defer fs.Close()
	ofilename := path + "/" + name + ".go"
	ofs, err1 := os.Create(ofilename)
	if err1 != nil {
		panic(err1)
	}
	defer ofs.Close()
	bofs := bufio.NewWriter(ofs)
	translate(fs, bofs, pkgname, varname, info.Name(), info)
	bofs.Flush()
	return
}

var gopath = os.Getenv("GOPATH")

func getFullPackageName(path string) string {
	for _, p := range filepath.SplitList(gopath) {
		abspath, err := filepath.Abs(p)
		if err != nil {
			panic(err)
		}
		root := filepath.Clean(abspath)
		root = root + "/src/"
		if strings.HasPrefix(path, root) {
			return path[len(root):]
		}
	}
	panic(errors.New("Path not in GOPATH"))
}

func DoDir(path string) {
	fmt.Fprintf(os.Stderr, "binary FS for %s\n", path)
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error opening ", path, " : ", err.Error())
		return
	}
	var files, subdirs, subdirs1 []string
	for {
		v, err := f.Readdir(100)
		if err != nil {
			if err != io.EOF {
				fmt.Fprintln(os.Stderr, "Readdir error", err.Error())
				return
			}
			break
		}
		for _, fi := range v {
			name := fi.Name()
			if strings.HasPrefix(name, ".") {
				continue
			}
			if strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "~") {
				continue
			}
			if fi.IsDir() {
				fmt.Fprintf(os.Stderr, "dir %s %s\n", path, name)
				subdirs = append(subdirs, name)
				subdirs1 = append(subdirs1, pathToPkg(name)+".Dir")
				continue
			}
			varname := genfile(path, fi)
			files = append(files, varname)
		}
	}
	sort.Strings(subdirs)
	sort.Strings(files)
	ofilename := path + "/fsdir.go"
	ofs, err1 := os.Create(ofilename)
	if err1 != nil {
		panic(err1)
	}
	defer ofs.Close()
	bofs := bufio.NewWriter(ofs)
	fmt.Fprintln(bofs, "package", pathToPkg(filepath.Base(path)))
	fmt.Fprintln(bofs, `import "github.com/apesternikov/bindata"`)
	if len(subdirs) > 0 {
		fullpkgname := getFullPackageName(path)
		fmt.Fprintln(bofs, "import (")
		for _, s := range subdirs {
			fmt.Fprintf(bofs, "  %s \"%s/%s\"\n", pathToPkg(s), fullpkgname, s)
		}
		fmt.Fprintln(bofs, ")")
	}

	fmt.Fprintln(bofs, "var Files = []*bindata.Bindata{", strings.Join(files, ", "), "}")
	fmt.Fprintln(bofs, "var Dirs = []*bindata.Dir{", strings.Join(subdirs1, ", "), "}")
	fmt.Fprintf(bofs, "var Dir = bindata.NewDir(\"%s\", Files, Dirs)\n", filepath.Base(path))

	bofs.Flush()
	for _, s := range subdirs {
		DoDir(path + "/" + s)
	}
}

func DoFile(path string, fi os.FileInfo) {
	idx := strings.LastIndex(path, "/")
	if idx == -1 {
		fmt.Fprintln(os.Stderr, "Unable separate path of ", path)
		return
	}
	path = path[:idx]
	genfile(path, fi)
}