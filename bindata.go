package bindata

import (
	"bytes"
	"errors"
	"flag"
	"go/build"
	ht "html/template"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/golang/glog"
)

var BindataDevMode = flag.Bool("bindata_dev_mode", false, "developer mode for bindata")

type Bindata struct {
	Data         []byte //Do not use directly to read content, use GetBytes() instead
	Filename     string
	FileMode     os.FileMode
	Time         time.Time
	FullPkgPath  string //relative to the $GOROOT/$GOPATH
	htmltemplate *ht.Template
}

func (b *Bindata) Name() string {
	return b.Filename
}
func (b *Bindata) Size() int64 {
	return int64(len(b.Data))
}

func (b *Bindata) Mode() os.FileMode {
	return b.FileMode
}
func (b *Bindata) ModTime() time.Time {
	return b.Time
}
func (b *Bindata) IsDir() bool {
	return false
}
func (b *Bindata) Sys() interface{} {
	return nil
}

func (b *Bindata) GetBytes() []byte {
	_, err := b.Refresh()
	if err != nil {
		panic(err) //only possible in dev mode
	}
	return b.Data
}

func (b *Bindata) AsHtmlTemplate() *ht.Template {
	changed, err := b.Refresh()
	if err != nil {
		panic(err) //only possible in dev mode
	}
	//TODO: race condition
	if b.htmltemplate == nil || changed {
		t, err := ht.New(b.Filename).Parse(string(b.Data))
		if err != nil {
			glog.Errorf("Error parsing template %s: %s", b.Filename, err)
			return b.htmltemplate //return old value
		}
		b.htmltemplate = t
	}
	return b.htmltemplate
}

var NoSuchFile = errors.New("No such file")

// Refresh the content of the bindata from a file in DevMode
// Does nothing in non-DevMode(prod mode)
func (b *Bindata) Refresh() (changed bool, err error) {
	if *BindataDevMode {
		for _, root := range build.Default.SrcDirs() {
			abspath := root + "/" + b.FullPkgPath
			if finf, err := os.Stat(abspath); err == nil {
				if finf.ModTime().After(b.Time) {
					data, err := ioutil.ReadFile(abspath)
					if err != nil {
						return false, err
					}
					//TODO: race condition
					b.Data = data
					b.Time = finf.ModTime()
					b.FileMode = finf.Mode()
					return true, nil
				}
				return false, nil
			}
		}
		return false, NoSuchFile
	}
	return false, nil
}

var starttime = time.Now()

func findAbsPath(pkgpath string) string {
	var abspath string
	for _, root := range build.Default.SrcDirs() {
		abspath = root + "/" + pkgpath
		if _, err := os.Stat(abspath); err == nil {
			return abspath
		}
	}
	//not found, return path within the last source dir and let http serve 404
	return abspath
}

func (d *Bindata) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if !*BindataDevMode {
		reader := bytes.NewReader(d.Data)
		http.ServeContent(w, req, d.Filename, starttime, reader)
	} else {
		http.ServeFile(w, req, findAbsPath(d.FullPkgPath))
	}
}

var badOffsetError = errors.New("Bad offset")

//Make sure Bindata implements io.ReaderAt
func (d *Bindata) ReadAt(p []byte, off int64) (n int, err error) {
	if int64(len(d.Data)) < off {
		return 0, badOffsetError
	}
	return copy(p, d.Data[off:]), nil
}

type Dir struct {
	Pkg         string
	Files       []*Bindata
	Dirs        []*Dir
	FullPkgName string
}

func NewDir(pkg string, files []*Bindata, dirs []*Dir) *Dir {
	return &Dir{Pkg: pkg, Files: files, Dirs: dirs}
}

type HttpFs struct {
	files map[string]*Bindata
}

// create a HTTP FileSystem implementation based on the bindata content.
// Use as
// mux.Handle("/static", http.FileServer(bindata.NewHttpFs(static.Dir)))
func NewHttpFs(root *Dir) http.FileSystem {
	if !*BindataDevMode {
		fs := &HttpFs{files: make(map[string]*Bindata)}
		fs.appendDir("/", root)
		return fs
	} else {
		// dev mode
		for _, d := range build.Default.SrcDirs() {
			abspath := d + "/" + root.FullPkgName
			if finf, err := os.Stat(abspath); err == nil && finf.IsDir() {
				return http.Dir(abspath)
			}
		}
		panic("Unable to locate dir for package " + root.FullPkgName)
	}
}

func (h *HttpFs) appendDir(base string, dir *Dir) {
	for _, f := range dir.Files {
		h.files[base+f.Name()] = f
	}
	for _, d := range dir.Dirs {
		h.appendDir(base+d.Pkg+"/", d)
	}
}

func (h *HttpFs) Open(name string) (http.File, error) {
	f, ok := h.files[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	return &httpFile{f, 0}, nil
}

type httpFile struct {
	f      *Bindata
	offset int64
}

func (h *httpFile) Close() error {
	return nil
}
func (h *httpFile) Stat() (os.FileInfo, error) {
	return h.f, nil
}
func (h *httpFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, os.ErrNotExist
}

func (h *httpFile) Read(bv []byte) (int, error) {
	if h.offset >= int64(len(h.f.Data)) {
		return 0, os.ErrInvalid
	}
	l := copy(bv, h.f.Data[h.offset:])
	h.offset += int64(l)
	return l, nil
}

func (h *httpFile) Seek(offset int64, whence int) (int64, error) {
	var newOffset int64
	switch whence {
	case 0:
		newOffset = offset
	case 1:
		newOffset = h.offset + offset
	case 3:
		newOffset = int64(len(h.f.Data)) + offset
	}
	// if newOffset > len(h.f.Data) {
	// 	return 0, os.ErrInvalid
	// }
	h.offset = newOffset
	return h.offset, nil

}
