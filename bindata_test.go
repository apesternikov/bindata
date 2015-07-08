package bindata_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/apesternikov/bindata"
	"github.com/apesternikov/bindata/teststatic"
)

func TestBinFile(t *testing.T) {
	if string(teststatic.File_txt.Data) != "file" {
		t.Errorf("unexpected file content '%s'", string(teststatic.File_txt.Data))
	}
}

func TestBinFile2(t *testing.T) {
	if string(teststatic.File2_txt.Data) != "file2" {
		t.Errorf("unexpected file content '%s'", string(teststatic.File2_txt.Data))
	}
}

type webtestcase struct {
	url             string
	code            int               //expected HTTP code
	content_matcher func(string) bool //content matcher should return true if expected
}

func equals(pattern string) func(string) bool {
	return func(s string) bool {
		return pattern == s
	}
}

func any() func(string) bool {
	return func(s string) bool {
		return true
	}
}

func startwith(pattern string) func(string) bool {
	return func(s string) bool {
		return strings.HasPrefix(s, pattern)
	}
}

func webtest(t *testing.T, corpus []webtestcase) {
	mux := http.NewServeMux()
	ts := httptest.NewServer(mux)
	defer ts.Close()
	fs := bindata.NewHttpFs(teststatic.Dir)
	t.Logf("fs: %v", fs)
	mux.Handle("/stat/", http.StripPrefix("/stat/", http.FileServer(fs)))
	t.Logf("serving at %s", ts.URL)
	for _, tc := range corpus {
		resp, err := http.Get(ts.URL + tc.url)
		if err != nil {
			t.Errorf("%s: Unexpected error %s", tc.url, err)
		}
		if resp.StatusCode != tc.code {
			t.Errorf("%s: Unexpected status code %d", tc.url, resp.StatusCode)
		}
		body, err := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()
		if err != nil {
			t.Errorf("%s: Unexpected error reading body", tc.url, err)
		}
		if !tc.content_matcher(string(body)) {
			t.Errorf("%s: Unexpected content '%s'", tc.url, string(body))
		}
	}

}

func TestHttpProd(t *testing.T) {
	webtest(t, []webtestcase{
		{"/stat/file.txt", 200, equals("file")},
		{"/stat/nosuchfile.txt", 404, any()},
		{"/stat/dir/file2.txt", 200, equals("def")},
		{"/stat/file1.txt", 404, any()}, // this one is not in Dir
	})
}

func TestHttpDev(t *testing.T) {
	oldflag := bindata.BindataDevMode
	defer func() { bindata.BindataDevMode = oldflag }()
	newflag := true
	bindata.BindataDevMode = &newflag

	webtest(t, []webtestcase{
		{"/stat/file.txt", 200, equals("file")},
		{"/stat/nosuchfile.txt", 404, any()},
		{"/stat/dir/file2.txt", 200, equals("def")},
		{"/stat/file.txt.go", 200, startwith("package teststatic")}, // generated go sources are exposed in dev mode
	})
}

func TestBinFile1(t *testing.T) {
	changed, err := teststatic.File1_txt.Refresh()
	if err != nil {
		t.Error("Unexpected err ", err)
	}
	if changed {
		t.Error("expected no data change")
	}
	if string(teststatic.File1_txt.Data) != "compiled" {
		t.Errorf("unexpected file content '%s'", string(teststatic.File1_txt.Data))
	}
}

func TestDevModeFile1(t *testing.T) {
	oldflag := bindata.BindataDevMode
	defer func() { bindata.BindataDevMode = oldflag }()
	newflag := true
	bindata.BindataDevMode = &newflag
	changed, err := teststatic.File1_txt.Refresh()
	if err != nil {
		t.Error("Unexpected err ", err)
	}
	if !changed {
		t.Error("expected data changed")
	}
	if string(teststatic.File1_txt.Data) != "file" {
		t.Errorf("unexpected file content '%s'", string(teststatic.File1_txt.Data))
	}
}
