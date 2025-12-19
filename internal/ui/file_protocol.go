package ui

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
	"time"
)

// file 子协议（SubProto=5），用于节点间文件传输。
const (
	subProtoFile uint8 = 5
)

// payload[0]：帧类型。
const (
	fileKindCtrl byte = 0x01
	fileKindData byte = 0x02
	fileKindAck  byte = 0x03
)

const (
	fileActionRead      = "read"
	fileActionWrite     = "write"
	fileActionReadResp  = "read_resp"
	fileActionWriteResp = "write_resp"
)

const (
	fileOpPull  = "pull"
	fileOpOffer = "offer"
	fileOpList  = "list"
	fileOpReadText = "read_text"
)

type fileMessage struct {
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
}

type fileReadReq struct {
	Op         string `json:"op"`
	Target     uint32 `json:"target,omitempty"`
	Dir        string `json:"dir,omitempty"`
	Name       string `json:"name,omitempty"`
	Overwrite  *bool  `json:"overwrite,omitempty"`
	ResumeFrom uint64 `json:"resume_from,omitempty"`
	WantHash   *bool  `json:"want_hash,omitempty"`
	Recursive  bool   `json:"recursive,omitempty"`
	MaxBytes   uint32 `json:"max_bytes,omitempty"`
}

type fileReadResp struct {
	Code      int      `json:"code"`
	Msg       string   `json:"msg,omitempty"`
	Op        string   `json:"op,omitempty"`
	SessionID string   `json:"session_id,omitempty"`
	Provider  uint32   `json:"provider,omitempty"`
	Consumer  uint32   `json:"consumer,omitempty"`
	Dir       string   `json:"dir,omitempty"`
	Name      string   `json:"name,omitempty"`
	Size      uint64   `json:"size,omitempty"`
	Sha256    string   `json:"sha256,omitempty"`
	StartFrom uint64   `json:"start_from,omitempty"`
	Chunk     uint32   `json:"chunk_bytes,omitempty"`
	Dirs      []string `json:"dirs,omitempty"`
	Files     []string `json:"files,omitempty"`
	Text      string   `json:"text,omitempty"`
	Truncated bool     `json:"truncated,omitempty"`
}

type fileWriteReq struct {
	Op        string `json:"op"`
	Target    uint32 `json:"target"`
	SessionID string `json:"session_id"`
	Dir       string `json:"dir,omitempty"`
	Name      string `json:"name"`
	Size      uint64 `json:"size"`
	Sha256    string `json:"sha256,omitempty"`
	Overwrite *bool  `json:"overwrite,omitempty"`
}

type fileWriteResp struct {
	Code       int    `json:"code"`
	Msg        string `json:"msg,omitempty"`
	Op         string `json:"op,omitempty"`
	SessionID  string `json:"session_id,omitempty"`
	Provider   uint32 `json:"provider,omitempty"`
	Consumer   uint32 `json:"consumer,omitempty"`
	Dir        string `json:"dir,omitempty"`
	Name       string `json:"name,omitempty"`
	Size       uint64 `json:"size,omitempty"`
	Sha256     string `json:"sha256,omitempty"`
	Accept     bool   `json:"accept,omitempty"`
	ResumeFrom uint64 `json:"resume_from,omitempty"`
}

func fileMustJSON(v any) json.RawMessage {
	raw, _ := json.Marshal(v)
	return raw
}

// --- UUID helpers (v4) ---

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

// --- binary header ---

const fileBinHeaderV1Size = 1 + 1 + 2 + 16 + 8

const (
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

func fileDecodeBinHeaderV1(payload []byte) (kind byte, hdr fileBinHeaderV1, body []byte, ok bool) {
	if len(payload) < 1 {
		return 0, fileBinHeaderV1{}, nil, false
	}
	kind = payload[0]
	if kind != fileKindData && kind != fileKindAck {
		return kind, fileBinHeaderV1{}, nil, false
	}
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
	if kind != fileKindData && kind != fileKindAck {
		return nil
	}
	outLen := 1 + fileBinHeaderV1Size
	if kind == fileKindData {
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
	if kind == fileKindData && len(body) > 0 {
		copy(out[29:], body)
	}
	return out
}

// --- name/dir safety ---

var (
	errFileInvalidName = errors.New("invalid name")
	errFileInvalidDir  = errors.New("invalid dir")
)

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

// --- file config (read from Win preferences config map) ---

const (
	cfgFileBaseDir          = "file.base_dir"
	cfgFileMaxSizeBytes     = "file.max_size_bytes"
	cfgFileMaxConcurrent    = "file.max_concurrent"
	cfgFileChunkBytes       = "file.chunk_bytes"
	cfgFileIncompleteTTLSec = "file.incomplete_ttl_sec"
	cfgFileWantSHA256       = "file.want_sha256"
)

type fileConfig struct {
	BaseDir       string
	MaxSizeBytes  uint64
	MaxConcurrent int
	ChunkBytes    int
	WantSHA256    bool
	IncompleteTTL time.Duration

	AckEveryBytes uint64
	AckEvery      time.Duration
}

func (c *Controller) fileConfig() fileConfig {
	cfg := c.currentConfig(true)
	baseDir := strings.TrimSpace(cfg[cfgFileBaseDir])
	if baseDir == "" {
		baseDir = "./file"
	}
	maxSizeBytes := fileReadUint64(cfg, cfgFileMaxSizeBytes, 0)
	maxConcurrent := fileReadInt(cfg, cfgFileMaxConcurrent, 4)
	chunkBytes := fileReadInt(cfg, cfgFileChunkBytes, 256*1024)
	incompleteTTL := time.Duration(fileReadInt64(cfg, cfgFileIncompleteTTLSec, 3600)) * time.Second
	wantSHA256 := fileReadBool(cfg, cfgFileWantSHA256, true)
	return fileConfig{
		BaseDir:       baseDir,
		MaxSizeBytes:  maxSizeBytes,
		MaxConcurrent: maxConcurrent,
		ChunkBytes:    chunkBytes,
		WantSHA256:    wantSHA256,
		IncompleteTTL: incompleteTTL,

		AckEveryBytes: 512 * 1024,
		AckEvery:      500 * time.Millisecond,
	}
}

func fileReadInt(cfg map[string]string, key string, def int) int {
	raw := strings.TrimSpace(cfg[key])
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

func fileReadInt64(cfg map[string]string, key string, def int64) int64 {
	raw := strings.TrimSpace(cfg[key])
	if raw == "" {
		return def
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

func fileReadUint64(cfg map[string]string, key string, def uint64) uint64 {
	raw := strings.TrimSpace(cfg[key])
	if raw == "" {
		return def
	}
	n, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return def
	}
	return n
}

func fileReadBool(cfg map[string]string, key string, def bool) bool {
	raw := strings.TrimSpace(cfg[key])
	if raw == "" {
		return def
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return def
	}
	return v
}
