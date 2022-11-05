package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"frixaco/bitgorrent/bencode"
)

type Peer struct {
	IP   net.IP
	Port uint16
}

func (p *Peer) String() string {
	return p.IP.String() + ":" + string(rune(p.Port))
}

func main() {
	torrentFileLocation := os.Args[1]
	downloadLocation := os.Args[2]
	fmt.Printf("Torrent file location: %v\n", torrentFileLocation)
	fmt.Printf("Download location: %v\n", downloadLocation)

	file, _ := ioutil.ReadFile(torrentFileLocation)
	torrentInfo, err := bencode.Unmarshal(&file)
	if err != nil {
		fmt.Println(err)
	}

	hashedInfo, infohash, err := bencode.GetInfoHash(&file)
	if err != nil {
		fmt.Println("Failed to get infohash")
	}
	fmt.Println("infohash:", hashedInfo, infohash)

	resp, _ := getTrackerPeers(torrentInfo.(map[string]interface{}), &hashedInfo)
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error reading body", err)
	}

	peerInfo, err := bencode.Unmarshal(&b)
	if err != nil {
		fmt.Println("error decoding peer info", string(b))
		fmt.Println(err)
		return
	}
	fmt.Println("peerinfo", peerInfo)
}

func getTrackerPeers(t map[string]interface{}, infohash *string) (*http.Response, []byte) {
	announce := t["ANNOUNCE"].(string)
	size := t["INFO"].(map[string]interface{})["LENGTH"].(int)
	peerId := generatePeerId()

	base, err := url.Parse(announce)
	if err != nil {
		fmt.Println("Error parsing Announce string", err)
	}

	params := url.Values{
		"info_hash":  []string{*infohash},
		"peer_id":    []string{string(peerId)},
		"port":       []string{strconv.Itoa(6881)},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"compact":    []string{"1"},
		"left":       []string{strconv.Itoa(size)},
		// "ip": []string{}
	}

	passkey := base.Query().Get("uk")
	if passkey != "" {
		params.Add("uk", passkey)
		fmt.Println("PASSKEY", passkey)
	}

	base.RawQuery = params.Encode()
	resp, err := http.Get(base.String())
	fmt.Println(base.String())

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
