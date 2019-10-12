package walker

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	//"github.com/karrick/godirwalk"
	"github.com/saracen/walker/testdata/fastwalk"
)

func testWalk(t *testing.T, files map[string]os.FileMode) {
	dir, err := ioutil.TempDir("", "walker-test")
	if err != nil {
		t.Error(err)
		return
	}
	defer os.RemoveAll(dir)

	for path, mode := range files {
		path = filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(path), 0777); err != nil {
			t.Fatal(err)
		}

		switch {
		case mode&os.ModeSymlink != 0 && mode&os.ModeDir != 0:
			if err := os.Symlink(filepath.Dir(path), path); err != nil {
				t.Fatal(err)
			}

		case mode&os.ModeSymlink != 0:
			if err := os.Symlink("foo/foo.go", path); err != nil {
				t.Fatal(err)
			}

		default:
			if err := ioutil.WriteFile(path, []byte(path), mode); err != nil {
				t.Fatal(err)
			}
		}
	}

	filepathResults := make(map[string]os.FileInfo)
	err = filepath.Walk(dir, func(pathname string, fi os.FileInfo, err error) error {
		if strings.Contains(pathname, "skip") {
			return filepath.SkipDir
		}

		filepathResults[pathname] = fi
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	var l sync.Mutex
	walkerResults := make(map[string]os.FileInfo)
	err = Walk(dir, func(pathname string, fi os.FileInfo) error {
		if strings.Contains(pathname, "skip") {
			return filepath.SkipDir
		}

		l.Lock()
		walkerResults[pathname] = fi
		l.Unlock()

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for path, info := range filepathResults {
		info2, ok := walkerResults[path]
		if !ok {
			t.Fatalf("walk mismatch, path %q doesn't exist", path)
		}

		if info.IsDir() != info2.IsDir() ||
			info.ModTime() != info2.ModTime() ||
			info.Mode() != info2.Mode() ||
			info.Name() != info2.Name() ||
			info.Size() != info2.Size() {
			t.Fatalf("walk mismatch, got %v, wanted %v", info2, info)
		}
	}
}

func TestWalker(t *testing.T) {
	testWalk(t, map[string]os.FileMode{
		"foo/foo.go":          0644,
		"bar/bar.go":          0777,
		"bar/foo/bar/foo/bar": 0600,
		"skip/file":           0700,
		"bar/symlink":         os.ModeDir | os.ModeSymlink | 0777,
		"bar/symlink.go":      os.ModeSymlink | 0777,
	})
}

var benchDir = flag.String("benchdir", runtime.GOROOT(), "The directory to scan for BenchmarkFilepathWalk and BenchmarkWalkerWalk")

func TestFilepathWalkDir(t *testing.T) {
	err := filepath.Walk(*benchDir, func(pathname string, fi os.FileInfo, err error) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
}

func BenchmarkFilepathWalk(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		err := filepath.Walk(*benchDir, func(pathname string, fi os.FileInfo, err error) error { return nil })
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestWalkerDir(t *testing.T) {
	err := Walk(*benchDir, func(pathname string, fi os.FileInfo) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
}

func BenchmarkWalkerWalk(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		err := Walk(*benchDir, func(pathname string, fi os.FileInfo) error { return nil })
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestFastwalkDir(t *testing.T) {
	err := fastwalk.Walk(*benchDir, func(pathname string, mode os.FileMode) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func BenchmarkFastwalkWalk(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		err := fastwalk.Walk(*benchDir, func(pathname string, mode os.FileMode) error {
			_, err := os.Lstat(pathname)
			return err
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

/*func TestGodirwalkDir(t *testing.T) {
	err := godirwalk.Walk(*benchDir, &godirwalk.Options{
		Callback: func(osPathname string, dirent *godirwalk.Dirent) error {
			return nil
		},
		Unsorted: true,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func BenchmarkGodirwalkWalk(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		err := godirwalk.Walk(*benchDir, &godirwalk.Options{
			Callback: func(osPathname string, dirent *godirwalk.Dirent) error {
				return nil
			},
			Unsorted: true,
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}*/
