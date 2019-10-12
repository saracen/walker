# walker

[![](https://godoc.org/github.com/saracen/walker?status.svg)](http://godoc.org/github.com/saracen/walker)

`walker` is a faster, parallel version, of `filepath.Walk`.

```
walker.Walk("/tmp", func(pathname string, fi os.FileInfo) error {
    fmt.Printf("%s: %d bytes\n", pathname, fi.Size())
})
```

## Benchmarks

- Standard library (`filepath.Walk`) is `FilepathWalk`.
- This library is `WalkerWalk`
- `FastwalkWalk` is [https://github.com/golang/tools/tree/master/internal/fastwalk](fastwalk).
- `GodirwalkWalk` is [https://github.com/karrick/godirwalk](godirwalk).

`Fastwalk` and `Godirwalk` reduce the syscall count by leaving `os.Lstat` up to the user, should they require a full `os.FileInfo`. This library instead performs the `os.Lstat` call, for better compatibility with `filepath.Walk`, and attempts to reduce the time taken through other means.

These benchmarks were performed with a warm cache.

```
goos: linux
goarch: amd64
pkg: github.com/saracen/walker
BenchmarkFilepathWalk-24               1        1437479938 ns/op        330704912 B/op    758715 allocs/op
BenchmarkWalkerWalk-24                20         100948844 ns/op        71853010 B/op     593451 allocs/op
BenchmarkFastwalkWalk-24               5         233001916 ns/op        72442246 B/op     581916 allocs/op
BenchmarkGodirwalkWalk-24              2         705022087 ns/op        141308672 B/op    707996 allocs/op
```

```
goos: windows
goarch: amd64
pkg: github.com/saracen/walker
BenchmarkFilepathWalk-16               1        3100710700 ns/op        269683440 B/op   1467916 allocs/op
BenchmarkWalkerWalk-16                 4         285985675 ns/op        137157000 B/op    877448 allocs/op
BenchmarkFastwalkWalk-16               2         988358100 ns/op        268348560 B/op   1474482 allocs/op
BenchmarkGodirwalkWalk-16              1        1200790300 ns/op        111854272 B/op   1310532 allocs/op
```

Performing benchmarks without having the OS cache the directory information isn't straight forward, but to get a sense of the performance, we can flush the cache and roughly time how long it took to walk a directory:

#### filepath.Walk
```
$ sudo su -c 'sync; echo 3 > /proc/sys/vm/drop_caches'; go test -v -run TestFilepathWalkDir -benchdir $GOPATH
ok      github.com/saracen/walker       5.790s
```

#### walker
```
$ sudo su -c 'sync; echo 3 > /proc/sys/vm/drop_caches'; go test -v -run TestWalkerDir -benchdir $GOPATH
ok      github.com/saracen/walker       0.593s
```

#### fastwalk
```
$ sudo su -c 'sync; echo 3 > /proc/sys/vm/drop_caches'; go test -v -run TestFastwalkDir -benchdir $GOPATH
ok      github.com/saracen/walker       0.551s
```

#### godirwalk
```
$ sudo su -c 'sync; echo 3 > /proc/sys/vm/drop_caches'; go test -v -run TestGodirwalkDir -benchdir $GOPATH
ok      github.com/saracen/walker       3.879s
```

In this case, `fastwalk` is faster. This is due to it not having to perform an additional `lstat`. The time is almost identical to `walker` if you perform the `lstat` call yourself.
