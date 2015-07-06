package bindata

import (
	"bytes"
	"errors"
	"net/http"
	"os"
	"time"
)

type Bindata struct {
	Data        []byte
	Filename    string
	FileMode    os.FileMode
	Time        time.Time
	FullPkgPath string //relative to the $GOROOT/$GOPATH
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

var BindataDevMode = flags.String()

var starttime = time.Now()

func (d *Bindata) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	reader := bytes.NewReader(d.Data)
	http.ServeContent(w, req, d.Filename, starttime, reader)
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
	return &Dir{pkg, files, dirs}
}

type HttpFs struct {
	files map[string]*Bindata
}

func NewHttpFs(root *Dir) *HttpFs {
	fs := &HttpFs{files: make(map[string]*Bindata)}
	fs.appendDir("/", root)
	return fs
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
