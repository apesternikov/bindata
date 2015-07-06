package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/apesternikov/bindata/internal/mkbinfs"
)

func main() {
	flag.Parse()
	if len(flag.Args()) == 0 {
		flag.Usage()
		return
	}
	for _, root := range flag.Args() {
		abspath, err := filepath.Abs(root)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Unable to obtain abs path of ", root, ":", err.Error())
			continue
		}
		root = filepath.Clean(abspath)
		fi, err := os.Stat(root)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Unable to stat file ", root, ":", err)
			continue
		}
		if fi.IsDir() {
			mkbinfs.DoDir(root)
		} else {
			mkbinfs.DoFile(root, fi)
		}

	}
}
