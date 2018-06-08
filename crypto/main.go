package main

import (
	"crypto/md5"
	"fmt"
	"os"
)

func main() {
	if len(os.Args[1:]) != 2 {
		fmt.Println("Usage: hash [method] [key]")
		return
	}
	switch os.Args[1] {
	case "md5":
		fmt.Printf("%x\ns", md5.Sum([]byte(os.Args[2])))
	default:
		fmt.Printf("unsupport hash method: '%s'\n", os.Args[1])
		return
	}
}
