package exiftool

import (
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
)

func IsGBK(data []byte) bool {
	if utf8.Valid(data) {
		return false
	}

	var i int
	for i < len(data) {
		if data[i] <= 0xff {
			// 编码小于等于127,只有一个字节的编码，兼容ASCII
			i++
			continue
		} else {
			// 大于127的使用双字节编码
			if data[i] >= 0x81 &&
				data[i] <= 0xfe &&
				data[i+1] >= 0x40 &&
				data[i+1] <= 0xfe &&
				data[i+1] != 0xf7 {
				i += 2
				continue
			} else {
				return false
			}
		}
	}
	return true
}

func GbkToUtf8(s []byte) ([]byte, error) {
	return simplifiedchinese.GBK.NewDecoder().Bytes(s)
}
