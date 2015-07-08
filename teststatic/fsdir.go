package teststatic
import "github.com/apesternikov/bindata"
import (
  dir "github.com/apesternikov/bindata/teststatic/dir"
)
var Files = []*bindata.Bindata{ File2_txt, File_txt }
var Dirs = []*bindata.Dir{ dir.Dir }
var Dir = &bindata.Dir{Pkg: "teststatic", Files: Files, Dirs: Dirs, FullPkgName: "github.com/apesternikov/bindata/teststatic"}
