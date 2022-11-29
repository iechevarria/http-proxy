package main

import (
	"syscall"
)

/*
As a first step, write a program which accepts a TCP connection and simply responds with whatever it reads.
You should be able to run it, nc into it, and see that it echos any data.
The purpose of this step is to ensure that you’re utilizing the correct socket-related system calls to act as a server: listen, accept, recv and send.
*/
func main() {
	sock, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		panic(err)
	}
	err = syscall.Bind(sock, &syscall.SockaddrInet4{Port: 8080})
	if err != nil {
		panic(err)
	}
	err = syscall.Listen(sock, 1)
	if err != nil {
		panic(err)
	}
	nfd, destAddr, err := syscall.Accept(sock)
	if err != nil {
		panic(err)
	}

	buf := make([]byte, 2048)
	n := 1
	for n != 0 {
		n, _, err = syscall.Recvfrom(nfd, buf, 0)
		if err != nil {
			panic(err)
		}
		err = syscall.Sendto(nfd, buf[:n], 0, destAddr)
		if err != nil {
			panic(err)
		}
	}

	err = syscall.Close(sock)
	if err != nil {
		panic(err)
	}
}
