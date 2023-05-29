// Copyright 2021 Francisco Souza. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package checksum

import (
	"crypto/md5"
	"encoding/base64"
	"hash/crc32"
)

var crc32cTable = crc32.MakeTable(crc32.Castagnoli)

func crc32cChecksum(content []byte) []byte {
	checksummer := crc32.New(crc32cTable)
	checksummer.Write(content)
	return checksummer.Sum(make([]byte, 0, 4))
}

func EncodedChecksum(checksum []byte) string {
	return base64.StdEncoding.EncodeToString(checksum)
}

func EncodedCrc32cChecksum(content []byte) string {
	return EncodedChecksum(crc32cChecksum(content))
}

func MD5Hash(b []byte) []byte {
	h := md5.New()
	h.Write(b)
	return h.Sum(nil)
}

func EncodedHash(hash []byte) string {
	return base64.StdEncoding.EncodeToString(hash)
}

func EncodedMd5Hash(content []byte) string {
	return EncodedHash(MD5Hash(content))
}
