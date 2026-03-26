package auth

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/yttydcs/myflowhub-proto/protocol/auth"
	"github.com/yttydcs/myflowhub-win/internal/services/logs"
	sessionsvc "github.com/yttydcs/myflowhub-win/internal/services/session"
	"github.com/yttydcs/myflowhub-win/internal/services/transport"
	storagesvc "github.com/yttydcs/myflowhub-win/internal/storage"
)

const defaultNodeKeysPath = "config/node_keys.json"

const defaultAuthTimeout = 8 * time.Second

const mcpDisplayNameKey = "mcp.display_name"
const nodeDisplayNameKey = "node.display_name"

type AuthService struct {
	session  *sessionsvc.SessionService
	logs     *logs.LogService
	store    *storagesvc.Store
	keyMu    sync.Mutex
	nodePriv *ecdsa.PrivateKey
	nodePub  string
	keysPath string
}

type registerRequest struct {
	auth.RegisterData
	DisplayName string `json:"display_name,omitempty"`
}

type loginRequest struct {
	auth.LoginData
	DisplayName string `json:"display_name,omitempty"`
}

func New(session *sessionsvc.SessionService, logsSvc *logs.LogService, store *storagesvc.Store) *AuthService {
	return &AuthService{session: session, logs: logsSvc, store: store, keysPath: defaultNodeKeysPath}
}

func (s *AuthService) SetKeysPath(path string) {
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}
	s.keyMu.Lock()
	cleaned := filepath.Clean(path)
	if s.keysPath != cleaned {
		s.keysPath = cleaned
		s.nodePriv = nil
		s.nodePub = ""
	}
	s.keyMu.Unlock()
}

func (s *AuthService) KeysPath() string {
	s.keyMu.Lock()
	defer s.keyMu.Unlock()
	return s.keysPath
}

func (s *AuthService) EnsureKeys() (string, error) {
	s.keyMu.Lock()
	defer s.keyMu.Unlock()
	if s.nodePriv != nil && strings.TrimSpace(s.nodePub) != "" {
		return s.nodePub, nil
	}
	priv, pub, err := loadOrCreateNodeKeys(s.keysPath)
	if err != nil {
		return "", err
	}
	s.nodePriv = priv
	s.nodePub = pub
	return pub, nil
}

func (s *AuthService) Register(ctx context.Context, sourceID, targetID uint32, deviceID string) (auth.RespData, error) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return auth.RespData{}, errors.New("device_id is required")
	}
	pub, err := s.EnsureKeys()
	if err != nil {
		return auth.RespData{}, err
	}
	payload, err := transport.EncodeMessage(auth.ActionRegister, s.newRegisterRequest(deviceID, pub))
	if err != nil {
		return auth.RespData{}, err
	}
	resp, err := s.sendAndAwait(ctx, sourceID, targetID, payload, auth.ActionRegister, auth.ActionRegisterResp)
	if err != nil {
		s.logs.Appendf("warn", "auth register failed device=%s: %v", deviceID, err)
		return auth.RespData{}, err
	}
	s.logs.Appendf("info", "auth register ok device=%s node=%d hub=%d role=%s", deviceID, resp.NodeID, resp.HubID, resp.Role)
	return resp, nil
}

func (s *AuthService) RegisterSimple(sourceID, targetID uint32, deviceID string) (auth.RespData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultAuthTimeout)
	defer cancel()
	return s.Register(ctx, sourceID, targetID, deviceID)
}

func (s *AuthService) Login(ctx context.Context, sourceID, targetID uint32, deviceID string, nodeID uint32) (auth.RespData, error) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return auth.RespData{}, errors.New("device_id is required")
	}
	if nodeID == 0 {
		return auth.RespData{}, errors.New("node_id is required")
	}
	login, err := s.SignLogin(deviceID, nodeID)
	if err != nil {
		return auth.RespData{}, err
	}
	payload, err := transport.EncodeMessage(auth.ActionLogin, s.newLoginRequest(login))
	if err != nil {
		return auth.RespData{}, err
	}
	resp, err := s.sendAndAwait(ctx, sourceID, targetID, payload, auth.ActionLogin, auth.ActionLoginResp)
	if err != nil {
		s.logs.Appendf("warn", "auth login failed device=%s node=%d: %v", deviceID, nodeID, err)
		return auth.RespData{}, err
	}
	s.logs.Appendf("info", "auth login ok device=%s node=%d hub=%d role=%s", deviceID, resp.NodeID, resp.HubID, resp.Role)
	return resp, nil
}

func (s *AuthService) LoginSimple(sourceID, targetID uint32, deviceID string, nodeID uint32) (auth.RespData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultAuthTimeout)
	defer cancel()
	return s.Login(ctx, sourceID, targetID, deviceID, nodeID)
}

func (s *AuthService) Send(ctx context.Context, sourceID, targetID uint32, action string, data any) error {
	payload, err := transport.EncodeMessage(action, data)
	if err != nil {
		return err
	}
	return s.send(ctx, sourceID, targetID, payload)
}

func (s *AuthService) SignLogin(deviceID string, nodeID uint32) (auth.LoginData, error) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return auth.LoginData{}, errors.New("device_id is required")
	}
	if nodeID == 0 {
		return auth.LoginData{}, errors.New("node_id is required")
	}
	s.keyMu.Lock()
	priv := s.nodePriv
	s.keyMu.Unlock()
	if priv == nil {
		if _, err := s.EnsureKeys(); err != nil {
			return auth.LoginData{}, err
		}
		s.keyMu.Lock()
		priv = s.nodePriv
		s.keyMu.Unlock()
	}
	if priv == nil {
		return auth.LoginData{}, errors.New("private key invalid")
	}
	ts := time.Now().Unix()
	nonce := generateNonce(12)
	sig, err := signLogin(priv, deviceID, nodeID, ts, nonce)
	if err != nil {
		return auth.LoginData{}, err
	}
	return auth.LoginData{
		DeviceID: deviceID,
		NodeID:   nodeID,
		TS:       ts,
		Nonce:    nonce,
		Sig:      sig,
		Alg:      "ES256",
	}, nil
}

func (s *AuthService) newRegisterRequest(deviceID, pub string) registerRequest {
	return registerRequest{
		RegisterData: auth.RegisterData{
			DeviceID: deviceID,
			PubKey:   pub,
			NodePub:  pub,
		},
		DisplayName: s.localNodeDisplayName(),
	}
}

func (s *AuthService) newLoginRequest(login auth.LoginData) loginRequest {
	return loginRequest{
		LoginData:   login,
		DisplayName: s.localNodeDisplayName(),
	}
}

func (s *AuthService) localNodeDisplayName() string {
	if s.store == nil {
		return ""
	}
	if raw, ok := s.store.GetRaw(mcpDisplayNameKey); ok {
		if displayName := normalizeNodeDisplayName(raw); displayName != "" {
			return displayName
		}
	}
	if raw, ok := s.store.GetRaw(nodeDisplayNameKey); ok {
		if displayName := normalizeNodeDisplayName(raw); displayName != "" {
			return displayName
		}
	}
	profile := s.store.CurrentProfile()
	if displayName := strings.TrimSpace(s.store.GetString(profile, mcpDisplayNameKey, "")); displayName != "" {
		return displayName
	}
	return strings.TrimSpace(s.store.GetString(profile, nodeDisplayNameKey, ""))
}

func normalizeNodeDisplayName(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case []byte:
		return strings.TrimSpace(string(v))
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}

func (s *AuthService) send(_ context.Context, sourceID, targetID uint32, payload []byte) error {
	if s.session == nil {
		return errors.New("session service not initialized")
	}
	return s.session.SendCommand(auth.SubProtoAuth, sourceID, targetID, payload)
}

func (s *AuthService) sendAndAwait(ctx context.Context, sourceID, targetID uint32, payload []byte, reqAction, respAction string) (auth.RespData, error) {
	var data auth.RespData
	if err := s.sendAndAwaitInto(ctx, sourceID, targetID, payload, reqAction, respAction, &data); err != nil {
		return auth.RespData{}, err
	}
	return data, nil
}

func (s *AuthService) sendAndAwaitInto(ctx context.Context, sourceID, targetID uint32, payload []byte, reqAction, respAction string, out any) error {
	if s.session == nil {
		return errors.New("session service not initialized")
	}
	trimmedAction := strings.TrimSpace(reqAction)
	resp, err := s.session.SendCommandAndAwait(ctx, auth.SubProtoAuth, sourceID, targetID, payload, respAction)
	if err != nil {
		if s.logs != nil {
			s.logs.Appendf("error", "auth %s await failed: %v", trimmedAction, err)
		}
		return fmt.Errorf("auth %s: %w", trimmedAction, toUIError(err))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(resp.Message.Data, out); err != nil {
		if s.logs != nil {
			s.logs.Appendf("error", "auth %s decode failed: %v", trimmedAction, err)
		}
		return err
	}
	code, msg := extractAuthCodeMsg(out)
	if code != 1 {
		msg = strings.TrimSpace(msg)
		if msg != "" {
			if s.logs != nil {
				s.logs.Appendf("warn", "auth %s failed (code=%d msg=%q)", trimmedAction, code, msg)
			}
			return fmt.Errorf("%s (code=%d)", msg, code)
		}
		if s.logs != nil {
			s.logs.Appendf("warn", "auth %s failed (code=%d)", trimmedAction, code)
		}
		return fmt.Errorf("auth failed (code=%d)", code)
	}
	return nil
}

func extractAuthCodeMsg(v any) (int, string) {
	switch t := v.(type) {
	case *auth.RespData:
		if t == nil {
			return 0, ""
		}
		return t.Code, t.Msg
	case *ListRolesResp:
		if t == nil {
			return 0, ""
		}
		return t.Code, t.Msg
	case *ListPendingRegistersResp:
		if t == nil {
			return 0, ""
		}
		return t.Code, t.Msg
	case *ApproveRegisterResp:
		if t == nil {
			return 0, ""
		}
		return t.Code, t.Msg
	case *RejectRegisterResp:
		if t == nil {
			return 0, ""
		}
		return t.Code, t.Msg
	case *IssueRegisterPermitResp:
		if t == nil {
			return 0, ""
		}
		return t.Code, t.Msg
	case *RevokeRegisterPermitResp:
		if t == nil {
			return 0, ""
		}
		return t.Code, t.Msg
	default:
		return 0, ""
	}
}

func toUIError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return errors.New("request timed out")
	}
	if errors.Is(err, context.Canceled) {
		return errors.New("request canceled")
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(msg, "timed out"):
		return errors.New("request timed out")
	case strings.Contains(msg, "session not initialized"):
		return errors.New("not connected")
	case strings.Contains(msg, "aborted by the software in your host machine"):
		return errors.New("connection aborted")
	case strings.Contains(msg, "connection") && strings.Contains(msg, "closed"):
		return errors.New("connection closed")
	default:
		return err
	}
}
