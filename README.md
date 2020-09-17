# files

A simple FILE Server written in Go.

### Run it!

The server will store the files it receives to the current working directory. You also need to pass the port on which the server should listen incoming connections. To run the server

```
$ go run cmd/server/* <port>
```

The client expects the name of the file and the server address as its arguments. Build the client first
```
$ go build cmd/client/*
$ client test.txt localhost:8888
```

