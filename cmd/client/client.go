package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/net/context"
)

func main() {
    if len(os.Args) != 3 {
        fmt.Printf("Usage:\n\tfilec <filename> <host>:<port>\n\n")
        return
    }

    filename := os.Args[1]
    file, err := os.Open(filename)
    if err != nil {
        fmt.Printf("could not open file %s\n%v\n", filename, err)
        return
    }
    defer file.Close()

    ctx, cancel := context.WithTimeout(context.Background(), 10 * time.Second)
    defer cancel()

    var d net.Dialer
    hostAddr := os.Args[2]
    con, err := d.DialContext(ctx, "tcp", hostAddr)
    if err != nil {
        fmt.Printf("could not connect to %s\n%v\n", hostAddr, err)
        return
    }
    defer con.Close()

    stat, err := file.Stat()
    if err != nil {
        fmt.Printf("could not access file properties, %v\n", err)
        return
    }

    fileSize := int(stat.Size())
    _, err = fmt.Fprintf(con, "%s\n%d\n", filepath.Base(file.Name()), fileSize)
    if err != nil {
        fmt.Printf("could not transfer metadata\n%v\n", err)
        return
    }

    buf := make([]byte, 1024)
    for i, n := 0, 0; i < fileSize; i += n {
        n, err = file.Read(buf)
        if n == 0 || err != nil {
            fmt.Printf("unexpected error reading file at byte %d, %v\n", i, err)
            return
        }

        _, err = con.Write(buf[:n])
        if err != nil {
            fmt.Printf("unexpected error transferring file at byte %d, %v\n", i, err)
            return
        }
        
        perc := 100.0 * float64(i) / float64(fileSize)
        fmt.Printf("   %.2f%% transfered (%db/%db)\r", perc, i, fileSize)
    }
    fmt.Printf("   100.00%% transfered (%db/%db)\n", fileSize, fileSize)

    n, err := con.Read(buf)
    serverFilename := string(buf[:n])

    fmt.Printf("saved as %q\n\n", serverFilename)
}
