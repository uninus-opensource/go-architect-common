package math

import (
	"crypto/md5"
	"fmt"
	"strconv"
)

// Fungsi untuk menghasilkan hash MD5 dari nilai salt (salt ex: increment orderid) dan nilai unik tertentu (unique value ex: userid)
func GenerateHash(uniqueValue string, salt int) string {
	data := []byte(strconv.Itoa(salt) + uniqueValue)
	hash := md5.Sum(data)
	return fmt.Sprintf("%x", hash)
}

// Fungsi untuk menghasilkan 13 digit alfanumerik dari hash yang dihasilkan
func Generate13DigitAlphaNumeric(hash string) string {
	// Ambil 13 karakter pertama dari hash sebagai 13 digit alfanumerik
	return hash[:13]
}
