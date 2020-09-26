package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/net/context"

	"github.com/cheggaaa/pb/v3"
)

type Parcel struct {
    File *os.File
    Path string
    Name string
    Size int
}

// NewParcel will construct the new parcel, filling it with information
// about the payload.
func NewParcel(fullPath string) (*Parcel, error) {
    parcel := &Parcel{
        Path: fullPath,
        Name: filepath.Base(fullPath),
    }

    var err error
    parcel.File, err = os.Open(fullPath)
    if err != nil {
        return nil, fmt.Errorf("could not create parcel, %v", err)
    }

    stat, err := parcel.File.Stat()
    if err != nil {
        return nil, fmt.Errorf("could not create parcel, %v\n", err)
    }

    parcel.Size = int(stat.Size())

    return parcel, nil
}

func (p *Parcel) Read(b []byte) (int, error) {
    return p.File.Read(b)
}

func (p *Parcel) Close() {
    p.File.Close()
}

func dial(hostAddr string) (net.Conn, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 10 * time.Second)
    defer cancel()

    var d net.Dialer
    con, err := d.DialContext(ctx, "tcp", hostAddr)
    if err != nil {
        return nil, fmt.Errorf("could not dial destination host, %v", err)
    }

    return con, nil
}

func main() {
    if len(os.Args) != 3 {
        fmt.Printf("Usage:\n\tfilec <filename> <host>:<port>\n\n")
        return
    }

    parcel, err := NewParcel(os.Args[1])
    if err != nil {
        fmt.Println(err)
        return
    }
    defer parcel.Close()

    con, err := dial(os.Args[2])
    if err != nil {
        fmt.Println(err)
        return
    }
    defer con.Close()

    // Protocol (with Clinet and Server)
    // C: <filename>\n
    // C: <file size in bytes>\n
    // S: <filename on the server>
    // C: <data>

    _, err = fmt.Fprintf(con, "%s\n%d\n", parcel.Name, parcel.Size)
    if err != nil {
        fmt.Printf("could not transfer metadata, %v\n", err)
        return
    }

    buf := make([]byte, 1024)
    n, err := con.Read(buf)
    serverFilename := string(buf[:n])
    if serverFilename != parcel.Name {
        fmt.Printf("warning: %s already exists on server, will be renamed to %s\n",
                  parcel.Name, serverFilename)
    }

    bar := pb.Full.Start(parcel.Size)
    barWriter := bar.NewProxyWriter(con)

    for i, n := 0, 0; i < parcel.Size; i += n {
        n, err = parcel.Read(buf)
        if err != nil && err != io.EOF {
            fmt.Printf("unexpected error reading file at byte %d, %v\n", i, err)
            return
        }

        _, err = barWriter.Write(buf[:n])
        if err != nil {
            fmt.Printf("unexpected error transferring file at byte %d, %v\n", i, err)
            return
        }
    }
    bar.Finish()
}
