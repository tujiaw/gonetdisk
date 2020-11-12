package util

import (
	"crypto/rand"
	"fmt"
	"os"
)

const (
	_           = iota             // ignore first value by assigning to blank identifier
	KB ByteSize = 1 << (10 * iota) // 1 << (10*1)
	MB                             // 1 << (10*2)
	GB                             // 1 << (10*3)
	TB                             // 1 << (10*4)
	PB                             // 1 << (10*5)
	EB                             // 1 << (10*6)
	ZB                             // 1 << (10*7)
	YB                             // 1 << (10*8)
)

func Uuidv4() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		fmt.Println("Error: ", err)
		return "", err
	}
	return fmt.Sprintf("%X%X%X%X%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

type ByteSize float64

func (b ByteSize) Format() string {
	if b >= GB {
		return fmt.Sprintf("%.2f GB", b/GB)
	} else if b >= MB {
		return fmt.Sprintf("%.2f MB", b/MB)
	} else if b >= KB {
		return fmt.Sprintf("%.2f KB", b/KB)
	} else {
		return fmt.Sprintf("%v B", b)
	}
}

func IsDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	fmt.Println("path exist, path:", path, "err:", err)
	return false
}

func If(yes bool, left interface{}, right interface{}) interface{} {
	if yes {
		return left
	}
	return right
}

func IfBool(yes bool, left bool, right bool) bool {
	if yes {
		return left
	}
	return right
}

func StringsIndex(list []string, str string) int {
	for i, s := range list {
		if s == str {
			return i
		}
	}
	return -1
}
