package helper

import (
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strconv"
)

type TorrentFile struct {
	Announce string
	// AnnounceList []interface{}
	Peers       string
	Comment     string
	Size        int64
	FileName    string
	PieceLength int64
	Pieces      []string
	InfoHash    [20]byte
}

func ParseBencodedData(reader *bufio.Reader, f *os.File) (result TorrentFile, err error) {
	isKey := true
	dict := make(map[string]interface{})
	var lastKey string

	b, err := reader.ReadByte()
	counter++
	for {
		b, err = reader.ReadByte()
		counter++
		if err != nil {
			break
		}

		val := c(b)

		switch val {
		case "i":
			intVal := getInt(reader, "e")
			// fmt.Println("INTEGER", intVal, isKey)

			if isKey {
				fmt.Println("This should not be possible")
			} else {
				dict[lastKey] = intVal
			}

		case "d":
			subDict := getDict(f, reader, &result)

			if isKey {
				fmt.Println("This should not be possible")
			} else {
				dict[lastKey] = subDict
			}

		case "l":
			list := getList(reader)
			// fmt.Println("LIST", list, isKey)

			if isKey {
				fmt.Println("This should not be possible")
			} else {
				dict[lastKey] = list
			}

		default:
			str, buf := getString(reader)
			fmt.Println("STRING", str, isKey)

			if isKey {
				lastKey = str
				dict[str] = nil
			} else {
				if lastKey == "peers" {
					dict[lastKey] = buf
				} else {
					dict[lastKey] = str
				}
			}
		}

		isKey = !isKey
		// fmt.Println("result", dict)
	}

	if _, keyExist := dict["announce"]; keyExist {
		result.Announce = dict["announce"].(string)
	}
	if _, keyExist := dict["size"]; keyExist {
		result.Size = dict["size"].(int64)
	}
	if _, keyExist := dict["name"]; keyExist {
		result.FileName = dict["name"].(string)
	}
	if _, keyExist := dict["comment"]; keyExist {
		result.Comment = dict["comment"].(string)
	}
	if _, keyExist := dict["length"]; keyExist {
		result.PieceLength = dict["length"].(int64)
	}
	if _, keyExist := dict["peers"]; keyExist {
		result.Peers = dict["peers"].(string)
	}
	// if _, keyExist := dict["announce-list"]; keyExist {
	// 	result.AnnounceList = announceList.([]interface{})
	// }

	if err == io.EOF {
		err = nil
	}

	return result, err
}

var start = 0

func parsePieces(r *bufio.Reader) []string {
	var pieces []string

	piecesLen := getInt(r, ":")
	// fmt.Println("Number of pieces", piecesLen)
	r.ReadByte()
	counter++

	for i := 0; i < piecesLen/20; i++ {
		bytes := make([]byte, 20)
		for i := 0; i < 20; i++ {
			b, _ := r.ReadByte()
			counter++
			bytes[i] = b
		}

		pieces = append(pieces, hex.EncodeToString(bytes))
	}
	// fmt.Println("HASHES", len(pieces))

	return pieces
}

var counter int = 0

func getDict(f *os.File, r *bufio.Reader, result *TorrentFile) (dict map[string]interface{}) {
	dict = make(map[string]interface{})
	// fmt.Println("=========== INNER DICT START =============")

	isKey := true
	var lastKey string
	start = counter

	for {
		b, err := r.ReadByte()
		counter++
		if err != nil {
			break
		}

		val := c(b)

		switch val {
		case "i":
			intVal := getInt(r, "e")
			// fmt.Println("INTEGER", intVal, isKey)

			if isKey {
				fmt.Println("This should not be possible")
			} else {
				dict[lastKey] = intVal
			}

		case "l":
			list := getList(r)
			// fmt.Println("LIST", list, isKey)

			if isKey {
				fmt.Println("This should not be possible")
			} else {
				dict[lastKey] = list
			}

		case "e":
			// fmt.Println("INNER DICT END")

			// fmt.Println("Number of bytes in infohash", start, counter)
			var infohashBytes []byte
			for i := start - 1; i < counter-1; i++ {
				bb := make([]byte, 1)
				f.ReadAt(bb, int64(i))
				infohashBytes = append(infohashBytes, bb[0])
			}

			infoHash := sha1.Sum(infohashBytes)

			// fmt.Println("INFOHASH", infoHash)
			result.InfoHash = infoHash

			return

		default:
			str, buf := getString(r)
			// fmt.Println("STRING", str, isKey)

			if str == "pieces" {
				pieces := parsePieces(r)
				result.Pieces = pieces
				isKey = !isKey
			}

			if isKey {
				lastKey = str
				dict[str] = nil
			} else {
				if lastKey == "peers" {
					dict[lastKey] = buf
				} else {
					dict[lastKey] = str
				}
			}
		}

		isKey = !isKey
		// fmt.Println("dict", dict)
	}

	return
}

func getList(r *bufio.Reader) (list []interface{}) {
	for {
		byte, err := r.ReadByte()
		counter++
		if err != nil {
			fmt.Println("Error in getList")
			break
		}

		val := c(byte)

		switch val {
		case "l":
			// fmt.Println("INNER LIST START")
			list = append(list, getList(r))

		case "e":
			// fmt.Println("INNER LIST END", list)
			return list

		default:
			if _, err := strconv.Atoi(val); err == nil {
				str, _ := getString(r)
				list = append(list, str)
				// fmt.Println("INNER STRING")
			}
		}
	}

	return
}

func getInt(r *bufio.Reader, d string) (intVal int) {
	intAsStr := ""

	for {
		byte, err := r.ReadByte()
		counter++
		if err != nil {
			fmt.Println("Error reading byte while getting integer value")
		}

		v := c(byte)
		if v == d || err == io.EOF {
			break
		}

		intAsStr += v
	}

	intVal, intErr := strconv.Atoi(intAsStr)
	if intErr != nil {
		fmt.Printf("Error converting str to int\n")
		return intVal
	}

	return
}

func getString(r *bufio.Reader) (str string, bytebuf string) {
	r.UnreadByte()
	counter--
	stringLen := getStringLen(r)

	tempbuf := ""

	for i := 0; i < stringLen; i++ {
		b, err := r.ReadByte()
		counter++
		if err != nil {
			fmt.Println("Error reading byte in getString")
		}

		str += c(b)
		tempbuf += hex.EncodeToString([]byte{b})
	}

	return str, tempbuf
}

func getStringLen(r *bufio.Reader) (len int) {
	lenBuffer := ""
	for {
		byte, err := r.ReadByte()
		counter++
		if err != nil {
			fmt.Println("Error reading byte while calculating string length")
		}

		v := c(byte)
		if v == ":" || err == io.EOF {
			break
		}

		lenBuffer += v
	}

	len, lenErr := strconv.Atoi(lenBuffer)
	if lenErr != nil {
		// fmt.Println(len, lenErr)
		fmt.Printf("Error converting string length\n")
	}

	return
}

func c(b byte) string {
	return fmt.Sprintf("%c", b)
}
