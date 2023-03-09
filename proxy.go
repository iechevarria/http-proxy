package main

import (
	"fmt"
	"syscall"
)

func main() {
	fmt.Println("HELLO WORLD")

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

	nfd, sa, err := syscall.Accept(sock)
	if err != nil {
		panic(err)
	}
	fmt.Println(nfd)
	fmt.Println(sa)

	fmt.Println(sock)
}
