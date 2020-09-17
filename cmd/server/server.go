package main

import (
    "fmt"
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

func (fi *FileIndex) resolve(filename string) (uniqueName string) {
    fi.Lock()
    defer fi.Unlock()

    uniqueName = filename

    copyNum, exists := fi.index[filename]

    if exists {
        bare := getBareFilename(filename)
        ext := filepath.Ext(filename)
        uniqueName = fmt.Sprintf("%s%s%d%s", bare, copySuffix, copyNum+1, ext)
        fi.index[filename]++
    }

    fi.index[uniqueName] = 0
    return
}

func main() {
    dir, err := os.Open("./")
    if err != nil {
        log.Fatalf("could not open current directory, %v", err)
    }

    index := &FileIndex{}
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

            var filename string
            _, err := fmt.Fscanf(con, "%s\n", &filename)
            if err != nil {
                log.Print("could not read the name of the file. connection terminated.")
                return
            }
            filename = index.resolve(filename)

            var fileSize int64
            _, err = fmt.Fscanf(con, "%d\n", &fileSize)
            if err != nil {
                log.Print("could not parse the size of the file. connection terminated.")
                return
            }

            log.Printf("receiving %q (expected %d bytes)", filename, fileSize)

            file, err := os.Create(filename)
            if err != nil {
                log.Printf("could not create file %q, %v", filename, err)
                return
            }
            defer file.Close()

            n, err := io.CopyN(file, con, fileSize)
            if err != nil {
                log.Printf("could not receive file %q, %v", filename, err)
            }

            log.Printf("received %q (%d bytes)", filename, n)

            _, err = fmt.Fprint(con, filename)
            if err != nil {
                log.Printf("could not send the name of the file back.")
            }
        }(con)
    }
}
