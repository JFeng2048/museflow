package token

import (
	"crypto/sha256"
	"encoding/hex"
)

// DeviceFingerprint 计算设备指纹 sha256(deviceId + userAgent + ip)。
func DeviceFingerprint(deviceID, userAgent, ip string) string {
	sum := sha256.Sum256([]byte(deviceID + userAgent + ip))
	return hex.EncodeToString(sum[:])
}
