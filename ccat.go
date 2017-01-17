package main

import (
	"fmt"
	"os"
  "log"
  //  "github.com/gongfarmer/ade"
)

func main() {
  var files = os.Args[1:]
	for _, f := range files {
    var (
      io *os.File
      err error
      fi os.FileInfo
    )
    fi, err = os.Stat(f)
    if err != nil {
      log.Fatal(err)
    }
    fi, err = os.Stat(f)
    if err != nil {
      log.Fatal(err)
    }
    fmt.Printf("Size of file %s is %d.\n", fi.Name(), fi.Size())

    data := make([]byte, fi.Size())
    count, err := io.Read(data)
    if err != nil {
      log.Fatal(err)
    }
    fmt.Printf("read %d bytes.\n", count)

    sz := data[0:4]
    fmt.Printf("read %d bytes: %q.\n", count, data[:count])
  }
}

