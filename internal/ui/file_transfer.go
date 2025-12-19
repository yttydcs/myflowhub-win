package ui

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	core "github.com/yttydcs/myflowhub-core"
	"github.com/yttydcs/myflowhub-core/header"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type fileState struct {
	mu sync.Mutex

	recv        map[[16]byte]*fileRecvSession
	send        map[[16]byte]*fileSendSession
	pendingPull map[filePullKey]*fileTask
	tasks       []*fileTask
	taskBySid   map[[16]byte]*fileTask

	list   *widget.List
	lastUI time.Time

	janitorStarted bool
	janitorCancel  context.CancelFunc
}

type filePullKey struct {
	provider uint32
	dir      string
	name     string
}

type fileTask struct {
	mu sync.Mutex

	taskID [16]byte

	sessionID    [16]byte
	hasSessionID bool

	createdAt time.Time
	updatedAt time.Time

	op        string // pull/offer
	direction string // download/upload

	provider uint32
	consumer uint32
	peer     uint32

	dir      string
	name     string
	size     uint64
	sha256   string
	wantHash bool

	localDir  string
	localName string
	localPath string
	filePath  string

	status    string
	lastError string

	sentBytes  uint64
	ackedBytes uint64
	doneBytes  uint64
}

type fileRecvSession struct {
	id       [16]byte
	provider uint32
	consumer uint32

	dir       string
	name      string
	finalPath string
	partPath  string

	size      uint64
	sha256Hex string
	overwrite bool

	mu              sync.Mutex
	file            *os.File
	hasher          hash.Hash
	expectedOffset  uint64
	pending         map[uint64][]byte
	pendingBytes    uint64
	maxPendingBytes uint64
	finSeen         bool
	lastActive      time.Time
	lastAckOffset   uint64
	lastAckTime     time.Time

	task *fileTask
}

type fileSendSession struct {
	id       [16]byte
	provider uint32
	consumer uint32

	dir      string
	name     string
	filePath string

	size      uint64
	sha256Hex string
	startFrom uint64

	mu         sync.Mutex
	ackedUntil uint64
	lastActive time.Time
	sentEOF    bool
	cancel     context.CancelFunc

	task *fileTask
}

func newFileState() *fileState {
	return &fileState{
		recv:        make(map[[16]byte]*fileRecvSession),
		send:        make(map[[16]byte]*fileSendSession),
		pendingPull: make(map[filePullKey]*fileTask),
		taskBySid:   make(map[[16]byte]*fileTask),
	}
}

func (c *Controller) fileStartJanitor() {
	if c == nil || c.ctx == nil || c.file == nil {
		return
	}
	c.file.mu.Lock()
	if c.file.janitorStarted {
		c.file.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(c.ctx)
	c.file.janitorStarted = true
	c.file.janitorCancel = cancel
	c.file.mu.Unlock()

	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.fileJanitorOnce()
			}
		}
	}()
}

func (c *Controller) fileJanitorOnce() {
	cfg := c.fileConfig()
	if cfg.IncompleteTTL <= 0 {
		return
	}
	baseAbs, err := filepath.Abs(cfg.BaseDir)
	if err != nil {
		return
	}
	now := time.Now()
	active := make(map[string]struct{})
	if c.file != nil {
		c.file.mu.Lock()
		for _, s := range c.file.recv {
			if s != nil && strings.TrimSpace(s.partPath) != "" {
				active[s.partPath] = struct{}{}
			}
		}
		c.file.mu.Unlock()
	}
	_ = filepath.WalkDir(baseAbs, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d == nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".part") {
			return nil
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return nil
		}
		if _, ok := active[abs]; ok {
			return nil
		}
		info, err := d.Info()
		if err != nil || info == nil {
			return nil
		}
		if now.Sub(info.ModTime()) <= cfg.IncompleteTTL {
			return nil
		}
		_ = os.Remove(abs)
		return nil
	})
}

func (c *Controller) handleFileFrame(h core.IHeader, payload []byte) {
	if c == nil || h == nil || len(payload) == 0 {
		return
	}
	switch payload[0] {
	case fileKindCtrl:
		c.fileHandleCtrl(h, payload)
		c.fileStartJanitor()
	case fileKindData:
		c.fileHandleData(h, payload)
	case fileKindAck:
		c.fileHandleAck(h, payload)
	default:
		return
	}
}

func (c *Controller) fileHandleCtrl(h core.IHeader, payload []byte) {
	if len(payload) < 2 {
		return
	}
	var msg fileMessage
	if err := json.Unmarshal(payload[1:], &msg); err != nil {
		return
	}
	action := strings.ToLower(strings.TrimSpace(msg.Action))
	switch action {
	case fileActionRead:
		var req fileReadReq
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			c.fileSendReadResp(h.SourceID(), fileReadResp{Code: 400, Msg: "invalid read", Op: strings.TrimSpace(req.Op)})
			return
		}
		c.fileHandleReadRequest(h, req)
	case fileActionWrite:
		var req fileWriteReq
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			c.fileSendWriteResp(h.SourceID(), fileWriteResp{Code: 400, Msg: "invalid write", Op: strings.TrimSpace(req.Op)})
			return
		}
		c.fileHandleWriteRequest(h, req)
	case fileActionReadResp:
		c.fileHandleReadResp(h, msg.Data)
	case fileActionWriteResp:
		c.fileHandleWriteResp(h, msg.Data)
	default:
		return
	}
}

func (c *Controller) fileHandleReadRequest(h core.IHeader, req fileReadReq) {
	op := strings.ToLower(strings.TrimSpace(req.Op))
	if op != fileOpPull && op != fileOpList && op != fileOpReadText {
		c.fileSendReadResp(h.SourceID(), fileReadResp{Code: 400, Msg: "invalid op", Op: op})
		return
	}
	localNode := c.storedNode
	if localNode == 0 {
		return
	}
	requester := h.SourceID()

	target := req.Target
	if op == fileOpList && target == 0 {
		target = localNode
	}
	if target == 0 {
		c.fileSendReadResp(requester, fileReadResp{Code: 400, Msg: "target required", Op: op})
		return
	}
	if target != localNode {
		// Win 端作为普通节点，不做 Hub 转交。
		return
	}

	switch op {
	case fileOpPull:
		c.fileHandlePullAsProvider(requester, req)
	case fileOpList:
		c.fileHandleListLocal(requester, req)
	case fileOpReadText:
		c.fileHandleReadTextLocal(requester, req)
	}
}

func (c *Controller) fileHandleWriteRequest(h core.IHeader, req fileWriteReq) {
	op := strings.ToLower(strings.TrimSpace(req.Op))
	if op != fileOpOffer {
		c.fileSendWriteResp(h.SourceID(), fileWriteResp{Code: 400, Msg: "invalid op", Op: op})
		return
	}
	localNode := c.storedNode
	if localNode == 0 {
		return
	}
	provider := h.SourceID()
	if req.Target == 0 || req.Target != localNode {
		c.fileSendWriteResp(provider, fileWriteResp{Code: 400, Msg: "target mismatch", Op: op, SessionID: strings.TrimSpace(req.SessionID)})
		return
	}
	sid, ok := fileParseUUID(req.SessionID)
	if !ok {
		c.fileSendWriteResp(provider, fileWriteResp{Code: 400, Msg: "invalid session", Op: op, SessionID: strings.TrimSpace(req.SessionID), Accept: false})
		return
	}
	cfg := c.fileConfig()
	if cfg.MaxSizeBytes > 0 && req.Size > cfg.MaxSizeBytes {
		c.fileSendWriteResp(provider, fileWriteResp{Code: 413, Msg: "too large", Op: op, SessionID: strings.TrimSpace(req.SessionID), Accept: false})
		return
	}
	if c.fileTotalSessions() >= cfg.MaxConcurrent {
		c.fileSendWriteResp(provider, fileWriteResp{Code: 429, Msg: "too many sessions", Op: op, SessionID: strings.TrimSpace(req.SessionID), Accept: false})
		return
	}
	dir := strings.ReplaceAll(strings.TrimSpace(req.Dir), "\\", "/")
	name := strings.TrimSpace(req.Name)
	if _, err := fileSanitizeDir(dir); err != nil {
		c.fileSendWriteResp(provider, fileWriteResp{Code: 400, Msg: "invalid dir", Op: op, SessionID: strings.TrimSpace(req.SessionID), Accept: false})
		return
	}
	if _, err := fileSanitizeName(name); err != nil {
		c.fileSendWriteResp(provider, fileWriteResp{Code: 400, Msg: "invalid name", Op: op, SessionID: strings.TrimSpace(req.SessionID), Accept: false})
		return
	}
	// 如果已存在会话，避免重复弹窗。
	if c.fileGetRecvSession(sid) != nil {
		return
	}
	c.filePromptIncomingOffer(h, req)
}

func (c *Controller) fileHandlePullAsProvider(consumer uint32, req fileReadReq) {
	cfg := c.fileConfig()
	provider := c.storedNode
	if provider == 0 {
		return
	}
	dir := strings.ReplaceAll(strings.TrimSpace(req.Dir), "\\", "/")
	name := strings.TrimSpace(req.Name)
	finalPath, _, err := fileResolvePaths(cfg.BaseDir, dir, name)
	if err != nil {
		c.fileSendReadResp(consumer, fileReadResp{Code: 400, Msg: "invalid path", Op: fileOpPull, Provider: provider, Consumer: consumer, Dir: dir, Name: name})
		return
	}
	info, err := os.Stat(finalPath)
	if err != nil || info == nil || info.IsDir() {
		c.fileSendReadResp(consumer, fileReadResp{Code: 404, Msg: "not found", Op: fileOpPull, Provider: provider, Consumer: consumer, Dir: dir, Name: name})
		return
	}
	size := uint64(info.Size())
	if cfg.MaxSizeBytes > 0 && size > cfg.MaxSizeBytes {
		c.fileSendReadResp(consumer, fileReadResp{Code: 413, Msg: "too large", Op: fileOpPull, Provider: provider, Consumer: consumer, Dir: dir, Name: name, Size: size})
		return
	}
	if c.fileTotalSessions() >= cfg.MaxConcurrent {
		c.fileSendReadResp(consumer, fileReadResp{Code: 429, Msg: "too many sessions", Op: fileOpPull, Provider: provider, Consumer: consumer, Dir: dir, Name: name, Size: size})
		return
	}
	wantHash := cfg.WantSHA256
	if req.WantHash != nil {
		wantHash = *req.WantHash
	}
	shaHex := ""
	if wantHash {
		if sh, err := fileHashSHA256(finalPath); err == nil {
			shaHex = sh
		}
	}
	startFrom := uint64(0)
	if req.ResumeFrom > 0 && req.ResumeFrom < size {
		startFrom = req.ResumeFrom
	}
	sid, err := fileNewUUID()
	if err != nil {
		c.fileSendReadResp(consumer, fileReadResp{Code: 500, Msg: "uuid failed", Op: fileOpPull})
		return
	}
	taskID, _ := fileNewUUID()
	task := &fileTask{
		taskID:       taskID,
		sessionID:    sid,
		hasSessionID: true,
		createdAt:    time.Now(),
		updatedAt:    time.Now(),
		op:           fileOpPull,
		direction:    "upload",
		provider:     provider,
		consumer:     consumer,
		peer:         consumer,
		dir:          dir,
		name:         name,
		size:         size,
		sha256:       shaHex,
		wantHash:     wantHash,
		filePath:     finalPath,
		status:       "发送中",
	}
	sess := &fileSendSession{
		id:         sid,
		provider:   provider,
		consumer:   consumer,
		dir:        dir,
		name:       name,
		filePath:   finalPath,
		size:       size,
		sha256Hex:  shaHex,
		startFrom:  startFrom,
		lastActive: time.Now(),
		task:       task,
	}
	c.fileAddTask(task)
	c.fileAddSendSession(sess)

	c.fileSendReadResp(consumer, fileReadResp{
		Code:      1,
		Msg:       "ok",
		Op:        fileOpPull,
		SessionID: fileUUIDToString(sid),
		Provider:  provider,
		Consumer:  consumer,
		Dir:       dir,
		Name:      name,
		Size:      size,
		Sha256:    shaHex,
		StartFrom: startFrom,
		Chunk:     uint32(cfg.ChunkBytes),
	})
	go c.fileSendData(sid)
}

func (c *Controller) fileHandleListLocal(requester uint32, req fileReadReq) {
	dir := strings.ReplaceAll(strings.TrimSpace(req.Dir), "\\", "/")
	dir, err := fileSanitizeDir(dir)
	if err != nil {
		c.fileSendReadResp(requester, fileReadResp{Code: 400, Msg: "invalid dir", Op: fileOpList, Dir: dir})
		return
	}
	cfg := c.fileConfig()
	baseAbs, err := filepath.Abs(cfg.BaseDir)
	if err != nil {
		c.fileSendReadResp(requester, fileReadResp{Code: 500, Msg: "invalid base", Op: fileOpList, Dir: dir})
		return
	}
	root := filepath.Join(baseAbs, filepath.FromSlash(dir))
	entries, err := os.ReadDir(root)
	if err != nil {
		c.fileSendReadResp(requester, fileReadResp{Code: 404, Msg: "not found", Op: fileOpList, Dir: dir, Files: []string{}, Dirs: []string{}})
		return
	}
	dirs := make([]string, 0, len(entries))
	files := make([]string, 0, len(entries))
	for _, e := range entries {
		if e == nil || e.IsDir() {
			if e != nil && e.IsDir() {
				dirs = append(dirs, e.Name())
			}
			continue
		}
		files = append(files, e.Name())
	}
	sort.Strings(dirs)
	sort.Strings(files)
	c.fileSendReadResp(requester, fileReadResp{Code: 1, Msg: "ok", Op: fileOpList, Dir: dir, Files: files, Dirs: dirs})
}

func (c *Controller) fileHandleReadTextLocal(requester uint32, req fileReadReq) {
	cfg := c.fileConfig()
	dir := strings.ReplaceAll(strings.TrimSpace(req.Dir), "\\", "/")
	name := strings.TrimSpace(req.Name)
	finalPath, _, err := fileResolvePaths(cfg.BaseDir, dir, name)
	if err != nil {
		c.fileSendReadResp(requester, fileReadResp{Code: 400, Msg: "invalid path", Op: fileOpReadText, Dir: dir, Name: name})
		return
	}
	info, err := os.Stat(finalPath)
	if err != nil || info == nil || info.IsDir() {
		c.fileSendReadResp(requester, fileReadResp{Code: 404, Msg: "not found", Op: fileOpReadText, Dir: dir, Name: name})
		return
	}
	size := uint64(info.Size())
	maxBytes := int(req.MaxBytes)
	if maxBytes <= 0 {
		maxBytes = 64 * 1024
	}
	if maxBytes > 256*1024 {
		maxBytes = 256 * 1024
	}
	f, err := os.Open(finalPath)
	if err != nil {
		c.fileSendReadResp(requester, fileReadResp{Code: 500, Msg: "open failed", Op: fileOpReadText, Dir: dir, Name: name})
		return
	}
	defer func() { _ = f.Close() }()
	buf := make([]byte, maxBytes)
	n, rerr := io.ReadFull(f, buf)
	if rerr == io.ErrUnexpectedEOF || rerr == io.EOF {
		// ok
	} else if rerr != nil {
		c.fileSendReadResp(requester, fileReadResp{Code: 500, Msg: "read failed", Op: fileOpReadText, Dir: dir, Name: name})
		return
	}
	buf = buf[:n]
	truncated := uint64(n) < size
	if !utf8.Valid(buf) {
		c.fileSendReadResp(requester, fileReadResp{Code: 415, Msg: "not text", Op: fileOpReadText, Dir: dir, Name: name, Size: size})
		return
	}
	c.fileSendReadResp(requester, fileReadResp{Code: 1, Msg: "ok", Op: fileOpReadText, Dir: dir, Name: name, Size: size, Text: string(buf), Truncated: truncated})
}

func (c *Controller) fileHandleReadResp(h core.IHeader, data json.RawMessage) {
	if c == nil || h == nil {
		return
	}
	var resp fileReadResp
	if err := json.Unmarshal(data, &resp); err != nil {
		return
	}
	op := strings.ToLower(strings.TrimSpace(resp.Op))
	if op == fileOpList {
		if c.fileBrowser != nil {
			c.fileBrowserHandleListResp(h.SourceID(), resp)
		}
		return
	}
	if op == fileOpReadText {
		if c.fileBrowser != nil {
			c.fileBrowserHandleReadTextResp(h.SourceID(), resp)
		}
		return
	}
	if op != fileOpPull {
		return
	}
	provider := h.SourceID()
	consumer := c.storedNode
	if consumer == 0 {
		return
	}
	if resp.Consumer != 0 && resp.Consumer != consumer {
		return
	}
	if resp.Provider != 0 && resp.Provider != provider {
		return
	}
	if resp.Code != 1 {
		c.fileFailPendingPull(provider, resp.Dir, resp.Name, fmt.Sprintf("%d %s", resp.Code, strings.TrimSpace(resp.Msg)))
		return
	}
	if strings.TrimSpace(resp.SessionID) == "" {
		return
	}
	sid, ok := fileParseUUID(resp.SessionID)
	if !ok {
		return
	}
	dir := strings.ReplaceAll(strings.TrimSpace(resp.Dir), "\\", "/")
	name := strings.TrimSpace(resp.Name)
	task := c.fileTakePendingPull(provider, dir, name)
	if task == nil {
		taskID, _ := fileNewUUID()
		task = &fileTask{
			taskID:    taskID,
			createdAt: time.Now(),
			updatedAt: time.Now(),
			op:        fileOpPull,
			direction: "download",
			provider:  provider,
			consumer:  consumer,
			peer:      provider,
			dir:       dir,
			name:      name,
			wantHash:  c.fileConfig().WantSHA256,
			status:    "接收中",
		}
		c.fileAddTask(task)
	}

	cfg := c.fileConfig()
	task.mu.Lock()
	localDir := strings.ReplaceAll(strings.TrimSpace(task.localDir), "\\", "/")
	localName := strings.TrimSpace(task.localName)
	wantHash := task.wantHash
	task.mu.Unlock()
	if localDir == "" {
		localDir = dir
	}
	saveName := name
	if localName != "" {
		saveName = localName
	}
	finalPath, partPath, err := fileResolvePaths(cfg.BaseDir, localDir, saveName)
	if err != nil {
		task.mu.Lock()
		task.status = "失败"
		task.lastError = "invalid save path"
		task.updatedAt = time.Now()
		task.mu.Unlock()
		c.fileRefreshUIThrottled()
		return
	}
	if cfg.MaxSizeBytes > 0 && resp.Size > cfg.MaxSizeBytes {
		task.mu.Lock()
		task.status = "失败"
		task.lastError = "too large"
		task.updatedAt = time.Now()
		task.mu.Unlock()
		c.fileRefreshUIThrottled()
		return
	}
	if c.fileTotalSessions() >= cfg.MaxConcurrent {
		task.mu.Lock()
		task.status = "失败"
		task.lastError = "too many sessions"
		task.updatedAt = time.Now()
		task.mu.Unlock()
		c.fileRefreshUIThrottled()
		return
	}
	if err := os.MkdirAll(filepath.Dir(finalPath), 0o755); err != nil {
		task.mu.Lock()
		task.status = "失败"
		task.lastError = "mkdir failed"
		task.updatedAt = time.Now()
		task.mu.Unlock()
		c.fileRefreshUIThrottled()
		return
	}

	startFrom := resp.StartFrom
	if startFrom > resp.Size {
		startFrom = 0
	}
	f, err := os.OpenFile(partPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		task.mu.Lock()
		task.status = "失败"
		task.lastError = "open failed"
		task.updatedAt = time.Now()
		task.mu.Unlock()
		c.fileRefreshUIThrottled()
		return
	}
	if err := f.Truncate(int64(startFrom)); err != nil {
		_ = f.Close()
		task.mu.Lock()
		task.status = "失败"
		task.lastError = "truncate failed"
		task.updatedAt = time.Now()
		task.mu.Unlock()
		c.fileRefreshUIThrottled()
		return
	}
	shaHex := strings.TrimSpace(resp.Sha256)
	var hasher hash.Hash
	if wantHash && shaHex != "" {
		hasher = sha256.New()
	}
	if hasher != nil && startFrom > 0 {
		if _, err := f.Seek(0, io.SeekStart); err == nil {
			buf := make([]byte, 256*1024)
			remain := startFrom
			for remain > 0 {
				n := int64(len(buf))
				if remain < uint64(len(buf)) {
					n = int64(remain)
				}
				readN, rerr := io.ReadFull(f, buf[:n])
				if readN > 0 {
					_, _ = hasher.Write(buf[:readN])
					remain -= uint64(readN)
				}
				if rerr != nil {
					break
				}
			}
		}
	}
	if _, err := f.Seek(int64(startFrom), io.SeekStart); err != nil {
		_ = f.Close()
		task.mu.Lock()
		task.status = "失败"
		task.lastError = "seek failed"
		task.updatedAt = time.Now()
		task.mu.Unlock()
		c.fileRefreshUIThrottled()
		return
	}

	task.mu.Lock()
	task.sessionID = sid
	task.hasSessionID = true
	task.size = resp.Size
	task.sha256 = shaHex
	task.localDir = localDir
	task.localName = saveName
	task.localPath = finalPath
	task.status = "接收中"
	task.doneBytes = startFrom
	task.updatedAt = time.Now()
	task.mu.Unlock()

	c.file.mu.Lock()
	c.file.taskBySid[sid] = task
	c.file.mu.Unlock()

	sess := &fileRecvSession{
		id:              sid,
		provider:        provider,
		consumer:        consumer,
		dir:             dir,
		name:            name,
		finalPath:       finalPath,
		partPath:        partPath,
		size:            resp.Size,
		sha256Hex:       shaHex,
		overwrite:       true,
		file:            f,
		hasher:          hasher,
		expectedOffset:  startFrom,
		pending:         make(map[uint64][]byte),
		maxPendingBytes: uint64(cfg.ChunkBytes) * 8,
		lastActive:      time.Now(),
		task:            task,
	}
	c.fileAddRecvSession(sess)
	c.fileRefreshUIThrottled()
}

func (c *Controller) fileHandleWriteResp(_ core.IHeader, data json.RawMessage) {
	if c == nil {
		return
	}
	var resp fileWriteResp
	if err := json.Unmarshal(data, &resp); err != nil {
		return
	}
	if strings.ToLower(strings.TrimSpace(resp.Op)) != fileOpOffer {
		return
	}
	sid, ok := fileParseUUID(resp.SessionID)
	if !ok {
		return
	}
	sess := c.fileGetSendSession(sid)
	if sess == nil {
		return
	}

	if resp.Code != 1 || !resp.Accept {
		sess.mu.Lock()
		if sess.task != nil {
			sess.task.mu.Lock()
			sess.task.status = "失败"
			sess.task.lastError = strings.TrimSpace(resp.Msg)
			sess.task.updatedAt = time.Now()
			sess.task.mu.Unlock()
		}
		sess.mu.Unlock()
		c.fileRemoveSendSession(sid)
		c.fileRefreshUIThrottled()
		return
	}

	sess.mu.Lock()
	sess.startFrom = resp.ResumeFrom
	if sess.startFrom > sess.size {
		sess.startFrom = 0
	}
	task := sess.task
	if task != nil {
		task.mu.Lock()
		task.status = "发送中"
		task.updatedAt = time.Now()
		task.mu.Unlock()
	}
	already := sess.size != 0 && resp.ResumeFrom >= sess.size
	sess.mu.Unlock()
	c.fileRefreshUIThrottled()

	if already {
		sess.mu.Lock()
		sess.ackedUntil = sess.size
		sess.sentEOF = true
		if task != nil {
			task.mu.Lock()
			task.status = "完成"
			task.ackedBytes = sess.size
			task.updatedAt = time.Now()
			task.mu.Unlock()
		}
		sess.mu.Unlock()
		c.fileRemoveSendSession(sid)
		c.fileRefreshUIThrottled()
		return
	}
	go c.fileSendData(sid)
}

func (c *Controller) fileSendCtrl(target uint32, msg fileMessage) error {
	if c == nil || c.session == nil || target == 0 {
		return fmt.Errorf("invalid target")
	}
	body, _ := json.Marshal(msg)
	payload := make([]byte, 1+len(body))
	payload[0] = fileKindCtrl
	copy(payload[1:], body)
	hdr := (&header.HeaderTcp{}).
		WithMajor(header.MajorCmd).
		WithSubProto(subProtoFile).
		WithSourceID(c.storedNode).
		WithTargetID(target).
		WithMsgID(uint32(time.Now().UnixNano()))
	return c.session.Send(hdr, payload)
}

func (c *Controller) fileSendReadResp(target uint32, data fileReadResp) {
	data.Op = strings.ToLower(strings.TrimSpace(data.Op))
	_ = c.fileSendCtrl(target, fileMessage{Action: fileActionReadResp, Data: fileMustJSON(data)})
}

func (c *Controller) fileSendWriteResp(target uint32, data fileWriteResp) {
	data.Op = strings.ToLower(strings.TrimSpace(data.Op))
	_ = c.fileSendCtrl(target, fileMessage{Action: fileActionWriteResp, Data: fileMustJSON(data)})
}

func (c *Controller) fileTotalSessions() int {
	if c == nil || c.file == nil {
		return 0
	}
	c.file.mu.Lock()
	defer c.file.mu.Unlock()
	return len(c.file.recv) + len(c.file.send)
}

func (c *Controller) fileAddRecvSession(sess *fileRecvSession) {
	if c == nil || c.file == nil || sess == nil {
		return
	}
	c.file.mu.Lock()
	c.file.recv[sess.id] = sess
	c.file.mu.Unlock()
}

func (c *Controller) fileAddSendSession(sess *fileSendSession) {
	if c == nil || c.file == nil || sess == nil {
		return
	}
	c.file.mu.Lock()
	c.file.send[sess.id] = sess
	c.file.mu.Unlock()
}

func (c *Controller) fileGetRecvSession(id [16]byte) *fileRecvSession {
	if c == nil || c.file == nil {
		return nil
	}
	c.file.mu.Lock()
	defer c.file.mu.Unlock()
	return c.file.recv[id]
}

func (c *Controller) fileGetSendSession(id [16]byte) *fileSendSession {
	if c == nil || c.file == nil {
		return nil
	}
	c.file.mu.Lock()
	defer c.file.mu.Unlock()
	return c.file.send[id]
}

func (c *Controller) fileRemoveRecvSession(id [16]byte) {
	if c == nil || c.file == nil {
		return
	}
	c.file.mu.Lock()
	sess := c.file.recv[id]
	delete(c.file.recv, id)
	c.file.mu.Unlock()
	if sess != nil {
		sess.mu.Lock()
		if sess.file != nil {
			_ = sess.file.Close()
			sess.file = nil
		}
		sess.mu.Unlock()
	}
}

func (c *Controller) fileRemoveSendSession(id [16]byte) {
	if c == nil || c.file == nil {
		return
	}
	c.file.mu.Lock()
	sess := c.file.send[id]
	delete(c.file.send, id)
	c.file.mu.Unlock()
	if sess != nil {
		sess.mu.Lock()
		if sess.cancel != nil {
			sess.cancel()
			sess.cancel = nil
		}
		sess.mu.Unlock()
	}
}

func (c *Controller) fileAddTask(t *fileTask) {
	if c == nil || c.file == nil || t == nil {
		return
	}
	c.file.mu.Lock()
	c.file.tasks = append(c.file.tasks, t)
	if t.hasSessionID {
		c.file.taskBySid[t.sessionID] = t
	}
	c.file.mu.Unlock()
	c.fileRefreshUIThrottled()
}

func (c *Controller) fileTakePendingPull(provider uint32, dir, name string) *fileTask {
	if c == nil || c.file == nil {
		return nil
	}
	key := filePullKey{provider: provider, dir: strings.TrimSpace(dir), name: strings.TrimSpace(name)}
	c.file.mu.Lock()
	defer c.file.mu.Unlock()
	t := c.file.pendingPull[key]
	delete(c.file.pendingPull, key)
	return t
}

func (c *Controller) fileFailPendingPull(provider uint32, dir, name, msg string) {
	if c == nil || c.file == nil {
		return
	}
	key := filePullKey{provider: provider, dir: strings.TrimSpace(dir), name: strings.TrimSpace(name)}
	c.file.mu.Lock()
	t := c.file.pendingPull[key]
	delete(c.file.pendingPull, key)
	c.file.mu.Unlock()
	if t == nil {
		return
	}
	t.mu.Lock()
	t.status = "失败"
	t.lastError = msg
	t.updatedAt = time.Now()
	t.mu.Unlock()
	c.fileRefreshUIThrottled()
}

func (c *Controller) fileRefreshUIThrottled() {
	if c == nil || c.file == nil {
		return
	}
	c.file.mu.Lock()
	list := c.file.list
	if list == nil {
		c.file.mu.Unlock()
		return
	}
	now := time.Now()
	if !c.file.lastUI.IsZero() && now.Sub(c.file.lastUI) < 200*time.Millisecond {
		c.file.mu.Unlock()
		return
	}
	c.file.lastUI = now
	c.file.mu.Unlock()
	runOnMain(c, func() { list.Refresh() })
}

func fileHashSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func fileVerifyPart(partPath string, size uint64, shaHex string) bool {
	info, err := os.Stat(partPath)
	if err != nil || info == nil || info.IsDir() {
		return false
	}
	if uint64(info.Size()) != size {
		return false
	}
	shaHex = strings.TrimSpace(strings.ToLower(shaHex))
	if shaHex == "" {
		return true
	}
	got, err := fileHashSHA256(partPath)
	if err != nil {
		return false
	}
	return strings.EqualFold(got, shaHex)
}

func fileURIToLocalPath(uri fyne.URI) string {
	if uri == nil {
		return ""
	}
	p := uri.Path()
	if p == "" {
		return ""
	}
	if runtime.GOOS == "windows" {
		if strings.HasPrefix(p, "/") && len(p) >= 3 && p[2] == ':' {
			p = p[1:]
		}
	}
	return filepath.FromSlash(p)
}

func (c *Controller) fileHandleData(h core.IHeader, payload []byte) {
	if c == nil || h == nil {
		return
	}
	kind, bh, body, ok := fileDecodeBinHeaderV1(payload)
	if !ok || kind != fileKindData || bh.Ver != fileBinVerV1 {
		return
	}
	sess := c.fileGetRecvSession(bh.SessionID)
	if sess == nil {
		return
	}
	sess.mu.Lock()
	defer sess.mu.Unlock()
	if h.TargetID() != sess.consumer || h.SourceID() != sess.provider {
		return
	}
	if sess.file == nil {
		return
	}
	if sess.expectedOffset > sess.size {
		return
	}

	fin := (bh.Flags & fileBinFlagFIN) != 0
	offset := bh.Offset
	if offset < sess.expectedOffset {
		c.fileMaybeAckLocked(sess, false)
		return
	}
	if offset > sess.expectedOffset {
		if sess.pendingBytes+uint64(len(body)) > sess.maxPendingBytes {
			c.fileMaybeAckLocked(sess, false)
			return
		}
		if _, ok := sess.pending[offset]; !ok {
			cp := make([]byte, len(body))
			copy(cp, body)
			sess.pending[offset] = cp
			sess.pendingBytes += uint64(len(body))
		}
		c.fileMaybeAckLocked(sess, false)
		return
	}

	if len(body) > 0 {
		if sess.expectedOffset+uint64(len(body)) > sess.size {
			return
		}
		if _, err := sess.file.Write(body); err != nil {
			c.fileFailRecvLocked(sess, "write failed")
			return
		}
		if sess.hasher != nil {
			_, _ = sess.hasher.Write(body)
		}
		sess.expectedOffset += uint64(len(body))
	}
	if fin {
		sess.finSeen = true
	}
	sess.lastActive = time.Now()
	if sess.task != nil {
		sess.task.mu.Lock()
		sess.task.doneBytes = sess.expectedOffset
		sess.task.updatedAt = time.Now()
		sess.task.mu.Unlock()
	}

	for {
		next, ok := sess.pending[sess.expectedOffset]
		if !ok {
			break
		}
		delete(sess.pending, sess.expectedOffset)
		sess.pendingBytes -= uint64(len(next))
		if sess.expectedOffset+uint64(len(next)) > sess.size {
			c.fileFailRecvLocked(sess, "overflow")
			return
		}
		if _, err := sess.file.Write(next); err != nil {
			c.fileFailRecvLocked(sess, "write failed")
			return
		}
		if sess.hasher != nil {
			_, _ = sess.hasher.Write(next)
		}
		sess.expectedOffset += uint64(len(next))
		if sess.task != nil {
			sess.task.mu.Lock()
			sess.task.doneBytes = sess.expectedOffset
			sess.task.updatedAt = time.Now()
			sess.task.mu.Unlock()
		}
	}

	done := sess.finSeen && sess.expectedOffset == sess.size
	c.fileMaybeAckLocked(sess, done)
	if done {
		c.fileFinishRecvLocked(sess)
	}
	c.fileRefreshUIThrottled()
}

func (c *Controller) fileHandleAck(h core.IHeader, payload []byte) {
	if c == nil || h == nil {
		return
	}
	kind, bh, _, ok := fileDecodeBinHeaderV1(payload)
	if !ok || kind != fileKindAck || bh.Ver != fileBinVerV1 {
		return
	}
	sess := c.fileGetSendSession(bh.SessionID)
	if sess == nil {
		return
	}
	done := false
	sess.mu.Lock()
	if h.TargetID() != sess.provider || h.SourceID() != sess.consumer {
		sess.mu.Unlock()
		return
	}
	if bh.Offset <= sess.size && bh.Offset > sess.ackedUntil {
		sess.ackedUntil = bh.Offset
	}
	sess.lastActive = time.Now()
	if sess.task != nil {
		sess.task.mu.Lock()
		sess.task.ackedBytes = sess.ackedUntil
		if sess.sentEOF && sess.ackedUntil == sess.size {
			sess.task.status = "完成"
		}
		sess.task.updatedAt = time.Now()
		sess.task.mu.Unlock()
	}
	done = sess.sentEOF && sess.ackedUntil == sess.size
	id := sess.id
	sess.mu.Unlock()
	if done {
		c.fileRemoveSendSession(id)
	}
	c.fileRefreshUIThrottled()
}

func (c *Controller) fileMaybeAckLocked(sess *fileRecvSession, force bool) {
	if sess == nil {
		return
	}
	cfg := c.fileConfig()
	need := force
	if !need && sess.expectedOffset-sess.lastAckOffset >= cfg.AckEveryBytes {
		need = true
	}
	if !need && time.Since(sess.lastAckTime) >= cfg.AckEvery {
		need = true
	}
	if !need {
		return
	}
	sess.lastAckOffset = sess.expectedOffset
	sess.lastAckTime = time.Now()
	go c.fileSendAck(sess.provider, sess.id, sess.expectedOffset)
}

func (c *Controller) fileSendAck(provider uint32, sid [16]byte, offset uint64) {
	consumer := c.storedNode
	if c == nil || c.session == nil || consumer == 0 || provider == 0 {
		return
	}
	payload := fileEncodeBinHeaderV1(fileKindAck, sid, offset, false, nil)
	hdr := (&header.HeaderTcp{}).
		WithMajor(header.MajorMsg).
		WithSubProto(subProtoFile).
		WithSourceID(consumer).
		WithTargetID(provider).
		WithMsgID(uint32(time.Now().UnixNano()))
	_ = c.session.Send(hdr, payload)
}

func (c *Controller) fileFailRecvLocked(sess *fileRecvSession, reason string) {
	if sess == nil {
		return
	}
	if sess.task != nil {
		sess.task.mu.Lock()
		sess.task.status = "失败"
		sess.task.lastError = reason
		sess.task.updatedAt = time.Now()
		sess.task.mu.Unlock()
	}
	go c.fileRemoveRecvSession(sess.id)
}

func (c *Controller) fileFinishRecvLocked(sess *fileRecvSession) {
	if sess == nil {
		return
	}
	if sess.file != nil {
		_ = sess.file.Close()
		sess.file = nil
	}
	ok := true
	if sess.size != 0 {
		info, err := os.Stat(sess.partPath)
		if err != nil || info == nil || info.IsDir() || uint64(info.Size()) != sess.size {
			ok = false
		}
	}
	if ok && sess.hasher != nil && strings.TrimSpace(sess.sha256Hex) != "" {
		got := hex.EncodeToString(sess.hasher.Sum(nil))
		if !strings.EqualFold(got, strings.TrimSpace(sess.sha256Hex)) {
			ok = false
		}
	}
	if !ok {
		if sess.task != nil {
			sess.task.mu.Lock()
			sess.task.status = "失败"
			sess.task.lastError = "integrity failed"
			sess.task.updatedAt = time.Now()
			sess.task.mu.Unlock()
		}
		go c.fileRemoveRecvSession(sess.id)
		return
	}
	if sess.overwrite {
		_ = os.Remove(sess.finalPath)
	}
	_ = os.Rename(sess.partPath, sess.finalPath)
	if sess.task != nil {
		sess.task.mu.Lock()
		sess.task.status = "完成"
		sess.task.localPath = sess.finalPath
		sess.task.updatedAt = time.Now()
		sess.task.mu.Unlock()
	}
	go c.fileRemoveRecvSession(sess.id)
}

func (c *Controller) fileSendData(id [16]byte) {
	sess := c.fileGetSendSession(id)
	if c == nil || sess == nil || c.session == nil {
		return
	}
	sess.mu.Lock()
	provider := sess.provider
	consumer := sess.consumer
	filePath := sess.filePath
	size := sess.size
	startFrom := sess.startFrom
	task := sess.task
	sess.mu.Unlock()

	ctx := c.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	sess.mu.Lock()
	sess.cancel = cancel
	sess.mu.Unlock()
	defer cancel()

	f, err := os.Open(filePath)
	if err != nil {
		sess.mu.Lock()
		if task != nil {
			task.mu.Lock()
			task.status = "失败"
			task.lastError = "open failed"
			task.updatedAt = time.Now()
			task.mu.Unlock()
		}
		sess.mu.Unlock()
		c.fileRemoveSendSession(id)
		c.fileRefreshUIThrottled()
		return
	}
	defer func() { _ = f.Close() }()
	if _, err := f.Seek(int64(startFrom), io.SeekStart); err != nil {
		sess.mu.Lock()
		if task != nil {
			task.mu.Lock()
			task.status = "失败"
			task.lastError = "seek failed"
			task.updatedAt = time.Now()
			task.mu.Unlock()
		}
		sess.mu.Unlock()
		c.fileRemoveSendSession(id)
		c.fileRefreshUIThrottled()
		return
	}

	chunkBytes := c.fileConfig().ChunkBytes
	if chunkBytes <= 0 {
		chunkBytes = 256 * 1024
	}
	buf := make([]byte, chunkBytes)
	offset := startFrom

	sendData := func(body []byte, fin bool) error {
		payload := fileEncodeBinHeaderV1(fileKindData, id, offset, fin, body)
		hdr := (&header.HeaderTcp{}).
			WithMajor(header.MajorMsg).
			WithSubProto(subProtoFile).
			WithSourceID(provider).
			WithTargetID(consumer).
			WithMsgID(uint32(time.Now().UnixNano()))
		return c.session.Send(hdr, payload)
	}

	if size == 0 {
		if err := sendData(nil, true); err != nil {
			sess.mu.Lock()
			if task != nil {
				task.mu.Lock()
				task.status = "失败"
				task.lastError = "send failed"
				task.updatedAt = time.Now()
				task.mu.Unlock()
			}
			sess.mu.Unlock()
			c.fileRemoveSendSession(id)
			c.fileRefreshUIThrottled()
			return
		}
		sess.mu.Lock()
		sess.sentEOF = true
		if task != nil {
			task.mu.Lock()
			task.status = "等待确认"
			task.updatedAt = time.Now()
			task.mu.Unlock()
		}
		sess.mu.Unlock()
		c.fileRefreshUIThrottled()
		return
	}

	for {
		select {
		case <-ctx.Done():
			sess.mu.Lock()
			if task != nil {
				task.mu.Lock()
				if task.status != "完成" {
					task.status = "取消"
					task.updatedAt = time.Now()
				}
				task.mu.Unlock()
			}
			sess.mu.Unlock()
			c.fileRefreshUIThrottled()
			return
		default:
		}
		n, rerr := f.Read(buf)
		if n == 0 && rerr == io.EOF {
			break
		}
		if rerr != nil && rerr != io.EOF {
			sess.mu.Lock()
			if task != nil {
				task.mu.Lock()
				task.status = "失败"
				task.lastError = "read failed"
				task.updatedAt = time.Now()
				task.mu.Unlock()
			}
			sess.mu.Unlock()
			c.fileRemoveSendSession(id)
			c.fileRefreshUIThrottled()
			return
		}
		body := buf[:n]
		fin := offset+uint64(n) == size
		if err := sendData(body, fin); err != nil {
			sess.mu.Lock()
			if task != nil {
				task.mu.Lock()
				task.status = "失败"
				task.lastError = "send failed"
				task.updatedAt = time.Now()
				task.mu.Unlock()
			}
			sess.mu.Unlock()
			c.fileRemoveSendSession(id)
			c.fileRefreshUIThrottled()
			return
		}
		offset += uint64(n)
		sess.mu.Lock()
		sess.lastActive = time.Now()
		if task != nil {
			task.mu.Lock()
			task.sentBytes = offset
			task.updatedAt = time.Now()
			task.mu.Unlock()
		}
		if fin {
			sess.sentEOF = true
			if task != nil {
				task.mu.Lock()
				task.status = "等待确认"
				task.mu.Unlock()
			}
		}
		sess.mu.Unlock()
		c.fileRefreshUIThrottled()
		if fin {
			break
		}
	}
}

// ---- UI entrypoints (由 Win 文件传输界面触发) ----

func (c *Controller) fileStartPull(provider uint32, dir, name, saveDir, saveName string, wantHash bool) error {
	if c == nil || c.session == nil {
		return fmt.Errorf("尚未连接")
	}
	if c.storedNode == 0 || c.storedHub == 0 {
		return fmt.Errorf("尚未登录")
	}
	if provider == 0 {
		return fmt.Errorf("Provider 不能为空")
	}
	dir = strings.ReplaceAll(strings.TrimSpace(dir), "\\", "/")
	name = strings.TrimSpace(name)
	saveDir = strings.ReplaceAll(strings.TrimSpace(saveDir), "\\", "/")
	if _, err := fileSanitizeDir(dir); err != nil {
		return fmt.Errorf("dir 非法")
	}
	if _, err := fileSanitizeName(name); err != nil {
		return fmt.Errorf("name 非法")
	}
	if _, err := fileSanitizeDir(saveDir); err != nil {
		return fmt.Errorf("保存目录非法")
	}
	saveName = strings.TrimSpace(saveName)
	if saveName != "" {
		if _, err := fileSanitizeName(saveName); err != nil {
			return fmt.Errorf("保存文件名非法")
		}
	} else {
		saveName = name
	}

	cfg := c.fileConfig()
	finalPath, partPath, err := fileResolvePaths(cfg.BaseDir, saveDir, saveName)
	if err != nil {
		return fmt.Errorf("保存路径非法")
	}
	if err := os.MkdirAll(filepath.Dir(finalPath), 0o755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}
	resumeFrom := uint64(0)
	if st, err := os.Stat(partPath); err == nil && st != nil && !st.IsDir() {
		if st.Size() > 0 {
			resumeFrom = uint64(st.Size())
		}
	}

	overwrite := true
	req := fileReadReq{
		Op:         fileOpPull,
		Target:     provider,
		Dir:        dir,
		Name:       name,
		Overwrite:  &overwrite,
		ResumeFrom: resumeFrom,
		WantHash:   &wantHash,
	}

	taskID, _ := fileNewUUID()
	task := &fileTask{
		taskID:    taskID,
		createdAt: time.Now(),
		updatedAt: time.Now(),
		op:        fileOpPull,
		direction: "download",
		provider:  provider,
		consumer:  c.storedNode,
		peer:      provider,
		dir:       dir,
		name:      name,
		wantHash:  wantHash,
		localDir:  saveDir,
		localName: saveName,
		localPath: finalPath,
		status:    "等待响应",
	}
	c.fileAddTask(task)
	c.file.mu.Lock()
	c.file.pendingPull[filePullKey{provider: provider, dir: strings.TrimSpace(dir), name: strings.TrimSpace(name)}] = task
	c.file.mu.Unlock()

	if err := c.fileSendCtrl(c.storedHub, fileMessage{Action: fileActionRead, Data: fileMustJSON(req)}); err != nil {
		task.mu.Lock()
		task.status = "失败"
		task.lastError = err.Error()
		task.updatedAt = time.Now()
		task.mu.Unlock()
		c.fileRefreshUIThrottled()
		return err
	}
	return nil
}

func (c *Controller) fileStartOffer(consumer uint32, filePath string, wantHash bool) error {
	if c == nil || c.session == nil {
		return fmt.Errorf("尚未连接")
	}
	if c.storedNode == 0 || c.storedHub == 0 {
		return fmt.Errorf("尚未登录")
	}
	if consumer == 0 {
		return fmt.Errorf("Target 不能为空")
	}
	cfg := c.fileConfig()
	baseAbs, err := filepath.Abs(cfg.BaseDir)
	if err != nil {
		return fmt.Errorf("base_dir 无效")
	}
	absFile, err := filepath.Abs(strings.TrimSpace(filePath))
	if err != nil {
		return fmt.Errorf("文件路径无效")
	}
	rel, err := filepath.Rel(baseAbs, absFile)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("只能发送 base_dir 下的文件")
	}
	relSlash := filepath.ToSlash(rel)
	dir := path.Dir(relSlash)
	if dir == "." {
		dir = ""
	}
	name := path.Base(relSlash)
	if _, err := fileSanitizeDir(dir); err != nil {
		return fmt.Errorf("dir 非法")
	}
	if _, err := fileSanitizeName(name); err != nil {
		return fmt.Errorf("name 非法")
	}

	info, err := os.Stat(absFile)
	if err != nil || info == nil || info.IsDir() {
		return fmt.Errorf("文件不存在")
	}
	size := uint64(info.Size())
	if cfg.MaxSizeBytes > 0 && size > cfg.MaxSizeBytes {
		return fmt.Errorf("文件过大")
	}
	if c.fileTotalSessions() >= cfg.MaxConcurrent {
		return fmt.Errorf("并发会话过多")
	}

	sid, err := fileNewUUID()
	if err != nil {
		return fmt.Errorf("生成 session 失败")
	}
	taskID, _ := fileNewUUID()
	task := &fileTask{
		taskID:       taskID,
		sessionID:    sid,
		hasSessionID: true,
		createdAt:    time.Now(),
		updatedAt:    time.Now(),
		op:           fileOpOffer,
		direction:    "upload",
		provider:     c.storedNode,
		consumer:     consumer,
		peer:         consumer,
		dir:          dir,
		name:         name,
		size:         size,
		wantHash:     wantHash,
		filePath:     absFile,
		status:       "准备中",
	}
	sess := &fileSendSession{
		id:         sid,
		provider:   c.storedNode,
		consumer:   consumer,
		dir:        dir,
		name:       name,
		filePath:   absFile,
		size:       size,
		startFrom:  0,
		lastActive: time.Now(),
		task:       task,
	}
	c.fileAddTask(task)
	c.fileAddSendSession(sess)

	go func() {
		shaHex := ""
		if wantHash {
			task.mu.Lock()
			task.status = "计算SHA256"
			task.updatedAt = time.Now()
			task.mu.Unlock()
			c.fileRefreshUIThrottled()
			if sh, err := fileHashSHA256(absFile); err == nil {
				shaHex = sh
			}
		}
		sess.mu.Lock()
		sess.sha256Hex = shaHex
		if sess.task != nil {
			sess.task.mu.Lock()
			sess.task.sha256 = shaHex
			sess.task.status = "等待对方确认"
			sess.task.updatedAt = time.Now()
			sess.task.mu.Unlock()
		}
		sess.mu.Unlock()
		c.fileRefreshUIThrottled()

		overwrite := true
		req := fileWriteReq{
			Op:        fileOpOffer,
			Target:    consumer,
			SessionID: fileUUIDToString(sid),
			Dir:       dir,
			Name:      name,
			Size:      size,
			Sha256:    shaHex,
			Overwrite: &overwrite,
		}
		if err := c.fileSendCtrl(c.storedHub, fileMessage{Action: fileActionWrite, Data: fileMustJSON(req)}); err != nil {
			sess.mu.Lock()
			if sess.task != nil {
				sess.task.mu.Lock()
				sess.task.status = "失败"
				sess.task.lastError = err.Error()
				sess.task.updatedAt = time.Now()
				sess.task.mu.Unlock()
			}
			sess.mu.Unlock()
			c.fileRemoveSendSession(sid)
			c.fileRefreshUIThrottled()
		}
	}()
	return nil
}

func (c *Controller) filePromptIncomingOffer(h core.IHeader, req fileWriteReq) {
	win := resolveWindow(c.app, c.mainWin, nil)
	if c == nil || win == nil {
		c.fileSendWriteResp(h.SourceID(), fileWriteResp{Code: 500, Msg: "ui not ready", Op: fileOpOffer, SessionID: strings.TrimSpace(req.SessionID), Accept: false})
		return
	}
	sid, sidOK := fileParseUUID(req.SessionID)
	if sidOK {
		c.file.mu.Lock()
		task := c.file.taskBySid[sid]
		c.file.mu.Unlock()
		if task == nil {
			taskID, _ := fileNewUUID()
			task = &fileTask{
				taskID:       taskID,
				sessionID:    sid,
				hasSessionID: true,
				createdAt:    time.Now(),
				updatedAt:    time.Now(),
				op:           fileOpOffer,
				direction:    "download",
				provider:     h.SourceID(),
				consumer:     c.storedNode,
				peer:         h.SourceID(),
				dir:          strings.ReplaceAll(strings.TrimSpace(req.Dir), "\\", "/"),
				name:         strings.TrimSpace(req.Name),
				size:         req.Size,
				sha256:       strings.TrimSpace(req.Sha256),
				wantHash:     strings.TrimSpace(req.Sha256) != "",
				status:       "待确认",
			}
			c.fileAddTask(task)
		} else {
			task.mu.Lock()
			task.status = "待确认"
			task.updatedAt = time.Now()
			task.mu.Unlock()
			c.fileRefreshUIThrottled()
		}
	}

	cfg := c.fileConfig()
	baseAbs, _ := filepath.Abs(cfg.BaseDir)
	if strings.TrimSpace(baseAbs) == "" {
		baseAbs = "."
	}
	defaultDir := strings.ReplaceAll(strings.TrimSpace(req.Dir), "\\", "/")
	if d, err := fileSanitizeDir(defaultDir); err == nil {
		defaultDir = d
	} else {
		defaultDir = ""
	}
	saveDir := widget.NewEntry()
	saveDir.SetText(defaultDir)
	chooseBtn := widget.NewButton("选择保存目录", func() {
		showBaseDirFolderPicker(c, win, cfg.BaseDir, func(abs string) {
			baseAbs, _ := filepath.Abs(cfg.BaseDir)
			rel, err := filepath.Rel(baseAbs, abs)
			if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
				dialog.ShowError(fmt.Errorf("请选择 base_dir(%s) 内的目录", baseAbs), win)
				return
			}
			relSlash := filepath.ToSlash(rel)
			if relSlash == "." {
				relSlash = ""
			}
			saveDir.SetText(relSlash)
		})
	})

	info := widget.NewLabel(fmt.Sprintf("来自节点 %d 的 offer\n文件: %s\n目录: %s\n大小: %d bytes\nsha256: %s",
		h.SourceID(), strings.TrimSpace(req.Name), strings.TrimSpace(req.Dir), req.Size, strings.TrimSpace(req.Sha256)))
	content := container.NewVBox(
		info,
		widget.NewSeparator(),
		widget.NewLabel("保存到（相对 base_dir 的目录，可为空）"),
		saveDir,
		chooseBtn,
	)

	dialog.ShowCustomConfirm("接收文件", "接收", "拒绝", content, func(accepted bool) {
		if !accepted {
			if sidOK {
				c.file.mu.Lock()
				task := c.file.taskBySid[sid]
				c.file.mu.Unlock()
				if task != nil {
					task.mu.Lock()
					task.status = "已拒绝"
					task.updatedAt = time.Now()
					task.mu.Unlock()
					c.fileRefreshUIThrottled()
				}
			}
			c.fileSendWriteResp(h.SourceID(), fileWriteResp{Code: 403, Msg: "rejected", Op: fileOpOffer, SessionID: strings.TrimSpace(req.SessionID), Accept: false})
			return
		}
		c.fileAcceptOffer(h.SourceID(), req, saveDir.Text)
	}, win)
}

func (c *Controller) fileAcceptOffer(provider uint32, req fileWriteReq, saveDir string) {
	localNode := c.storedNode
	if c == nil || c.session == nil || localNode == 0 {
		return
	}
	cfg := c.fileConfig()
	if c.fileTotalSessions() >= cfg.MaxConcurrent {
		c.fileSendWriteResp(provider, fileWriteResp{Code: 429, Msg: "too many sessions", Op: fileOpOffer, SessionID: strings.TrimSpace(req.SessionID), Accept: false})
		return
	}
	sid, ok := fileParseUUID(req.SessionID)
	if !ok {
		c.fileSendWriteResp(provider, fileWriteResp{Code: 400, Msg: "invalid session", Op: fileOpOffer, SessionID: strings.TrimSpace(req.SessionID), Accept: false})
		return
	}
	dir := strings.ReplaceAll(strings.TrimSpace(saveDir), "\\", "/")
	if _, err := fileSanitizeDir(dir); err != nil {
		c.fileSendWriteResp(provider, fileWriteResp{Code: 400, Msg: "invalid dir", Op: fileOpOffer, SessionID: strings.TrimSpace(req.SessionID), Accept: false})
		return
	}
	name := strings.TrimSpace(req.Name)
	finalPath, partPath, err := fileResolvePaths(cfg.BaseDir, dir, name)
	if err != nil {
		c.fileSendWriteResp(provider, fileWriteResp{Code: 400, Msg: "invalid path", Op: fileOpOffer, SessionID: strings.TrimSpace(req.SessionID), Accept: false})
		return
	}
	if err := os.MkdirAll(filepath.Dir(finalPath), 0o755); err != nil {
		c.fileSendWriteResp(provider, fileWriteResp{Code: 500, Msg: "mkdir failed", Op: fileOpOffer, SessionID: strings.TrimSpace(req.SessionID), Accept: false})
		return
	}

	overwrite := true
	if req.Overwrite != nil {
		overwrite = *req.Overwrite
	}
	if !overwrite {
		if _, err := os.Stat(finalPath); err == nil {
			c.fileSendWriteResp(provider, fileWriteResp{Code: 409, Msg: "exists", Op: fileOpOffer, SessionID: strings.TrimSpace(req.SessionID), Accept: false})
			return
		}
	}

	resumeFrom := uint64(0)
	if st, err := os.Stat(partPath); err == nil && st != nil && !st.IsDir() {
		if uint64(st.Size()) <= req.Size {
			resumeFrom = uint64(st.Size())
		} else {
			_ = os.Remove(partPath)
		}
	}
	if resumeFrom == req.Size && req.Size != 0 {
		if fileVerifyPart(partPath, req.Size, strings.TrimSpace(req.Sha256)) {
			if overwrite {
				_ = os.Remove(finalPath)
			}
			_ = os.Rename(partPath, finalPath)
		} else {
			_ = os.Remove(partPath)
			resumeFrom = 0
		}
	}
	if req.Size != 0 && resumeFrom == req.Size {
		c.file.mu.Lock()
		task := c.file.taskBySid[sid]
		c.file.mu.Unlock()
		if task != nil {
			task.mu.Lock()
			task.status = "完成"
			task.localDir = dir
			task.localPath = finalPath
			task.doneBytes = req.Size
			task.updatedAt = time.Now()
			task.mu.Unlock()
			c.fileRefreshUIThrottled()
		}
		c.fileSendWriteResp(provider, fileWriteResp{
			Code:       1,
			Msg:        "ok",
			Op:         fileOpOffer,
			SessionID:  strings.TrimSpace(req.SessionID),
			Provider:   provider,
			Consumer:   localNode,
			Dir:        strings.ReplaceAll(strings.TrimSpace(req.Dir), "\\", "/"),
			Name:       name,
			Size:       req.Size,
			Sha256:     strings.TrimSpace(req.Sha256),
			Accept:     true,
			ResumeFrom: resumeFrom,
		})
		return
	}

	f, err := os.OpenFile(partPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		c.fileSendWriteResp(provider, fileWriteResp{Code: 500, Msg: "open failed", Op: fileOpOffer, SessionID: strings.TrimSpace(req.SessionID), Accept: false})
		return
	}
	if err := f.Truncate(int64(resumeFrom)); err != nil {
		_ = f.Close()
		c.fileSendWriteResp(provider, fileWriteResp{Code: 500, Msg: "truncate failed", Op: fileOpOffer, SessionID: strings.TrimSpace(req.SessionID), Accept: false})
		return
	}
	var hasher hash.Hash
	shaHex := strings.TrimSpace(req.Sha256)
	if shaHex != "" {
		hasher = sha256.New()
	}
	if hasher != nil && resumeFrom > 0 {
		if _, err := f.Seek(0, io.SeekStart); err == nil {
			buf := make([]byte, 256*1024)
			remain := resumeFrom
			for remain > 0 {
				n := int64(len(buf))
				if remain < uint64(len(buf)) {
					n = int64(remain)
				}
				readN, rerr := io.ReadFull(f, buf[:n])
				if readN > 0 {
					_, _ = hasher.Write(buf[:readN])
					remain -= uint64(readN)
				}
				if rerr != nil {
					break
				}
			}
		}
	}
	if _, err := f.Seek(int64(resumeFrom), io.SeekStart); err != nil {
		_ = f.Close()
		c.fileSendWriteResp(provider, fileWriteResp{Code: 500, Msg: "seek failed", Op: fileOpOffer, SessionID: strings.TrimSpace(req.SessionID), Accept: false})
		return
	}

	c.file.mu.Lock()
	task := c.file.taskBySid[sid]
	c.file.mu.Unlock()
	if task == nil {
		taskID, _ := fileNewUUID()
		task = &fileTask{
			taskID:       taskID,
			sessionID:    sid,
			hasSessionID: true,
			createdAt:    time.Now(),
			updatedAt:    time.Now(),
			op:           fileOpOffer,
			direction:    "download",
			provider:     provider,
			consumer:     localNode,
			peer:         provider,
			dir:          strings.ReplaceAll(strings.TrimSpace(req.Dir), "\\", "/"),
			name:         name,
			size:         req.Size,
			sha256:       shaHex,
			wantHash:     shaHex != "",
			status:       "接收中",
		}
		c.fileAddTask(task)
		c.file.mu.Lock()
		c.file.taskBySid[sid] = task
		c.file.mu.Unlock()
	} else {
		task.mu.Lock()
		task.status = "接收中"
		task.localDir = dir
		task.localPath = finalPath
		task.size = req.Size
		task.sha256 = shaHex
		task.doneBytes = resumeFrom
		task.updatedAt = time.Now()
		task.mu.Unlock()
		c.fileRefreshUIThrottled()
	}

	sess := &fileRecvSession{
		id:              sid,
		provider:        provider,
		consumer:        localNode,
		dir:             strings.ReplaceAll(strings.TrimSpace(req.Dir), "\\", "/"),
		name:            name,
		finalPath:       finalPath,
		partPath:        partPath,
		size:            req.Size,
		sha256Hex:       shaHex,
		overwrite:       overwrite,
		file:            f,
		hasher:          hasher,
		expectedOffset:  resumeFrom,
		pending:         make(map[uint64][]byte),
		maxPendingBytes: uint64(cfg.ChunkBytes) * 8,
		lastActive:      time.Now(),
		task:            task,
	}
	c.fileAddRecvSession(sess)
	c.fileRefreshUIThrottled()

	c.fileSendWriteResp(provider, fileWriteResp{
		Code:       1,
		Msg:        "ok",
		Op:         fileOpOffer,
		SessionID:  strings.TrimSpace(req.SessionID),
		Provider:   provider,
		Consumer:   localNode,
		Dir:        strings.ReplaceAll(strings.TrimSpace(req.Dir), "\\", "/"),
		Name:       name,
		Size:       req.Size,
		Sha256:     shaHex,
		Accept:     true,
		ResumeFrom: resumeFrom,
	})
}

func (c *Controller) fileOnDisconnect(reason error) {
	if c == nil || c.file == nil {
		return
	}
	msg := "connection closed"
	if reason != nil {
		msg = reason.Error()
	}
	c.file.mu.Lock()
	recvIDs := make([][16]byte, 0, len(c.file.recv))
	for id := range c.file.recv {
		recvIDs = append(recvIDs, id)
	}
	sendIDs := make([][16]byte, 0, len(c.file.send))
	for id := range c.file.send {
		sendIDs = append(sendIDs, id)
	}
	for _, t := range c.file.tasks {
		if t == nil {
			continue
		}
		t.mu.Lock()
		switch t.status {
		case "发送中", "接收中", "等待确认", "等待响应", "等待对方确认", "准备中", "计算SHA256", "待确认":
			t.status = "失败"
			t.lastError = msg
			t.updatedAt = time.Now()
		}
		t.mu.Unlock()
	}
	c.file.mu.Unlock()
	for _, id := range recvIDs {
		c.fileRemoveRecvSession(id)
	}
	for _, id := range sendIDs {
		c.fileRemoveSendSession(id)
	}
	c.fileRefreshUIThrottled()
}
