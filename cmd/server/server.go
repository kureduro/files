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

func getBareFilename(filename string) string {
    return strings.TrimSuffix(filename, filepath.Ext(filename))
}

type FileIndex struct {
    index map[string]int
    sync.Mutex
}

// NewFileIndexFromSlice will generate a file index give a slice of filenames.
// It will process the filenames and determine tha maximal copy number for
// each filename.
func NewFileIndexFromSlice(filenames []string) (*FileIndex, error) {
    fi := &FileIndex{}
    fi.index = make(map[string]int)

    for _, filename := range filenames {
        latestCopy := 0

        fileBare := getBareFilename(filename)
        for _, copyName := range filenames {
            if !strings.HasPrefix(copyName, fileBare) {
                continue
            }
            copyName := copyName[len(fileBare):]

            copyBare := getBareFilename(copyName)
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

        fi.index[filename] = latestCopy
    }

    return fi, nil
}

// NewFileIndexFromDir will generate a FileIndex given a specified directory.
func NewFileIndexFromDir(dir *os.File) (*FileIndex, error) {
    filenames, err := dir.Readdirnames(-1)
    if err != nil {
        return nil, fmt.Errorf("could not generate index, %v", err)
    }

    return NewFileIndexFromSlice(filenames)
}

// Resolve will return the passed in filename if there's no file in the root
// with the same name. Otherwise, a new filename is generated in the form
// "<original filename><copy suffix><copy number><file extension>".
// Additionally, the index itself is updated to reflect the expected changes
// in the filesystem. Was the filesystem really changed or not, doesn't matter, 
// it is assumed that the name of the presumed copy is occupied.
func (fi *FileIndex) Resolve(filename string) (uniqueName string) {
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

// receiveFile is the handler for the incomming connections.
// It expects the preferred name of the file and the file size in bytes to be
// specified in the first two lines of the input respectively. After the
// expected number of bytes is received, the actual name of the file, where
// the data is saved, is written to the socket (without \n) and the connection 
// is closed.
func receiveFile(con net.Conn, index *FileIndex) {
    defer con.Close()

    var filename string
    _, err := fmt.Fscanf(con, "%s\n", &filename)
    if err != nil {
        log.Print("could not read the name of the file. connection terminated.")
        return
    }
    filename = index.Resolve(filename)

    var fileSize int64
    _, err = fmt.Fscanf(con, "%d\n", &fileSize)
    if err != nil {
        log.Print("could not parse the size of the file. connection terminated.")
        return
    }

    file, err := os.Create(filename)
    if err != nil {
        log.Printf("could not create file %q, %v", filename, err)
        return
    }
    defer file.Close()

    log.Printf("receiving %q (expected %d bytes)", filename, fileSize)

    n, err := io.CopyN(file, con, fileSize)
    if err != nil {
        log.Printf("could not receive file %q, %v", filename, err)
    }

    log.Printf("received %q (%d bytes)", filename, n)

    _, err = fmt.Fprint(con, filename)
    if err != nil {
        log.Printf("could not send the name of the file back.")
    }
}

func main() {
    dir, err := os.Open("./")
    if err != nil {
        log.Fatalf("could not open current directory, %v", err)
    }
    defer dir.Close()

    index, err := NewFileIndexFromDir(dir)
    if err != nil {
        log.Fatal(err)
    }

    l, err := net.Listen("tcp", ":8888")
    if err != nil {
        log.Fatalf("could not start listening, %v", err)
    }
    defer l.Close()

    for {
        con, err := l.Accept()
        if err != nil {
            log.Fatalf("could not accept an incoming connection, %v", err)
        }

        go receiveFile(con, index)
    }
}
