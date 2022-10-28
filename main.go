package main

import (
	"fmt"
	"os"

	"frixaco/bitgorrent/helper"
)

func main() {

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

	torrentFile, parseErr := helper.ParseTorrentFile(file)
	if parseErr != nil {
		fmt.Println("Error parsing torrent file")
		fmt.Println(parseErr)
	}
	fmt.Printf("%+v", torrentFile)
}
