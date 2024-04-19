package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	json "github.com/json-iterator/go"
	"github.com/shirou/gopsutil/v3/host"
	"strings"
)

// GenerateUserCode 通过mac地址，主机信息生成用户唯一不可逆标识码
func GenerateUserCode() string {
	hash := md5.Sum([]byte(GetHostInfoString()))
	return hex.EncodeToString(hash[:])
}

// GenerateLicenseKey 使用加密工具和用户标识码并添加额外信息生成license
func GenerateLicenseKey(userCode string, data any) string {
	licenseBytes, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	iv := make([]byte, cipherAead.NonceSize())
	copy(iv[:], userCode)
	cipherBytes := cipherAead.Seal(nil, iv, licenseBytes, []byte(`😄`))
	iv = append(iv, cipherBytes...)
	return base64.StdEncoding.EncodeToString(iv)
}

// DecodeLicenseKey 解密license，并与用户唯一标识码比对
func DecodeLicenseKey(encodedCode string) []byte {
	nonceSize := cipherAead.NonceSize()
	decodeBytes, err := base64.StdEncoding.DecodeString(encodedCode)
	if err != nil || len(decodeBytes) < nonceSize {
		return nil
	}

	iv := decodeBytes[:nonceSize]
	cipherBytes := decodeBytes[nonceSize:]
	licenseBytes, err := cipherAead.Open(nil, iv, cipherBytes, []byte(`😄`))
	if err != nil {
		return nil
	}

	ivExpected := make([]byte, cipherAead.NonceSize())
	copy(ivExpected[:], GenerateUserCode())
	if !bytes.Equal(iv, ivExpected) {
		return nil
	}

	return licenseBytes
}

// GetHostInfoString 获取主机信息并将其中变化参数设置为常量
// 通过修改hostId实现对之前版本的兼容
func GetHostInfoString() string {
	hostInfo, err := host.Info()
	if err != nil {
		panic(err)
	}
	hostInfo.BootTime = 20230308
	hostInfo.Uptime = 20230520
	hostInfo.Procs = 1314
	return strings.ReplaceAll(hostInfo.String(), `"hostId":`, `"hostid":`)
}

var cipherAead cipher.AEAD

func init() {
	keyBytes := make([]byte, 32)
	copy(keyBytes[:], "😄202305201314😄")
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		panic(err)
	}

	cipherAead, err = cipher.NewGCM(block)
	if err != nil {
		panic(err)
	}
}
