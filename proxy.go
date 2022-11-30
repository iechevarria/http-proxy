package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func panicAndCleanUp(socks []int, err error) {
	cleanUp(socks)
	panic(err)
}

func cleanUp(socks []int) {
	for _, s := range socks {
		fmt.Printf("Closing socket fd %v\n", s)
		err := syscall.Close(s)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func main() {
	sock, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		panicAndCleanUp([]int{sock}, err)
	}
	err = syscall.Bind(sock, &syscall.SockaddrInet4{Port: 8080})
	if err != nil {
		panicAndCleanUp([]int{sock}, err)
	}

	// Close socket on interrupt
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		cleanUp([]int{sock})
		os.Exit(0)
	}()

	err = syscall.Listen(sock, 1)
	if err != nil {
		panicAndCleanUp([]int{sock}, err)
	}
	nfd, _, err := syscall.Accept(sock)
	if err != nil {
		panicAndCleanUp([]int{sock}, err)
	}

	buf := make([]byte, 2048)
	n, _, err := syscall.Recvfrom(nfd, buf, 0)
	if err != nil {
		panicAndCleanUp([]int{sock}, err)
	}
	fmt.Println(string(buf[:n]))

	// Make a new socket, forward request
	sock2, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		panicAndCleanUp([]int{sock, sock2}, err)
	}
	err = syscall.Bind(sock2, &syscall.SockaddrInet4{Port: 8088})
	if err != nil {
		panicAndCleanUp([]int{sock, sock2}, err)
	}
	err = syscall.Connect(sock2, &syscall.SockaddrInet4{Port: 9000, Addr: [4]byte{127, 0, 0, 1}})
	if err != nil {
		panicAndCleanUp([]int{sock, sock2}, err)
	}
	_, err = syscall.Write(sock2, buf[:n])
	if err != nil {
		panicAndCleanUp([]int{sock, sock2}, err)
	}
	recvN, _, err := syscall.Recvfrom(sock2, buf, 0)
	if err != nil {
		panicAndCleanUp([]int{sock, sock2}, err)
	}
	cleanUp([]int{sock2})
	fmt.Println(string(buf[:recvN]))

	_, err = syscall.Write(nfd, buf[:recvN])
	if err != nil {
		panicAndCleanUp([]int{sock}, err)
	}

	cleanUp([]int{sock})
}
