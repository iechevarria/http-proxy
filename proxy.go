package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"syscall"
)

type Request struct {
	Method  string
	Target  string
	Version string
	Headers map[string]string
	Body    string
}

type Response struct {
	Version    string
	StatusCode string
	StatusText string
	Headers    map[string]string
	Body       string
}

func ReadHeaders(reader *bufio.Reader) (map[string]string, error) {
	m := make(map[string]string)
	for {
		hl, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		headerLine := strings.Split(hl, ": ")
		if len(headerLine) != 2 {
			break
		}
		m[headerLine[0]] = headerLine[1]
	}
	return m, nil
}

func ReadRequest(request []byte) (*Request, error) {
	reader := bufio.NewReader(bytes.NewReader(request))

	// Read start-line
	l, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	startLine := strings.Split(l, " ")
	if len(startLine) != 3 {
		return nil, fmt.Errorf("Wrong start line HTTP: %s", startLine)
	}
	var r Request
	r.Method = startLine[0]
	r.Target = startLine[1]
	r.Version = startLine[2]

	// Read headers and body
	r.Headers, err = ReadHeaders(reader)
	if err != nil {
		return nil, err
	}
	b, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	r.Body = string(b)

	return &r, nil
}

func ReadResponse(response []byte) (*Response, error) {
	reader := bufio.NewReader(bytes.NewReader(response))

	// Read status line
	l, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	statusLine := strings.Split(l, " ")
	if len(statusLine) != 3 {
		return nil, fmt.Errorf("Wrong status line HTTP: %s", statusLine)
	}
	var r Response
	r.Version = statusLine[0]
	r.StatusCode = statusLine[1]
	r.StatusText = statusLine[2]

	// Read headers and body
	r.Headers, err = ReadHeaders(reader)
	if err != nil {
		return nil, err
	}
	b, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	r.Body = string(b)

	return &r, nil
}

func NewConnectedSocket(port int, addr [4]byte) (int, error) {
	var sock int
	var err error

	// retry proxy conn on EINTR
	for {
		sock, err = syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
		if err != nil {
			return -1, err
		}

		err = syscall.Connect(sock, &syscall.SockaddrInet4{Port: port, Addr: addr})
		if err == syscall.EINTR {
			syscall.Close(sock)
			continue
		}
		if err != nil {
			return -1, err
		}
		break
	}

	syscall.SetsockoptInt(sock, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
	return sock, nil
}

func ProxyRequest(sock int, request []byte) ([]byte, error) {
	// send client request
	_, err := syscall.Write(sock, request)
	if err != nil {
		return nil, fmt.Errorf("ProxyRequest error writing to sock: %s", err)
	}

	// read the whole server response
	var response []byte
	buf := make([]byte, 2048)
	n := 1
	for n != 0 {
		n, _, err = syscall.Recvfrom(sock, buf, 0)
		if err != nil {
			return nil, fmt.Errorf("ProxyRequest error reading from sock: %s", err)
		}
		response = append(response, buf[:n]...)
	}

	return response, nil
}

func HandleRequest(sock int, backendSock int, cache map[string][]byte) {
	// TODO: remove
	prefix := "chumba"

	buf := make([]byte, 2048)

	n, err := syscall.Read(sock, buf)
	if err != nil {
		panic(err)
	}

	req, err := ReadRequest(buf[:n])
	if err != nil {
		panic(err)
	}

	// proxy request along or get cached result
	var res []byte
	var ok bool
	if strings.HasPrefix(req.Target, prefix) {
		res, ok = cache[req.Target]
		if ok {
			fmt.Printf("Returning cached result for %s\n", req.Target)
		} else {
			res, err = ProxyRequest(backendSock, buf[:n])
			if err != nil {
				panic(err)
			}
			cache[req.Target] = res
			fmt.Printf("Cached result for %s\n", req.Target)
		}
	} else {
		res, err = ProxyRequest(backendSock, buf[:n])
		if err != nil {
			panic(err)
		}
	}

	// resStruct, err := ReadResponse(res)
	// if err != nil {
	// 	panic(err)
	// }

	// forward response back
	_, err = syscall.Write(sock, res)
	if err != nil {
		panic(err)
	}
	syscall.Close(sock)
}

func worker(id int, socks <-chan int, cache map[string][]byte) {
	// open conn to backend
	backendSock, err := NewConnectedSocket(9000, [4]byte{127, 0, 0, 1})
	if err != nil {
		panic(err)
	}

	fmt.Printf("Worker %v ready\n", id)
	fmt.Printf("Worker %v backendSock: %v\n", id, backendSock)

	for s := range socks {
		fmt.Printf("Worker %v starting request\n", id)
		fmt.Printf("Worker %v backendSock: %v\n", id, backendSock)
		HandleRequest(s, backendSock, cache)
		fmt.Printf("Worker %v finished request\n", id)
	}
}

func main() {
	fmt.Println("Proxy starting")
	// TODO: make this a response obj + marshal
	cache := make(map[string][]byte)

	// listen on localhost 8000
	sock, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		panic(err)
	}
	err = syscall.Bind(sock, &syscall.SockaddrInet4{Port: 8000, Addr: [4]byte{127, 0, 0, 1}})
	if err != nil {
		panic(err)
	}
	err = syscall.Listen(sock, 64)
	if err != nil {
		panic(err)
	}
	defer syscall.Close(sock)

	// idk how much to do here
	jobs := make(chan int, 64)
	for w := 1; w <= 1; w++ {
		go worker(w, jobs, cache)
	}

	for {
		// accept conn + send sock to workers
		nfd, _, err := syscall.Accept(sock)
		if err != nil {
			panic(err)
		}
		// fmt.Printf("New conn sock: %v\n", nfd)
		jobs <- nfd
	}
}
