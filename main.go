package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"frixaco/bitgorrent/helper"
)

type Peer struct {
	IP   net.IP
	Port uint16
}

func (p *Peer) String() string {
	return p.IP.String() + ":" + string(p.Port)
}

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

	reader := bufio.NewReader(file)
	torrentFile, parseErr := helper.ParseBencodedData(reader, file)
	if parseErr != nil {
		fmt.Println("Error parsing torrent file")
		fmt.Println(parseErr)
	}
	fmt.Printf("%+v\n", torrentFile.Announce)

	peersFromTracker, peerId := getTrackerPeers(&torrentFile, 6881)
	b, err := io.ReadAll(peersFromTracker.Body)
	if err != nil {
		fmt.Println("Error parsing body", err)
	}
	fmt.Println(string(b))
	fmt.Println("peerId", peerId)
	reader2 := bufio.NewReader(bytes.NewReader(b))
	parsedResponse, parseErr := helper.ParseBencodedData(reader2, nil)
	if parseErr != nil {
		fmt.Println("Error parsing response", parseErr)
	}
	// fmt.Println("Peers", hex.EncodeToString(parsedResponse.Peers))
	const peerSize = 6 // 4 for IP, 2 for port
	peersBin, _ := hex.DecodeString(parsedResponse.Peers)
	numPeers := len(peersBin) / peerSize
	if len(peersBin)%peerSize != 0 {
		fmt.Errorf("Received malformed peers")
		return
	}
	peers := make([]Peer, numPeers)
	for i := 0; i < numPeers; i++ {
		offset := i * peerSize
		peers[i].IP = net.IP(peersBin[offset : offset+4])
		peers[i].Port = binary.BigEndian.Uint16(peersBin[offset+4 : offset+6])
	}
	fmt.Println("peers", peers)

	// conn, err := net.DialTimeout("tcp", peers[0].String(), 3*time.Second)
	// if err != nil {
	// 	return nil, err
	// }
}

func getTrackerPeers(t *helper.TorrentFile, port int) (*http.Response, []byte) {
	base, err := url.Parse(t.Announce)
	if err != nil {
		fmt.Println("Error parsing Announce string", err)
	}

	peerId := generatePeerId()
	params := url.Values{
		"info_hash": []string{string(t.InfoHash[:])},
		"peer_id":   []string{string(peerId)},
		// "ip": []string{}
		"port":       []string{strconv.Itoa(port)},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"compact":    []string{"1"},
		"left":       []string{strconv.Itoa(int(t.Size))},
	}

	passkey := base.Query().Get("uk")
	if passkey != "" {
		params.Add("uk", passkey)
		fmt.Println("PASSKEY", passkey)

	}

	base.RawQuery = params.Encode()
	fmt.Println("url string", t.Announce)

	resp, err := http.Get(base.String())
	if err != nil {
		fmt.Println("Error making GET request", err)
	}

	return resp, peerId
}

func generatePeerId() (token []byte) {
	rand.Seed(time.Now().UnixNano())
	token = make([]byte, 20)
	rand.Read(token)
	return
}
