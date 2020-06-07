package walker_test

import (
	"context"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	// "github.com/karrick/godirwalk"
	"github.com/saracen/walker"
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
		err := os.MkdirAll(filepath.Dir(path), 0777)
		if err != nil {
			t.Fatal(err)
		}

		switch {
		case mode&os.ModeSymlink != 0 && mode&os.ModeDir != 0:
			err = os.Symlink(filepath.Dir(path), path)

		case mode&os.ModeSymlink != 0:
			err = os.Symlink("foo/foo.go", path)

		case mode&os.ModeDir != 0:
			err = os.Mkdir(path, mode)

		default:
			err = ioutil.WriteFile(path, []byte(path), mode)
		}
		if err != nil {
			t.Fatal(err)
		}
	}

	filepathResults := make(map[string]os.FileInfo)
	err = filepath.Walk(dir, func(pathname string, fi os.FileInfo, err error) error {
		if strings.Contains(pathname, "skip") {
			return filepath.SkipDir
		}

		if filepath.Base(pathname) == "perm-error" && runtime.GOOS != "windows" {
			if err == nil {
				t.Errorf("expected permission error for path %v", pathname)
			}
		} else {
			if err != nil {
				t.Errorf("unexpected error for path %v", pathname)
			}
		}

		filepathResults[pathname] = fi
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	var l sync.Mutex
	walkerResults := make(map[string]os.FileInfo)
	err = walker.Walk(dir, func(pathname string, fi os.FileInfo) error {
		if strings.Contains(pathname, "skip") {
			return filepath.SkipDir
		}

		l.Lock()
		walkerResults[pathname] = fi
		l.Unlock()

		return nil
	}, walker.WithErrorCallback(func(pathname string, err error) error {
		if filepath.Base(pathname) == "perm-error" {
			if err == nil {
				t.Errorf("expected permission error for path %v", pathname)
			}
		} else {
			if err != nil {
				t.Errorf("unexpected error for path %v", pathname)
			}
		}
		return nil
	}))

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
		"perm-error":          os.ModeDir | 0000,
	})
}

func TestWalkerWithContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Nanosecond)
	defer cancel()
	err := walker.WalkWithContext(ctx, runtime.GOROOT(), func(pathname string, fi os.FileInfo) error {
		return nil
	})
	if err == nil {
		t.Fatalf("expecting timeout error, got nil")
	}
}

var benchDir = flag.String("benchdir", runtime.GOROOT(), "The directory to scan for BenchmarkFilepathWalk and BenchmarkWalkerWalk")

type tester interface {
	Fatal(args ...interface{})
}

func filepathWalk(t tester) {
	err := filepath.Walk(*benchDir, func(pathname string, fi os.FileInfo, err error) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
}

func filepathWalkAppend(t tester) (paths []string) {
	err := filepath.Walk(*benchDir, func(pathname string, fi os.FileInfo, err error) error {
		paths = append(paths, pathname)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return
}

func TestFilepathWalkDir(t *testing.T) { filepathWalk(t) }

func BenchmarkFilepathWalk(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		filepathWalk(b)
	}
}

func BenchmarkFilepathWalkAppend(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = filepathWalkAppend(b)
	}
}

func walkerWalk(t tester) {
	err := walker.Walk(*benchDir, func(pathname string, fi os.FileInfo) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
}

func walkerWalkAppend(t tester) (paths []string) {
	var l sync.Mutex
	err := walker.Walk(*benchDir, func(pathname string, fi os.FileInfo) error {
		l.Lock()
		paths = append(paths, pathname)
		l.Unlock()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return
}

func TestWalkerWalkDir(t *testing.T) { walkerWalk(t) }

func BenchmarkWalkerWalk(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		walkerWalk(b)
	}
}

func BenchmarkWalkerWalkAppend(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = walkerWalkAppend(b)
	}
}

func fastwalkWalk(t tester) {
	err := fastwalk.Walk(*benchDir, func(pathname string, mode os.FileMode) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
}

func fastwalkWalkLstat(t tester) {
	err := fastwalk.Walk(*benchDir, func(pathname string, mode os.FileMode) error {
		_, err := os.Lstat(pathname)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func fastwalkWalkAppend(t tester) (paths []string) {
	var l sync.Mutex
	err := fastwalk.Walk(*benchDir, func(pathname string, mode os.FileMode) error {
		l.Lock()
		paths = append(paths, pathname)
		l.Unlock()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return
}

func TestFastwalkWalkDir(t *testing.T) { fastwalkWalk(t) }

func TestFastwalkWalkLstatDir(t *testing.T) { fastwalkWalkLstat(t) }

func BenchmarkFastwalkWalk(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		fastwalkWalk(b)
	}
}

func BenchmarkFastwalkWalkAppend(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = fastwalkWalkAppend(b)
	}
}

func BenchmarkFastwalkWalkLstat(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		fastwalkWalkLstat(b)
	}
}

/*func godirwalkWalk(t tester) {
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

func godirwalkWalkLstat(t tester) (paths []string) {
	err := godirwalk.Walk(*benchDir, &godirwalk.Options{
		Callback: func(osPathname string, dirent *godirwalk.Dirent) error {
			_, err := os.Lstat(osPathname)
			return err
		},
		Unsorted: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return
}

func godirwalkWalkAppend(t tester) (paths []string) {
	err := godirwalk.Walk(*benchDir, &godirwalk.Options{
		Callback: func(osPathname string, dirent *godirwalk.Dirent) error {
			paths = append(paths, osPathname)
			return nil
		},
		Unsorted: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return
}

func TestGodirwalkWalkDir(t *testing.T) { godirwalkWalk(t) }

func BenchmarkGodirwalkWalk(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		godirwalkWalk(b)
	}
}

func BenchmarkGodirwalkWalkAppend(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = godirwalkWalkAppend(b)
	}
}

func BenchmarkGodirwalkWalkLstat(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		godirwalkWalkLstat(b)
	}
}*/
