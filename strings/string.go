package strings

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
)

var realEscape map[string]string = map[string]string{"\\": "\\\\", "'": `\'`, "\\0": "\\\\0", "\n": "\\n", "\r": "\\r", `"`: `\"`, "\x1a": "\\Z"}

func RemoveAllNonASCII(char string) string {
	t := strings.Map(func(r rune) rune {
		if r > unicode.MaxASCII {
			return -1
		}
		return r
	}, char)
	fmt.Println(t)
	return t
}

func RemoveEmptyObjectJSON(jsonString string, tags []string) (string, error) {
	var i interface{}
	if err := json.Unmarshal([]byte(jsonString), &i); err != nil {
		return "", err
	}
	for _, tag := range tags {
		removeObject(i, tag)
	}
	output, err := json.Marshal(i)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func removeObject(i interface{}, tag string) {
	if m, ok := i.(map[string]interface{}); ok {
		for k, v := range m {
			if k == tag {
				if v == "0001-01-01T00:00:00Z" || fmt.Sprintf("%s", v) == "map[]" || fmt.Sprintf("%s", v) == "map[LSB:%!s(float64=0) MSB:%!s(float64=0)]" {
					delete(m, tag)
				}
			} else {
				if n, ok := v.([]interface{}); ok {
					for _, v2 := range n {
						removeObject(v2, tag)
					}
				}
			}
		}

	}
}

// SQLRealEscape escape string for sql
func SQLRealEscape(str string) string {
	for b, a := range realEscape {
		str = strings.Replace(str, b, a, -1)
	}
	return str
}

type StarFormat int32

const (
	StarFormatASC  StarFormat = 1
	StarFormatDESC StarFormat = 2
)

func GetMaskAsterisk(str string, digit int, format StarFormat) string {
	s := []rune(str)
	masStr := len(s)
	fst := masStr - digit
	str = ""
	for i := 0; i < fst; i++ {
		str += "*"
	}
	if format == StarFormatASC {
		val := s[fst:masStr]
		str += string(val)
	} else {
		val := s[:digit]
		str = string(val) + str
	}
	return str
}
