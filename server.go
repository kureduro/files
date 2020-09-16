package main

import (
	"bufio"
	"log"
	"net"
)

func main() {
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
            s := bufio.NewScanner(con)
            s.Scan()
            log.Printf("%q (%d bytes)", s.Text(), len(s.Text()))
            con.Close()
        }(con)
    }
}
