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
	Comment     string
	Size        int64
	FileName    string
	PieceLength int64
	Pieces      []string
	InfoHash    string
}

func ParseTorrentFile(f *os.File) (result TorrentFile, err error) {
	reader := bufio.NewReader(f)

	isKey := true
	dict := make(map[string]interface{})
	var lastKey string

	b, err := reader.ReadByte()
	for {
		b, err = reader.ReadByte()
		if err != nil {
			break
		}

		val := c(b)

		switch val {
		case "i":
			intVal := getInt(reader, "e")
			fmt.Println("INTEGER", intVal, isKey)

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
			fmt.Println("LIST", list, isKey)

			if isKey {
				fmt.Println("This should not be possible")
			} else {
				dict[lastKey] = list
			}

		default:
			str := getString(reader)
			fmt.Println("STRING", str, isKey)

			if isKey {
				lastKey = str
				dict[str] = nil
			} else {
				dict[lastKey] = str
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
	// if _, keyExist := dict["announce-list"]; keyExist {
	// 	result.AnnounceList = announceList.([]interface{})
	// }

	if err == io.EOF {
		err = nil
	}

	return result, err
}

func parsePieces(r *bufio.Reader) []string {
	var pieces []string

	piecesLen := getInt(r, ":")
	fmt.Println("Number of pieces", piecesLen)
	r.ReadByte()

	for i := 0; i < piecesLen/20; i++ {
		bytes := make([]byte, 20)
		for i := 0; i < 20; i++ {
			b, _ := r.ReadByte()
			bytes[i] = b
		}
		fmt.Println("HASHES", bytes)

		pieces = append(pieces, hex.EncodeToString(bytes))
	}

	return pieces
}

func getDict(f *os.File, r *bufio.Reader, result *TorrentFile) (dict map[string]interface{}) {
	dict = make(map[string]interface{})
	fmt.Println("=========== INNER DICT START =============")

	isKey := true
	var lastKey string

	start, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		fmt.Println("Error seeking start")
	}

	for {
		b, err := r.ReadByte()
		if err != nil {
			break
		}

		val := c(b)

		switch val {
		case "i":
			intVal := getInt(r, "e")
			fmt.Println("INTEGER", intVal, isKey)

			if isKey {
				fmt.Println("This should not be possible")
			} else {
				dict[lastKey] = intVal
			}

		case "l":
			list := getList(r)
			fmt.Println("LIST", list, isKey)

			if isKey {
				fmt.Println("This should not be possible")
			} else {
				dict[lastKey] = list
			}

		case "e":
			fmt.Println("INNER DICT END")

			end, err := f.Seek(0, io.SeekCurrent)
			if err != nil {
				fmt.Println("Error seeking end")
			}
			fmt.Println("InfoHash bytes len", end, start)
			infoBytes, err := r.Peek(int(end - start))
			if err != nil {
				fmt.Println("Error getting info hash")
			}
			infoHash := fmt.Sprintf("%x", sha1.Sum(infoBytes))
			fmt.Println("INFOHASH", infoHash)

			result.InfoHash = infoHash
			return

		default:
			str := getString(r)
			fmt.Println("STRING", str, isKey)

			if str == "pieces" {
				pieces := parsePieces(r)
				result.Pieces = pieces
				isKey = !isKey
			}

			if isKey {
				lastKey = str
				dict[str] = nil
			} else {
				dict[lastKey] = str
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
		if err != nil {
			fmt.Println("Error in getList")
			break
		}

		val := c(byte)

		switch val {
		case "l":
			fmt.Println("INNER LIST START")
			list = append(list, getList(r))

		case "e":
			fmt.Println("INNER LIST END", list)
			return list

		default:
			if _, err := strconv.Atoi(val); err == nil {
				list = append(list, getString(r))
				fmt.Println("INNER STRING")
			}
		}
	}

	return
}

func getInt(r *bufio.Reader, d string) (intVal int) {
	intAsStr := ""

	for {
		byte, err := r.ReadByte()
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

func getString(r *bufio.Reader) (str string) {
	r.UnreadByte()
	stringLen := getStringLen(r)

	for i := 0; i < stringLen; i++ {
		byte, err := r.ReadByte()
		if err != nil {
			fmt.Println("Error reading byte in getString")
		}

		str += c(byte)
	}

	return
}

func getStringLen(r *bufio.Reader) (len int) {
	lenBuffer := ""
	for {
		byte, err := r.ReadByte()
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
		fmt.Println(len, lenErr)
		fmt.Printf("Error converting string length\n")
	}

	return
}

func c(b byte) string {
	return fmt.Sprintf("%c", b)
}
