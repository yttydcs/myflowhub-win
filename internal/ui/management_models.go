package ui

type mgmtNodeEntry struct {
	ID          uint32
	HasChildren bool
}

type mgmtConfigEntry struct {
	Key   string
	Value string
}
