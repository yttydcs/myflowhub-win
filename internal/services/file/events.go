package file

import "time"

const (
	EventFileTasks = "file.tasks"
	EventFileList  = "file.list"
	EventFileText  = "file.text"
	EventFileOffer = "file.offer"
)

type FilePrefs struct {
	BaseDir           string `json:"baseDir"`
	MaxSizeBytes      uint64 `json:"maxSizeBytes"`
	MaxConcurrent     int    `json:"maxConcurrent"`
	ChunkBytes        int    `json:"chunkBytes"`
	IncompleteTTLSec  int64  `json:"incompleteTtlSec"`
	WantSHA256        bool   `json:"wantSha256"`
	AutoAccept        bool   `json:"autoAccept"`
}

type FileListEvent struct {
	NodeID uint32   `json:"nodeId"`
	Dir    string   `json:"dir"`
	Code   int      `json:"code"`
	Msg    string   `json:"msg"`
	Dirs   []string `json:"dirs"`
	Files  []string `json:"files"`
}

type FileTextEvent struct {
	NodeID    uint32 `json:"nodeId"`
	Dir       string `json:"dir"`
	Name      string `json:"name"`
	Code      int    `json:"code"`
	Msg       string `json:"msg"`
	Size      uint64 `json:"size"`
	Text      string `json:"text"`
	Truncated bool   `json:"truncated"`
}

type FileOfferEvent struct {
	SessionID  string `json:"sessionId"`
	Provider   uint32 `json:"provider"`
	Consumer   uint32 `json:"consumer"`
	Dir        string `json:"dir"`
	Name       string `json:"name"`
	Size       uint64 `json:"size"`
	Sha256     string `json:"sha256"`
	SuggestDir string `json:"suggestDir"`
}

type FileTaskView struct {
	TaskID    string `json:"taskId"`
	SessionID string `json:"sessionId"`

	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`

	Op        string `json:"op"`
	Direction string `json:"direction"`
	Status    string `json:"status"`
	LastError string `json:"lastError"`

	Provider uint32 `json:"provider"`
	Consumer uint32 `json:"consumer"`
	Peer     uint32 `json:"peer"`

	Dir  string `json:"dir"`
	Name string `json:"name"`
	Size uint64 `json:"size"`

	Sha256   string `json:"sha256"`
	WantHash bool   `json:"wantHash"`

	LocalDir  string `json:"localDir"`
	LocalName string `json:"localName"`
	LocalPath string `json:"localPath"`
	FilePath  string `json:"filePath"`

	SentBytes  uint64 `json:"sentBytes"`
	AckedBytes uint64 `json:"ackedBytes"`
	DoneBytes  uint64 `json:"doneBytes"`
}

type FileTasksEvent struct {
	Tasks     []FileTaskView `json:"tasks"`
	UpdatedAt time.Time      `json:"updatedAt"`
}
