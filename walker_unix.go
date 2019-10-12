// +build linux darwin freebsd openbsd netbsd
// +build !appengine

package walker

import (
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func (w *walker) walk(dirname string) error {
	fd, err := syscall.Open(dirname, 0, 0)
	if err != nil {
		return &os.PathError{Op: "open", Path: dirname, Err: err}
	}
	defer syscall.Close(fd)

	buf := make([]byte, 8<<10)
	n, err := unix.ReadDirent(fd, buf)
	if err != nil {
		return err
	}

	names := make([]string, 0, 100)
	offset := 0
	for {
		consumed, count, names := unix.ParseDirent(buf[offset:n], 100, names[0:])
		offset += consumed

		if count <= 0 {
			return nil
		}

		for _, name := range names[:count] {
			fi, err := os.Lstat(dirname + "/" + name)
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return err
			}
			if err = w.do(dirname, fi); err != nil {
				return err
			}
		}
	}
	return nil
}
