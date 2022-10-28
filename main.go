package main

import (
	"fmt"
	"os"

	"frixaco/bitgorrent/helper"
)

func main() {
	helper.ParseFromReader()

	torrentFileLocation := os.Args[1]
	downloadLocation := os.Args[2]
	fmt.Printf("Torrent file location: %v\n", torrentFileLocation)
	fmt.Printf("Download location: %v\n", downloadLocation)

	file, err := os.Open(torrentFileLocation)
	if err != nil {
		fmt.Println("Error opening torrent file")
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		fmt.Println("Error checking file size")
		return
	}
	fmt.Println("size", fileInfo.Size())

	bytes := make([]byte, 1)
	for i := 1582; i < 1683; i++ {
		_, err = file.ReadAt(bytes, int64(i))
		if err != nil {
			fmt.Println("Error reading bytes from torrent file")
			return
		}
		fmt.Println(string(bytes))
	}
}
