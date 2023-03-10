package main

import (
	"fmt"
	"syscall"
)

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
