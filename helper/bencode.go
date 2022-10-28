package helper

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strconv"
)

type TorrentFile struct {
	Announce    string
	Comment     string
	Size        int64
	FileName    string
	PieceLength int64
	Pieces      []string
}

func ParseTorrentFile(f *os.File) (result TorrentFile, err error) {
	reader := bufio.NewReader(f)

	isKey := true
	dict := make(map[string]interface{})

	b, err := reader.ReadByte()
	for {
		b, err = reader.ReadByte()
		if err != nil {
			break
		}

		val := c(b)

		switch val {
		case "i":
			fmt.Println("INTEGER")
			intVal := getInt(reader)

			if isKey {
				fmt.Println("This should not be possible")
				// keyVal[intVal] = nil
			} else {
				for k, v := range dict {
					if v == nil {
						dict[k] = intVal
					}
					break
				}
			}

		case "l":
			fmt.Println("LIST")
			list := getList(reader)

			if isKey {
				fmt.Println("This should not be possible")
				// keyVal[intVal] = nil
			} else {
				for k, v := range dict {
					if v == nil {
						dict[k] = list
					}
					break
				}
			}

		default:
			fmt.Println("STRING")
			str := getString(reader)

			if str == "pieces" {
				parsePieces(reader, &result.Pieces)
				isKey = !isKey
			}

			if isKey {
				dict[str] = nil
			} else {
				for k, v := range dict {
					if v == nil {
						dict[k] = str
					}
					break
				}
			}
		}

		isKey = !isKey
		fmt.Println("result", dict)
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

	return result, err
}

func parsePieces(r *bufio.Reader, pieces *[]string) {
	p := *pieces

	for {
		bytes := make([]byte, 20)
		for i := 0; i < 20; i++ {
			b, _ := r.ReadByte()
			bytes[i] = b
		}

		p = append(p, hex.EncodeToString(bytes))
	}
}

func getList(r *bufio.Reader) (list []interface{}) {
	byte, err := r.ReadByte()
	for err == nil {
		val := c(byte)
		if val == "e" || err == io.EOF {
			break
		}

		switch val {
		case "l":
			fmt.Println("LIST")
			list = append(list, getList(r))

		default:
			fmt.Println("STRING")
			list = append(list, getString(r))
		}
	}

	return
}

func getInt(r *bufio.Reader) (intVal int) {
	intAsStr := ""

	for {
		byte, err := r.ReadByte()
		if err != nil {
			fmt.Println("Error reading byte while getting integer value")
		}

		v := c(byte)
		if v == "e" || err == io.EOF {
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
