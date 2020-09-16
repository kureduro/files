package main

import (
    "fmt"
	"bufio"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

const copySuffix = "_copy"

type FileIndex struct {
    index map[string]int
    sync.Mutex
}

func getBareFilename(filename string) string {
    return strings.TrimSuffix(filename, filepath.Ext(filename))
}

func main() {
    dir, err := os.Open("./")
    if err != nil {
        log.Fatalf("could not open current directory, %v", err)
    }

    var index FileIndex
    index.index = make(map[string]int)

    filenames, err := dir.Readdirnames(-1)
    for _, filename := range filenames {
        latestCopy := 0

        fileBare := getBareFilename(filename)
        for _, copyName := range filenames {
            if !strings.HasPrefix(copyName, fileBare) {
                continue
            }

            copyBare := getBareFilename(copyName[len(fileBare):])
            numStart := strings.LastIndex(copyBare, copySuffix)
            if numStart == -1 {
                continue
            }
            numStart += len(copySuffix)

            copyNum, err := strconv.Atoi(copyBare[numStart:])
            if err != nil {
                continue
            }

            if latestCopy < copyNum {
                latestCopy = copyNum
            }
        }

        index.index[filename] = latestCopy
    }

    fmt.Print(index.index)


    l, err := net.Listen("tcp", ":8888")
    if err != nil {
        log.Fatalf("could not start listening, %v", err)
    }
    defer l.Close()

    log.Println("Ready")

    for {
        con, err := l.Accept()
        if err != nil {
            log.Fatalf("could not accept an incoming connection, %v", err)
        }

        go func(con net.Conn) {
            defer con.Close()

            s := bufio.NewScanner(con)
            s.Scan()
            filename := s.Text()
            log.Printf("receiving %q", filename)

            index.Lock()
            copyNum, exists := index.index[filename]

            if !exists {
                index.index[filename] = 0    
            } else {
                index.index[filename]++
                filename = fmt.Sprintf("%s%s%d%s", getBareFilename(filename), copySuffix, copyNum+1, filepath.Ext(filename))
                index.index[filename] = 0
                log.Printf("name conflict resolved, receiving %q", filename)
            }
            index.Unlock()

            file, err := os.Create(filename)
            if err != nil {
                log.Fatalf("could not create file %q, %v", filename, err)
            }

            n, err := io.Copy(file, con)
            if err != nil {
                log.Fatalf("could not receive file %q, %v", filename, err)
            }

            log.Printf("received %q (%d bytes)", filename, n)

        }(con)
    }
}
