package util

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
)

// Encryption process:
//  1. Process the data, pad the data using PKCS7 (when the key length is insufficient, pad with the number of missing bytes).
//  2. Encrypt the data using AES encryption in CBC mode.
//  3. Encode the encrypted data using base64 to get a string.
// Decryption process is the reverse.

// pkcs7Padding Padding
func pkcs7Padding(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}

// pkcs7UnPadding Unpad the padding
func pkcs7UnPadding(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("encryption string error: empty data")
	}
	unPadding := int(data[length-1])
	if unPadding < 1 || unPadding > length {
		return nil, errors.New("encryption string error: invalid padding")
	}
	return data[:(length - unPadding)], nil
}

// generateRandomIV Generates a random IV
func generateRandomIV(blockSize int) ([]byte, error) {
	iv := make([]byte, blockSize)
	_, err := rand.Read(iv)
	if err != nil {
		return nil, err
	}
	return iv, nil
}

// AesEncrypt Encrypt
func AesEncrypt(data []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	iv, err := generateRandomIV(blockSize)
	if err != nil {
		return nil, err
	}
	encryptBytes := pkcs7Padding(data, blockSize)
	crypted := make([]byte, len(encryptBytes)+blockSize)
	copy(crypted, iv)
	blockMode := cipher.NewCBCEncrypter(block, iv)
	blockMode.CryptBlocks(crypted[blockSize:], encryptBytes)
	return crypted, nil
}

// AesDecrypt Decrypt
func AesDecrypt(data []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	if len(data) < blockSize {
		return nil, errors.New("data too short")
	}
	iv := data[:blockSize]
	data = data[blockSize:]
	blockMode := cipher.NewCBCDecrypter(block, iv)
	crypted := make([]byte, len(data))
	blockMode.CryptBlocks(crypted, data)
	crypted, err = pkcs7UnPadding(crypted)
	if err != nil {
		return nil, err
	}
	return crypted, nil
}

// GenerateKey Generates a secure key using SHA-256
func GenerateKey(pwdKey []byte) []byte {
	hash := sha256.Sum256(pwdKey)
	return hash[:]
}

// EncryptByAes Encrypt using AES and then base64 encode
func EncryptByAes(data, pwdKey string) (string, error) {
	key := GenerateKey([]byte(pwdKey))
	res, err := AesEncrypt([]byte(data), key)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(res), nil
}

// DecryptByAes Decrypt using AES
func DecryptByAes(data, pwdKey string) (string, error) {
	key := GenerateKey([]byte(pwdKey))
	dataByte, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", err
	}
	byteVal, byteErr := AesDecrypt(dataByte, key)
	if byteErr != nil {
		return "", byteErr
	}
	return string(byteVal), nil
}
