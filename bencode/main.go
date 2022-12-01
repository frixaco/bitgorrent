package bencode

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type Decoder struct {
	data             []byte
	cursor           int
	infohashStartIdx int
	infohashEndIdx   int
}

func Unmarshal(data *[]byte) (interface{}, error) {
	d := Decoder{data: *data}
	return d.Decode()
}

func (d *Decoder) Decode() (interface{}, error) {
	b := d.data[d.cursor]

	switch b {
	case 'i':
		return d.decodeInt()
	case 'l':
		d.cursor += 1
		l := []interface{}{}

		for {
			if d.data[d.cursor] == 'e' {
				d.cursor += 1
				return l, nil
			}

			v, err := d.Decode()
			if err != nil {
				return nil, err
			}

			l = append(l, v)
		}
	case 'd':
		d.cursor += 1
		dict := map[string]interface{}{}

		for {
			if d.data[d.cursor] == 'e' {
				d.cursor += 1
				return dict, nil
			}

			key, err := d.decodeStr()
			if err != nil {
				fmt.Println("bencode: error decoding dict key", err)
				return nil, err
			}

      if key == "info" {
        d.infohashStartIdx = d.cursor
      }

			value, err := d.Decode()
			if err != nil {
				fmt.Println("bencode: error decoding dict value", err)
				return nil, err
			}
      
      if key == "info" {
        d.infohashEndIdx = d.cursor
      }

			dict[strings.ReplaceAll(strings.ToUpper(key.(string)), "-", "_")] = value
		}

	default:
		return d.decodeStr()
	}
}

func isSHA1(b byte) bool {
	return b > unicode.MaxASCII
}

func (d *Decoder) decodeStr() (interface{}, error) {
	lenIdx := bytes.IndexByte(d.data[d.cursor:], ':')
	if lenIdx == -1 {
		return nil, errors.New("bencode: error getting string length")
	}

	lenSlice := d.data[d.cursor : d.cursor+lenIdx]
	d.cursor += lenIdx + 1

	l, err := strconv.Atoi(string(lenSlice))

	if err != nil {
		return nil, errors.New("bencode: error converting str len to int")
	}

	v := d.data[d.cursor : d.cursor+l]

	if len(v) == 0 || d.cursor == len(d.data) {
		return nil, nil
	}

	var parsedStr string
	if isSHA1(v[0]) || isSHA1(v[1]) || isSHA1(v[2]) || isSHA1(v[3]) {
		d.cursor += l
		return v, nil

		// FOR PARSING REQUEST WITH PEER INFO
		// pieces := hex.EncodeToString(v)
		// d.cursor += l

		// FOR PARSING TORRENT FILE INFO.PIECES
		// pieces := make([]string, l/20)
		// c := 0
		// for i := 0; i < l/20; i++ {
		// 	c++
		// 	hash := hex.EncodeToString(d.data[d.cursor : d.cursor+20])
		// 	pieces[i] = hash
		// 	d.cursor += 20
		// }
		// fmt.Println("pieces", pieces)
		// return pieces, nil
	}

	d.cursor += l

	parsedStr = string(v)
	return parsedStr, nil
}

func (d *Decoder) decodeInt() (interface{}, error) {
	d.cursor += 1
	idx := bytes.IndexByte(d.data[d.cursor:], 'e')
	if idx == -1 {
		return nil, errors.New("bencode: error parsing int")
	}

	strSlice := string(d.data[d.cursor : d.cursor+idx])
	v, err := strconv.Atoi(strSlice)
	if err != nil {
		return nil, errors.New("bencode: invalid int")
	}
	d.cursor += idx + 1
	return v, nil
}

func (d *Decoder) Encode(data interface{}) (interface{}, error) {
	return nil, nil
}

func GetInfoHash(data *[]byte) (string, string, error) {
	d := Decoder{data: *data}
	_, _ = d.Decode()

	infohashBytes := d.data[d.infohashStartIdx:d.infohashEndIdx]
  fmt.Println("BYTES", string(infohashBytes))
	infoHash := sha1.Sum(infohashBytes)

	hash := hex.EncodeToString(infoHash[:])

	return string(infoHash[:]), hash, nil
}
