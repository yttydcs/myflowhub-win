package auth

import (
	"encoding/json"
	"strings"
	"testing"

	protoauth "github.com/yttydcs/myflowhub-proto/protocol/auth"
	storagesvc "github.com/yttydcs/myflowhub-win/internal/storage"
)

func newTestStore(t *testing.T) *storagesvc.Store {
	t.Helper()
	base := t.TempDir()
	t.Setenv("APPDATA", base)
	t.Setenv("XDG_CONFIG_HOME", base)
	t.Setenv("HOME", base)

	store, err := storagesvc.NewStore()
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	return store
}

func TestNewRegisterRequestIncludesRawDisplayName(t *testing.T) {
	store := newTestStore(t)
	if err := store.SetRaw(nodeDisplayNameKey, "  raw-node  "); err != nil {
		t.Fatalf("SetRaw() error = %v", err)
	}
	if err := store.SetCurrentProfile("work"); err != nil {
		t.Fatalf("SetCurrentProfile() error = %v", err)
	}
	if err := store.SetString("work", nodeDisplayNameKey, "profile-node"); err != nil {
		t.Fatalf("SetString() error = %v", err)
	}

	svc := New(nil, nil, store)
	req := svc.newRegisterRequest("device-1", "pub-key")
	if req.DisplayName != "raw-node" {
		t.Fatalf("expected raw display_name, got %q", req.DisplayName)
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if !strings.Contains(string(data), `"display_name":"raw-node"`) {
		t.Fatalf("expected display_name in json, got %s", string(data))
	}
}

func TestNewRegisterRequestPrefersMCPDisplayName(t *testing.T) {
	store := newTestStore(t)
	if err := store.SetRaw(mcpDisplayNameKey, "  mcp-node  "); err != nil {
		t.Fatalf("SetRaw(mcp) error = %v", err)
	}
	if err := store.SetRaw(nodeDisplayNameKey, "legacy-node"); err != nil {
		t.Fatalf("SetRaw(legacy) error = %v", err)
	}

	svc := New(nil, nil, store)
	req := svc.newRegisterRequest("device-1", "pub-key")
	if req.DisplayName != "mcp-node" {
		t.Fatalf("expected mcp display_name, got %q", req.DisplayName)
	}
}

func TestNewLoginRequestFallsBackToProfileDisplayName(t *testing.T) {
	store := newTestStore(t)
	if err := store.SetCurrentProfile("work"); err != nil {
		t.Fatalf("SetCurrentProfile() error = %v", err)
	}
	if err := store.SetString("work", nodeDisplayNameKey, "  profile-node  "); err != nil {
		t.Fatalf("SetString() error = %v", err)
	}

	svc := New(nil, nil, store)
	req := svc.newLoginRequest(protoauth.LoginData{DeviceID: "device-1", NodeID: 7})
	if req.DisplayName != "profile-node" {
		t.Fatalf("expected profile display_name, got %q", req.DisplayName)
	}
}

func TestNewLoginRequestOmitsDisplayNameWhenEmpty(t *testing.T) {
	svc := New(nil, nil, newTestStore(t))
	req := svc.newLoginRequest(protoauth.LoginData{
		DeviceID: "device-1",
		NodeID:   7,
		TS:       123,
		Nonce:    "nonce",
		Sig:      "sig",
		Alg:      "ES256",
	})

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if strings.Contains(string(data), `"display_name"`) {
		t.Fatalf("expected display_name omitted, got %s", string(data))
	}
}

func TestExtractAuthCodeMsgSupportsListRegisterPermitsResp(t *testing.T) {
	resp := &ListRegisterPermitsResp{
		Code: 1,
		Msg:  "ok",
		Items: []RegisterPermitInfo{
			{Permit: "permit_123", DeviceID: "device-1", Role: "admin"},
		},
	}

	code, msg := extractAuthCodeMsg(resp)
	if code != 1 || msg != "ok" {
		t.Fatalf("extractAuthCodeMsg() = (%d, %q), want (1, %q)", code, msg, "ok")
	}
}
