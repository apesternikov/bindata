package testdata
import "github.com/apesternikov/bindata"
var Files = []*bindata.Bindata{ File_txt }
var Dirs = []*bindata.Dir{  }
var Dir = bindata.NewDir("testdata", Files, Dirs)
