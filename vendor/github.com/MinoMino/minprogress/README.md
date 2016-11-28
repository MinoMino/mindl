[![minprogress](https://godoc.org/github.com/MinoMino/minprogress?status.svg)](https://godoc.org/github.com/MinoMino/minprogress)

# minprogress

A progress bar for Go that can also track the speed of progress.

Example of it used in conjuction with [minterm](https://github.com/MinoMino/minterm).
![GIF of progress bar using minterm.](http://minomino.org/screenshots/bYBB_2016-11-23_05-28-25.gif)

# Usage

###Import
```go
import "github.com/MinoMino/minprogress"
```

###Simple Use
Make 5 units of progress every second while printing the bar every second:
```go
const total = 50
pb := minprogress.NewProgressBar(total)
tick := time.Tick(time.Millisecond * 200)
go func() {
  for {
    <-tick
    pb.Progress(1)
  }
}()

for {
  if pb.Progress(0) == total {
    fmt.Println("Done!")
    break
  } else {
    fmt.Println(pb.String())
    time.Sleep(time.Second)
  }
}
```

Output:
```
[...]
   60% █████████████████░░░░░░░░░░░░ (30/50)
   70% ████████████████████░░░░░░░░░ (35/50)
   80% ███████████████████████░░░░░░ (40/50)
   90% ██████████████████████████░░░ (45/50)
Done!
```

###Advanced Use
Simulate concurrent file downloads and track download speed:
```go
const total = 50
const workerCount = 10
pb := minprogress.NewProgressBar(total)
pb.SpeedUnits = minprogress.DataUnits
// For nicer output, tell it the singular and plural name of the units.
pb.Unit = "file"
pb.Units = "files"

go func() {
  // Keep only workerCount number of goroutines working concurrently.
  workerQueue := make(chan struct{}, workerCount)
  for i := 0; i < total; i++ {
    workerQueue <- struct{}{}
    go func(uid int) {
      // 200KiB - 1 MiB
      size := rand.Intn(800000) + 200000
      downloaded := 0
      for downloaded < size {
        // Download 1 - 10 KiB every 100 - 200 ms.
        time.Sleep(time.Millisecond*time.Duration(rand.Intn(100)) + 100)
        got := rand.Intn(9000) + 1000
        // Use worker number as UID.
        pb.Report(uid, got)
        downloaded += got
      }
      // Report we're done with this particular download.
      pb.Done(uid)
      // Since one download doesn't necessarily mean one unit of progress,
      // we must also tell the progress bar we made some progress.
      pb.Progress(1)
      // Tell queue we're done.
      <-workerQueue
    }(i)
  }
}()

for {
  if pb.Progress(0) == total {
    fmt.Println("Done!")
    break
  } else {
    fmt.Println(pb.String())
    time.Sleep(time.Second)
  }
}
```

Output:
```
[...]
   82% ████████████████████████░░░░░ (41/50) files [3.4 MiB/s]
   88% ██████████████████████████░░░ (44/50) files [1.9 MiB/s]
   92% ███████████████████████████░░ (46/50) files [1.3 MiB/s]
   94% ███████████████████████████░░ (47/50) files [730.5 KiB/s]
   94% ███████████████████████████░░ (47/50) files [704.7 KiB/s]
   96% ████████████████████████████░ (48/50) files [764.3 KiB/s]
   98% ████████████████████████████░ (49/50) files [356.9 KiB/s]
   98% ████████████████████████████░ (49/50) files [259.5 KiB/s]
Done!
```

# License

MIT. See `LICENSE` for details.
