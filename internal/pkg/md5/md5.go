package md5

import (
	"crypto/md5"
	"fmt"
)

// md5str 返回字符串的 MD5 十六进制摘要，用于缓存 key
func md5str(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}
