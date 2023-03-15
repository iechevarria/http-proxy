package main

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"syscall"
)

type Request struct {
	Method  string
	Target  string
	Version string
	Headers map[string]string
	Body    []byte
}

type Response struct {
	Version    string
	StatusCode string
	StatusText string
	Headers    map[string]string
	Body       []byte
}

func ReadHeaders(scanner *bufio.Scanner) map[string]string {
	m := make(map[string]string)
	for scanner.Scan() {
		headerLine := strings.Split(scanner.Text(), ": ")
		if len(headerLine) != 2 {
			break
		}
		m[headerLine[0]] = headerLine[1]
		fmt.Println("RUN")
	}
	return m
}

func ReadRequest(request []byte) (*Request, error) {
	reader := bytes.NewReader(request)
	scanner := bufio.NewScanner(reader)

	// Read start-line
	scanner.Scan()
	startLine := strings.Split(scanner.Text(), " ")
	if len(startLine) != 3 {
		return nil, fmt.Errorf("Wrong start line HTTP: %s", startLine)
	}
	var r Request
	r.Method = startLine[0]
	r.Target = startLine[1]
	r.Version = startLine[2]
	fmt.Println(reader.Len())

	// Read headers and body
	r.Headers = ReadHeaders(scanner)

	fmt.Println(reader.Len())
	r.Body = make([]byte, reader.Len())
	if reader.Len() > 0 {
		_, err := reader.Read(r.Body)
		if err != nil {
			return nil, err
		}
	}

	return &r, nil
}

func ReadResponse(response []byte) (*Response, error) {
	reader := bytes.NewReader(response)
	scanner := bufio.NewScanner(reader)

	scanner.Scan()
	statusLine := strings.Split(scanner.Text(), " ")
	if len(statusLine) != 3 {
		return nil, fmt.Errorf("Wrong status line HTTP: %s", statusLine)
	}
	var r Response
	r.Version = statusLine[0]
	r.StatusCode = statusLine[1]
	r.StatusText = statusLine[2]

	// Read headers and body
	r.Headers = ReadHeaders(scanner)

	r.Body = make([]byte, reader.Len())
	if reader.Len() > 0 {
		_, err := reader.Read(r.Body)
		if err != nil {
			return nil, err
		}
	}

	return &r, nil
}

func main() {
	fmt.Println("proxy starting")

	// list on localhost 8080
	sock, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		panic(err)
	}
	err = syscall.Bind(sock, &syscall.SockaddrInet4{Port: 8080, Addr: [4]byte{127, 0, 0, 1}})
	if err != nil {
		panic(err)
	}
	err = syscall.Listen(sock, 1)
	if err != nil {
		panic(err)
	}
	defer syscall.Close(sock)

	for {
		// accept conn + read incoming req
		nfd, _, err := syscall.Accept(sock)
		if err != nil {
			panic(err)
		}
		buf := make([]byte, 4096)
		n, err := syscall.Read(nfd, buf)
		if err != nil {
			panic(err)
		}
		fmt.Println("-> got client request")
		req, err := ReadRequest(buf[:n])
		if err != nil {
			panic(err)
		}
		fmt.Printf("%+v\n", req)

		// proxy to server
		var sock2 int
		success := false
		// retry proxy conn on EINTR
		for success != true {
			sock2, err = syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
			if err != nil {
				panic(err)
			}

			err = syscall.Connect(sock2, &syscall.SockaddrInet4{Port: 9000, Addr: [4]byte{127, 0, 0, 1}})
			if err == syscall.EINTR {
				syscall.Close(sock2)
				continue
			}
			if err != nil {
				panic(err)
			}
			success = true
		}
		_, err = syscall.Write(sock2, buf[:n])
		if err != nil {
			panic(err)
		}
		fmt.Println("fwd to server ->")

		// read the whole response
		var res string
		n = 1
		for n != 0 {
			n, _, err = syscall.Recvfrom(sock2, buf, 0)
			if err != nil {
				panic(err)
			}
			res += string(buf[:n])
		}
		syscall.Close(sock2)

		resStruct, err := ReadResponse([]byte(res))
		if err != nil {
			panic(err)
		}
		fmt.Printf("%+v\n", resStruct)

		fmt.Println("read server response <-")

		// forward response back
		_, err = syscall.Write(nfd, []byte(res))
		if err != nil {
			panic(err)
		}
		syscall.Close(nfd)
		fmt.Println("<- response sent to client")
	}
}
