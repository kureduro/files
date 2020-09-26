# files

A simple **FILE S**erver written in Go.

### Run it!

The server will store the files it receives to the current working directory. You also need to pass a port number on which the server should listen incoming connections. To run the server

```
$ go run cmd/server/* <port>
```

The client expects a name of the file and the server's address as its arguments. Build the client first
```
$ go build cmd/client/*
$ ./client test.txt localhost:8888
```

If a file with the same name already exists in current working directory, the file will be renamed. For example `name.ext` will be renamed to `name_copy2.ext`. The server takes care of tracking down the "copies" of the files, such that you can restart the server at any time and the copy numbers will be correctly increased with no files overwritten accidentally.
