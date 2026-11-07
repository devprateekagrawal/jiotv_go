package handlers

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKodiLogAddonZip(t *testing.T) {
	z, err := kodiLogAddonZip("http://192.168.1.1:5001")
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(z), int64(len(z)))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, f := range zr.File {
		r, _ := f.Open()
		var b bytes.Buffer
		b.ReadFrom(r)
		got[f.Name] = b.String()
	}
	py := got[kodiLogAddonID+"/default.py"]
	if !strings.Contains(py, `BASE = "http://192.168.1.1:5001"`) || strings.Contains(py, "%!") {
		t.Fatalf("default.py:\n%s", py)
	}
	if !strings.Contains(py, `"%s: %s" % (name`) {
		t.Fatalf("python format strings not unescaped:\n%s", py)
	}
	if !strings.Contains(got[kodiLogAddonID+"/addon.xml"], `id="`+kodiLogAddonID+`"`) {
		t.Fatal("addon.xml missing")
	}
}

func TestSaveKodiLog(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < kodiLogKeep+2; i++ {
		os.WriteFile(filepath.Join(dir, "2000010"+string(rune('0'+i))+"-old.log"), nil, 0o644)
	}
	path, n, err := saveKodiLog(dir, "../kodi.log", strings.NewReader("hello"))
	if err != nil || n != 5 {
		t.Fatal(n, err)
	}
	if filepath.Dir(path) != dir || !strings.HasSuffix(path, "-.._kodi.log") {
		t.Fatalf("path %s", path)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != kodiLogKeep {
		t.Fatalf("%d files kept", len(entries))
	}
}
