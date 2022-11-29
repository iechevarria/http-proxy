package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	sock, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		panic(err)
	}
	err = syscall.Bind(sock, &syscall.SockaddrInet4{Port: 8080})
	if err != nil {
		panic(err)
	}

	// close socket on interrupt
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		fmt.Println("\nClosing socket")
		syscall.Close(sock)
		os.Exit(0)
	}()

	err = syscall.Listen(sock, 5)
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

		ips, err := net.LookupIP(string(buf[:n-1]))
		// for _, ip := range ips {
		// 	if
		// }
		fmt.Println(err)
		fmt.Println(ips)

		fmt.Println(destAddr)

		err = syscall.Sendto(nfd, buf[:n], 0, destAddr)
		if err != nil {
			panic(err)
		}
	}
}
