## bindata

bindata is a [Go](http://golang.org) package that allows embedding html, images or any other files into your go binary. This would simplify your server deployment to a single binary file and eliminate many deployment - related errors like missing resource files.

### Installation
```
go get -u github.com/apesternikov/bindata
go install github.com/apesternikov/bindata/mkbinfs
```

### Usage

Place your files in your go source tree, for example:
```
$GOPATH/src/yourapplication/static/docroot
$GOPATH/src/yourapplication/static/tpls
```

Run the mkbinfs utility on your files:
```
mkbinfs $GOPATH/src/yourapplication/static
```
It will generate several .go files in the tree rooted under static directory.
You can start using it as http filesystem:
```go
import "yourapplication/static/docroot"

http.Handle("/", http.FileServer(bindata.NewHttpFs(docroot.Dir)))
```
Or use it for your html templates:

```go
import "yourapplication/static/tpls"

http.Handle("/yoururl", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
	data := Data{...}
	err := tpls.Stats_html.AsHtmlTemplate().Execute(w, &data)
	if err != nil {
		glog.Error("unable to execute template: ", err)
		return
	}
	return
}))
```
If tempate compilation fails AsHtmlTemplate() would log error and return the previous compilable version. If there were no compilable template since the binary start AsHtmlTemplate() would return nil.

### Developer mode

Developer mode could be used to update the content of your embedded data without recompilation or restarting. Just use ```-bindata_dev_mode``` command line parameter for your binary and the bindata package will try to locate files providing $GOPATH is set.
