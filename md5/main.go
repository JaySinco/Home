package main

import (
	"crypto/md5"
	"fmt"
	"os"
)

func main() {
	if len(os.Args[1:]) != 1 {
		fmt.Println("Usage: md5 [key]")
		return
	}
	fmt.Printf("%x\n", md5.Sum([]byte(os.Args[1])))
}
