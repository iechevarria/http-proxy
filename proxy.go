package main

import (
	"fmt"
	"syscall"
)

func main() {
	sock, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		panic(err)
	}

	fmt.Println(sock)

	err = syscall.Close(sock)
	if err != nil {
		panic(err)
	}
}
