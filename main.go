package main

import (
	"encoding/binary"
	"encoding/hex"
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
  "sync"
  "bytes"

	"frixaco/bitgorrent/bencode"
)

type Peer struct {
	IP   net.IP
	Port uint16
}

func (p *Peer) String() string {
	return p.IP.String() + ":" + strconv.Itoa(int(p.Port)) 
}

func main() {
	torrentFileLocation := os.Args[1]
  _ = os.Args[2]
	// fmt.Printf("Torrent file location: %v\n", torrentFileLocation)
	// fmt.Printf("Download location: %v\n", downloadLocation)

	file, _ := ioutil.ReadFile(torrentFileLocation)
	torrentInfo, err := bencode.Unmarshal(&file)
  // fmt.Println("TORRENT", torrentInfo)
	if err != nil {
		fmt.Println(err)
    return
	}

	piecesBytes := torrentInfo.(map[string]interface{})["INFO"].(map[string]interface{})["PIECES"].([]byte)
	_, err = getPieceHashes(piecesBytes, len(piecesBytes))
	if err != nil {
		fmt.Println("error parsing pieces hashes", err)
		return
	}
	// fmt.Println("PIECES", pieceHashes)

	hashedInfo, infohash, err := bencode.GetInfoHash(&file)
	if err != nil {
		fmt.Println("Failed to get infohash")
	}
	fmt.Println("INFOHASH", hashedInfo, infohash)

	resp, peerID, err := getTrackerPeers(torrentInfo.(map[string]interface{}), &hashedInfo)
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
	// fmt.Println("BODY", string(body))

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

  var wg sync.WaitGroup

  // _, err = getPiece(peers[0], []byte(hashedInfo), peerID)
  // if err != nil {
  //   fmt.Println(err)
  // }
  for _, peer := range peers {
    wg.Add(1)
    go getPiece(peer, []byte(hashedInfo), peerID, &wg)
  }

  wg.Wait()
}

func getPiece(peer Peer, infohash []byte, peerID []byte, wg *sync.WaitGroup) ([]byte, error) {
	fmt.Println("start getting piece", peer.String())
  defer wg.Done()

	piece := []byte{}

	conn, err := net.DialTimeout("tcp", peer.String(), 10*time.Second)
  if err != nil {
    fmt.Println(err)
    return nil, err
  }

	fmt.Println("conn", conn)

  conn.SetDeadline(time.Now().Add(3 * time.Second))
	defer conn.SetDeadline(time.Time{}) // Disable the deadline

  pstr := "BitTorrent protocol"
  buf := make([]byte, len(pstr)+49)
	buf[0] = byte(len(pstr))
	curr := 1
	curr += copy(buf[curr:], pstr)
	curr += copy(buf[curr:], make([]byte, 8)) // 8 reserved bytes
	curr += copy(buf[curr:], infohash[:])
	curr += copy(buf[curr:], peerID[:])

	_, err = conn.Write(buf)
	if err != nil {
		return nil, err
	}

  // response from handshake
  lengthBuf := make([]byte, 1)
	_, err = io.ReadFull(conn, lengthBuf)
	if err != nil {
		return nil, err
	}
	pstrlen := int(lengthBuf[0])

	if pstrlen == 0 {
		err := fmt.Errorf("pstrlen cannot be 0")
		return nil, err
	}

	handshakeBuf := make([]byte, 48+pstrlen)
	_, err = io.ReadFull(conn, handshakeBuf)
	if err != nil {
		return nil, err
	}

	var infoHash, peerId [20]byte

	copy(infoHash[:], handshakeBuf[pstrlen+8:pstrlen+8+20])
	copy(peerId[:], handshakeBuf[pstrlen+8+20:])

  infohash2 := infoHash
  fmt.Println("response", infohash2, infohash)

	if err != nil {
		return nil, err
	}
	if !bytes.Equal(infohash2[:], infohash[:]) {
		return nil, fmt.Errorf("Expected infohash %x but got %x", infohash2, infohash)
  }

  buf2 := make([]byte, 4)
	binary.BigEndian.PutUint32(buf2[0:4], 1)
  conn.Write(buf2)

  buf3 := make([]byte, 4)
	binary.BigEndian.PutUint32(buf3[0:4], 2)
  conn.Write(buf3)

  payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], uint32(1))
	binary.BigEndian.PutUint32(payload[4:8], uint32(0))
	binary.BigEndian.PutUint32(payload[8:12], uint32(16384))
  fmt.Println("payload", payload)

	return piece, nil
}

func getTrackerPeers(t map[string]interface{}, infohash *string) (*http.Response, []byte, error) {
	announce := t["ANNOUNCE"].(string)
	size := t["INFO"].(map[string]interface{})["LENGTH"].(int)
	peerID := generatePeerID()

	base, err := url.Parse(announce)
	if err != nil {
		fmt.Println("Error parsing Announce string", err)
	}

	params := url.Values{
		"info_hash":  []string{*infohash},
		"peer_id":    []string{string(peerID)},
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
		return nil, peerID, nil
	}

	return resp, peerID, nil
}

func generatePeerID() (token []byte) {
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

func getPeers(b []byte) ([]Peer, error) {
  peersCount := len(b) / 6
  peers := make([]Peer, peersCount)

  for i := 0; i < peersCount; i++ {
    offset := i * 6
    maybeIP := net.IP(b[offset:offset+4])
    if maybeIP == nil {
      continue
    }
    peers[i].IP = maybeIP
    peers[i].Port = binary.BigEndian.Uint16(b[offset+4:offset+6])
  }

  return peers, nil
}
