package util

import (
	"crypto/md5"
)

func CryptoByMd5(data []byte, saltVal []byte) []byte {
	md5Ctx := md5.New()
	md5Ctx.Write(data)
	if saltVal != nil {
		md5Ctx.Write(saltVal)
	}
	return md5Ctx.Sum(nil)
}
