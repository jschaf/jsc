+++
slug = "go-server-with-syscalls"
date = 2019-03-12
visibility = "published"
+++

# Create a Go web server from scratch with Linux system calls

> A web-server with Linux syscalls.

One itch I’ve wanted to scratch for a while is to create a web-server from
scratch without relying on libraries and without first
[inventing the universe](https://www.goodreads.com/quotes/32952-if-you-wish-to-make-an-apple-pie-from-scratch).
I’ve also wanted a chance to take Go for a spin. I’ll cover how to create a web
server in Go using Linux system calls.

**Completed Code at GitHub**:
[scratch_server.go](https://gist.github.com/jschaf/93f37aedb5327c54cb356b2f1f0427e3)

## Non-goals

The Go `net` package is a full-featured, production ready library. We’ll skip
the following features:

- HTTP
  [100 Continue](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/100)
  support
- TLS
- Most error checking
- Persistent and chunked connections
- HTTP Redirects
- Deadline and cancellation
- Non-blocking sockets

CONTINUE_READING

## Overview

The steps follow the same structure as this in-depth
[Medium article](https://medium.com/from-the-scratch/http-server-what-do-you-need-to-know-to-build-a-simple-http-server-from-scratch-d1ef8945e4fa):

- Create the socket -
  [socket](http://man7.org/linux/man-pages/man2/socket.2.html)
- Identify the socket by binding it to a socket address -
  [bind](http://man7.org/linux/man-pages/man2/bind.2.html)
- Allow connections to the socket -
  [listen](http://man7.org/linux/man-pages/man2/listen.2.html)
- `while true` serve requests:
  - Create a new socket to read and write data -
    [accept](http://man7.org/linux/man-pages/man2/accept.2.html)
  - Parse the HTTP request -
    [read](http://man7.org/linux/man-pages/man2/read.2.html)
  - Write the response -
    [write](http://man7.org/linux/man-pages/man2/write.2.html)

## **Struct for socket file descriptor**

Create a struct to hold the descriptor to implement `Read`, `Write` and
`Accept`.

```go
// netSocket is a file descriptor for a system socket.
type netSocket struct {
    // System file descriptor.
    fd int
}

func (ns netSocket) Read(p []byte) (int, error) {
    if len(p) == 0 {
        return 0, nil
    }
    n, err := syscall.Read(ns.fd, p)
    if err != nil {
        n = 0
    }
    return n, err
}

// Other methods omitted.
```

## Create, bind and listen on the socket

Next, create the socket and bind it to the localhost port. The details of each
step are below the code block.

```go
// Creates a new socket file descriptor, binds it and listens on it.
func newNetSocket(ip net.IP, port int) (*netSocket, error) {
    // ForkLock docs state that socket syscall requires the lock.
    syscall.ForkLock.Lock()

        // Step 1. Socket creation.
    // AF_INET = Address Family for IPv4
    // SOCK_STREAM = virtual circuit service
    // 0: the protocol for SOCK_STREAM, there's only 1.
    fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
    if err != nil {
        return nil, os.NewSyscallError("socket", err)
    }
    syscall.ForkLock.Unlock()

    // Allow reuse of recently-used addresses.
    if err = syscall.SetsockoptInt(
        fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
        syscall.Close(fd)
        return nil, os.NewSyscallError("setsockopt", err)
    }

    // Step 2. Bind the socket to a port
    sa := &syscall.SockaddrInet4{Port: port}
    copy(sa.Addr[:], ip)
    if err = syscall.Bind(fd, sa); err != nil {
        return nil, os.NewSyscallError("bind", err)
    }

    // Step 3. Listen for incoming connections.
    if err = syscall.Listen(fd, syscall.SOMAXCONN); err != nil {
        return nil, os.NewSyscallError("listen", err)
    }

    return &netSocket{fd: fd}, nil
}
```

The breakdown of steps 1, 2 and 3 from the above code snippet:

1.  `socket(domain, type, protocol)` creates an endpoint for communication and
    returns a descriptor.

    **domain**: selects the protocol (aka address) family. `AF_INET` represents
    IPv4.

    **type**: the semantics of the communication. `SOCK_STREAM` provides the
    sequenced, reliable two-way communication required by HTTP.

    **protocol**: the specific protocol for the socket. Usually 0 because
    there’s only 1 protocol for each type.

2.  `bind(socket, sockaddr, address_len)` assigns a port to the unnamed socket
    created by `socket`.

    **socket**: the descriptor returned by the `socket` syscall.

    **sockaddr**: For `AF_INET`, the IP address and port.

3.  `listen(socket, backlog)` allows `SOCK_STREAM` sockets to accept incoming
    connections.

    **socket**: the descriptor returned by the `socket` syscall.

    **backlog**: the max length for the queue of incoming connections.

## Serve loop

### Accept new connections on the socket

The socket created by `newNetSocket` doesn’t receive data; we need another
socket for that using the `accept` syscall.

```go
// Creates a new netSocket for the next pending connection request.
func (ns *netSocket) Accept() (*netSocket, error) {
    // syscall.ForkLock doc states lock not needed for blocking
    // accept.
    nfd, _, err := syscall.Accept(ns.fd)
    if err == nil {
        syscall.CloseOnExec(nfd)
    }
    if err != nil {
        return nil, err
    }
    return &netSocket{nfd}, nil
}
```

`accept(socket, sockaddr, address_len)` gets the first pending connection,
creates a new socket and allocates a file descriptor. By default, `accept`
blocks until there is an incoming connection.

### Parse read request

Next, parse the HTTP request by `read`ing the newly accepted socket. Use the
`textproto` library to avoid tedious header parsing.

```go
func parseRequest(c *netSocket) (*request, error) {
    b := bufio.NewReader(*c)
    tp := textproto.NewReader(b)
    req := new(request)

    // First line: parse "GET /index.html HTTP/1.0"
    var s string
    s, _ = tp.ReadLine()
    sp := strings.Split(s, " ")
    req.method, req.uri, req.proto = sp[0], sp[1], sp[2]

    // Parse headers
    mimeHeader, _ := tp.ReadMIMEHeader()
    req.header = mimeHeader

    // Parse body
    if req.method == "GET" || req.method == "HEAD" {
        return req, nil
    }
    if len(req.header["Content-Length"]) == 0 {
        return nil, errors.New("no content length")
    }
    length, err := strconv.Atoi(req.header\["Content-Length"\][0])
    if err != nil {
        return nil, err
    }
    body := make([]byte, length)
    if _, err = io.ReadFull(b, body); err != nil {
        return nil, err
    }
    req.body = body
    return req, nil
}
```

## Write response

Write the response in the accepted socket `rw`.

```go
io.WriteString(rw, "HTTP/1.1 200 OK\r\n"+
            "Content-Type: text/html; charset=utf-8\r\n"+
            "Content-Length: 20\r\n"+
            "\r\n"+
            "<h1>hello world</h1>")
```
