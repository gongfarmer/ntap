package main

import (
	"fmt"
	"os"
  "log"
  "bytes"
  "encoding/binary"
  //  "github.com/gongfarmer/ade"
)

func main() {
  var files = os.Args[1:]
	for _, f := range files {
    var (
      err error
      fi os.FileInfo
      fh *os.File
    )
    fi, err = os.Stat(f)
    if err != nil {
      log.Fatal(err)
    }
    fmt.Printf("Size of file %s is %d.\n", fi.Name(), fi.Size())

    data := make([]byte, fi.Size())
    fh, err = os.Open(f)
    count, err := fh.Read(data)
    if err != nil {
      fmt.Printf("Got fatal error ")
      log.Fatal(err)
    }
    fmt.Printf("read %d bytes.\n", count)

    sz := data[0:4]
    fmt.Printf("first 4 bytes: % X\n", sz)

    var size uint32
    buf := bytes.NewReader(data)
    err = binary.Read(buf, binary.BigEndian, &size)
    if err != nil {
      fmt.Println("bytes.Read failed: ", err)
    }
    fmt.Printf("bytes converted to int: %d\n", size)
  }
}

