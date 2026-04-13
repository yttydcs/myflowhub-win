package stream

const (
	EventStreamDelivery = "stream.delivery"
	EventStreamText     = "stream.text"
	EventStreamStats    = "stream.stats"
	EventStreamMedia    = "stream.media"
)

type StreamDeliveryEvent struct {
	DeliveryID   string `json:"deliveryId"`
	SourceID     string `json:"sourceId,omitempty"`
	Producer     uint32 `json:"producer,omitempty"`
	Consumer     uint32 `json:"consumer,omitempty"`
	ConsumerID   string `json:"consumerId,omitempty"`
	Kind         string `json:"kind,omitempty"`
	ContentType  string `json:"contentType,omitempty"`
	Mode         string `json:"mode,omitempty"`
	UnitMode     string `json:"unitMode,omitempty"`
	State        string `json:"state"`
	BytesIn      uint64 `json:"bytesIn"`
	FramesIn     uint64 `json:"framesIn"`
	LastPosition uint64 `json:"lastPosition"`
	LastPtsMs    uint64 `json:"lastPtsMs"`
	LastAckPos   uint64 `json:"lastAckPos"`
	LastFlags    uint8  `json:"lastFlags"`
	LastError    string `json:"lastError,omitempty"`
	UpdatedAt    string `json:"updatedAt"`
}

type StreamTextEvent struct {
	DeliveryID string `json:"deliveryId"`
	Kind       string `json:"kind"`
	Text       string `json:"text"`
	Position   uint64 `json:"position"`
	PtsMs      uint64 `json:"ptsMs"`
	Flags      uint8  `json:"flags"`
	UpdatedAt  string `json:"updatedAt"`
}

type StreamStatsEvent struct {
	DeliveryID   string `json:"deliveryId"`
	Kind         string `json:"kind"`
	BytesIn      uint64 `json:"bytesIn"`
	FramesIn     uint64 `json:"framesIn"`
	LastPosition uint64 `json:"lastPosition"`
	LastPtsMs    uint64 `json:"lastPtsMs"`
	LastAckPos   uint64 `json:"lastAckPos"`
	LastFlags    uint8  `json:"lastFlags"`
	UpdatedAt    string `json:"updatedAt"`
}

type StreamMediaEvent struct {
	DeliveryID     string `json:"deliveryId"`
	Kind           string `json:"kind"`
	ContentType    string `json:"contentType,omitempty"`
	State          string `json:"state"`
	MediaURL       string `json:"mediaUrl,omitempty"`
	AvailableBytes uint64 `json:"availableBytes"`
	Complete       bool   `json:"complete"`
	Error          string `json:"error,omitempty"`
	UpdatedAt      string `json:"updatedAt"`
}
