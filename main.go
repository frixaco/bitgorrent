package main

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
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

	piecesBytes := torrentInfo.(map[string]interface{})["INFO"].(map[string]interface{})["PIECES"].([]byte)
	pieceHashes, err := getPieceHashes(piecesBytes, len(piecesBytes))
	if err != nil {
		fmt.Println("error parsing pieces hashes", err)
		return
	}
	fmt.Println("PIECES", pieceHashes)

	hashedInfo, infohash, err := bencode.GetInfoHash(&file)
	if err != nil {
		fmt.Println("Failed to get infohash")
	}
	fmt.Println("INFOHASH", infohash)

	resp, _, err := getTrackerPeers(torrentInfo.(map[string]interface{}), &hashedInfo)
	if err != nil {
		fmt.Println("error making request", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error reading body", err)
		return
	}
	fmt.Println("BODY", string(body))

	tracker, err := bencode.Unmarshal(&body)
	if err != nil {
		fmt.Println("error decoding peer info", err)
		return
	}
	fmt.Println("TRACKER", tracker)

	peers, err := getPeers(
		tracker.(map[string]interface{})["PEERS"].([]byte),
	)
	if err != nil {
		fmt.Println("error getting peers", err)
	}
	fmt.Println("PEERS", peers)

	// downloadedPiece, err := getPiece(peers.([]byte))
	// if err != nil {
	// 	fmt.Println("error getting piece")
	// 	return
	// }
	// fmt.Println("downloaded piece", downloadedPiece)
}

func getPiece(peers []byte) ([]byte, error) {
	peer := string(peers[0:6])
	conn, err := net.DialTimeout("tcp", peer, 3*time.Second)
	if err != nil {
		return nil, err
	}
	fmt.Println("conn", conn)
	piece := []byte{}
	return piece, nil
}

func getTrackerPeers(t map[string]interface{}, infohash *string) (*http.Response, []byte, error) {
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
	fmt.Println("GET REQUEST URL", base.String())

	if err != nil {
		fmt.Println("Error making GET request", err)
		return nil, peerId, nil
	}

	return resp, peerId, nil
}

func generatePeerId() (token []byte) {
	rand.Seed(time.Now().UnixNano())
	token = make([]byte, 20)
	rand.Read(token)
	return
}

func getPieceHashes(d []byte, l int) ([]string, error) {
	pieces := make([]string, l/20)
	c := 0
	for i := 0; i < l/20; i++ {
		hash := hex.EncodeToString(d[c : c+20])
		pieces[i] = hash
		c += 20
	}
	return pieces, nil
}

func getPeers(peersBin []byte) ([]Peer, error) {
	const peerSize = 6 // 4 for IP, 2 for port
	numPeers := len(peersBin) / peerSize
	if len(peersBin)%peerSize != 0 {
		return nil, errors.New("error getting peers number")
	}
	peers := make([]Peer, numPeers)
	for i := 0; i < numPeers; i++ {
		offset := i * peerSize
		peers[i].IP = net.IP(peersBin[offset : offset+4])
		peers[i].Port = binary.BigEndian.Uint16(peersBin[offset+4 : offset+6])
	}
	return peers, nil
}
