package file

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	fileBinHeaderV1Size = 1 + 1 + 2 + 16 + 8

	fileBinVerV1   = 1
	fileBinFlagFIN = 1 << 0
)

type fileBinHeaderV1 struct {
	Ver       uint8
	Flags     uint8
	Reserved  uint16
	SessionID [16]byte
	Offset    uint64
}

var (
	errFileInvalidName = errors.New("invalid name")
	errFileInvalidDir  = errors.New("invalid dir")
)

func fileNewUUID() ([16]byte, error) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return [16]byte{}, err
	}
	id[6] = (id[6] & 0x0f) | 0x40
	id[8] = (id[8] & 0x3f) | 0x80
	return id, nil
}

func fileUUIDToString(id [16]byte) string {
	var buf [36]byte
	hex.Encode(buf[0:8], id[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], id[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], id[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], id[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:36], id[10:16])
	return string(buf[:])
}

func fileParseUUID(s string) ([16]byte, bool) {
	var id [16]byte
	s = strings.TrimSpace(strings.ToLower(s))
	if len(s) != 36 {
		return [16]byte{}, false
	}
	if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return [16]byte{}, false
	}
	raw := strings.ReplaceAll(s, "-", "")
	if len(raw) != 32 {
		return [16]byte{}, false
	}
	b, err := hex.DecodeString(raw)
	if err != nil || len(b) != 16 {
		return [16]byte{}, false
	}
	copy(id[:], b)
	return id, true
}

func fileDecodeBinHeaderV1(payload []byte) (kind byte, hdr fileBinHeaderV1, body []byte, ok bool) {
	if len(payload) < 1 {
		return 0, fileBinHeaderV1{}, nil, false
	}
	kind = payload[0]
	if len(payload) < 1+fileBinHeaderV1Size {
		return kind, fileBinHeaderV1{}, nil, false
	}
	i := 1
	hdr.Ver = payload[i]
	hdr.Flags = payload[i+1]
	hdr.Reserved = binary.BigEndian.Uint16(payload[i+2 : i+4])
	copy(hdr.SessionID[:], payload[i+4:i+20])
	hdr.Offset = binary.BigEndian.Uint64(payload[i+20 : i+28])
	body = payload[i+28:]
	return kind, hdr, body, true
}

func fileEncodeBinHeaderV1(kind byte, sessionID [16]byte, offset uint64, fin bool, body []byte) []byte {
	outLen := 1 + fileBinHeaderV1Size
	if body != nil && len(body) > 0 {
		outLen += len(body)
	}
	out := make([]byte, outLen)
	out[0] = kind
	out[1] = fileBinVerV1
	flags := uint8(0)
	if fin {
		flags |= fileBinFlagFIN
	}
	out[2] = flags
	copy(out[5:21], sessionID[:])
	binary.BigEndian.PutUint64(out[21:29], offset)
	if len(body) > 0 {
		copy(out[29:], body)
	}
	return out
}

func fileSanitizeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return "", errFileInvalidName
	}
	if strings.ContainsAny(name, "/\\") {
		return "", errFileInvalidName
	}
	if strings.ContainsRune(name, 0) {
		return "", errFileInvalidName
	}
	return name, nil
}

func fileSanitizeDir(dir string) (string, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" || dir == "." {
		return "", nil
	}
	if strings.ContainsRune(dir, 0) {
		return "", errFileInvalidDir
	}
	if strings.HasPrefix(dir, "/") || strings.HasPrefix(dir, "\\") {
		return "", errFileInvalidDir
	}
	if len(dir) >= 2 {
		if ((dir[0] >= 'a' && dir[0] <= 'z') || (dir[0] >= 'A' && dir[0] <= 'Z')) && dir[1] == ':' {
			return "", errFileInvalidDir
		}
	}
	if strings.Contains(dir, "\\") {
		return "", errFileInvalidDir
	}
	clean := path.Clean(dir)
	if clean == "." {
		return "", nil
	}
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", errFileInvalidDir
	}
	return clean, nil
}

func fileResolvePaths(baseDir, dir, name string) (finalPath, partPath string, err error) {
	name, err = fileSanitizeName(name)
	if err != nil {
		return "", "", err
	}
	dir, err = fileSanitizeDir(dir)
	if err != nil {
		return "", "", err
	}
	baseDir = strings.TrimSpace(baseDir)
	if baseDir == "" {
		baseDir = "."
	}
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", "", err
	}
	absFinal, err := filepath.Abs(filepath.Join(absBase, filepath.FromSlash(dir), name))
	if err != nil {
		return "", "", err
	}
	rel, err := filepath.Rel(absBase, absFinal)
	if err != nil {
		return "", "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", "", errFileInvalidDir
	}
	return absFinal, absFinal + ".part", nil
}

func parseUint32List(raw string) []uint32 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var list []uint32
	if err := json.Unmarshal([]byte(raw), &list); err == nil {
		return list
	}
	var asStrings []string
	if err := json.Unmarshal([]byte(raw), &asStrings); err == nil {
		out := make([]uint32, 0, len(asStrings))
		for _, item := range asStrings {
			if n, err := strconv.ParseUint(strings.TrimSpace(item), 10, 32); err == nil {
				out = append(out, uint32(n))
			}
		}
		return out
	}
	return nil
}

func encodeUint32List(list []uint32) string {
	raw, _ := json.Marshal(list)
	return string(raw)
}
