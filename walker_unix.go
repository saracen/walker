// +build linux darwin freebsd openbsd netbsd
// +build !appengine

package walker

import (
	"os"
	"syscall"
)

func (w *walker) readdir(dirname string) error {
	fd, err := open(dirname, 0, 0)
	if err != nil {
		return &os.PathError{Op: "open", Path: dirname, Err: err}
	}
	defer syscall.Close(fd)

	buf := make([]byte, 8<<10)
	names := make([]string, 0, 100)

	nbuf := 0
	bufp := 0
	for {
		if bufp >= nbuf {
			bufp = 0
			nbuf, err = readDirent(fd, buf)
			if err != nil {
				return err
			}
			if nbuf <= 0 {
				return nil
			}
		}

		consumed, count, names := syscall.ParseDirent(buf[bufp:nbuf], 100, names[0:])
		bufp += consumed

		for _, name := range names[:count] {
			fi, err := os.Lstat(dirname + "/" + name)
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return err
			}
			if err = w.walk(dirname, fi); err != nil {
				return err
			}
		}
	}
	// never reach
}

// According to https://golang.org/doc/go1.14#runtime
// A consequence of the implementation of preemption is that on Unix systems, including Linux and macOS
// systems, programs built with Go 1.14 will receive more signals than programs built with earlier releases.
//
// This causes syscall.Open and syscall.ReadDirent sometimes fail with EINTR errors.
// We need to retry in this case.
func open(path string, mode int, perm uint32) (fd int, err error) {
	for {
		fd, err := syscall.Open(path, mode, perm)
		if err != syscall.EINTR {
			return fd, err
		}
	}
}

func readDirent(fd int, buf []byte) (n int, err error) {
	for {
		nbuf, err := syscall.ReadDirent(fd, buf)
		if err != syscall.EINTR {
			return nbuf, err
		}
	}
}
