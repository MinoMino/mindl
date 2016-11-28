[![minterm](https://godoc.org/github.com/MinoMino/minterm?status.svg)](https://godoc.org/github.com/MinoMino/minterm)

# minterm

Just a small library to help manipulate the terminal in Go.
I needed something like Python's `shutil.get_terminal_size()`,
so I wrote this, then added some more stuff I ended up needing.

# Usage

###Import
```go
import "github.com/MinoMino/minterm"
```

###Terminal Size
```go
columns, rows, err := minterm.TerminalSize()
```

###Line Reservation
```go
lr, _ := NewLineReserver()
defer lr.Release()
lr.Set("This line will always be the last line.")
fmt.Println("You can now print normally.")
```
An example of it used in conjunction with [minprogress](https://github.com/MinoMino/minprogress):
![Image of line reservation in use with minprogress.](http://minomino.org/screenshots/mkuJ_2016-11-23_06-59-57.gif)

# License

MIT. See `LICENSE` for details.
