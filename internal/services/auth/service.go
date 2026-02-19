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
)

const defaultNodeKeysPath = "config/node_keys.json"

const defaultAuthTimeout = 8 * time.Second

type AuthService struct {
	session  *sessionsvc.SessionService
	logs     *logs.LogService
	keyMu    sync.Mutex
	nodePriv *ecdsa.PrivateKey
	nodePub  string
	keysPath string
}

func New(session *sessionsvc.SessionService, logsSvc *logs.LogService) *AuthService {
	return &AuthService{session: session, logs: logsSvc, keysPath: defaultNodeKeysPath}
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
	payload, err := transport.EncodeMessage(auth.ActionRegister, auth.RegisterData{
		DeviceID: deviceID,
		PubKey:   pub,
		NodePub:  pub,
	})
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
	payload, err := transport.EncodeMessage(auth.ActionLogin, login)
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

func (s *AuthService) send(_ context.Context, sourceID, targetID uint32, payload []byte) error {
	if s.session == nil {
		return errors.New("session service not initialized")
	}
	return s.session.SendCommand(auth.SubProtoAuth, sourceID, targetID, payload)
}

func (s *AuthService) sendAndAwait(ctx context.Context, sourceID, targetID uint32, payload []byte, reqAction, respAction string) (auth.RespData, error) {
	if s.session == nil {
		return auth.RespData{}, errors.New("session service not initialized")
	}
	trimmedAction := strings.TrimSpace(reqAction)
	resp, err := s.session.SendCommandAndAwait(ctx, auth.SubProtoAuth, sourceID, targetID, payload, respAction)
	if err != nil {
		if s.logs != nil {
			s.logs.Appendf("error", "auth %s await failed: %v", trimmedAction, err)
		}
		return auth.RespData{}, fmt.Errorf("auth %s: %w", trimmedAction, toUIError(err))
	}
	var data auth.RespData
	if err := json.Unmarshal(resp.Message.Data, &data); err != nil {
		if s.logs != nil {
			s.logs.Appendf("error", "auth %s decode failed: %v", trimmedAction, err)
		}
		return auth.RespData{}, err
	}
	if data.Code != 1 {
		msg := strings.TrimSpace(data.Msg)
		if msg != "" {
			if s.logs != nil {
				s.logs.Appendf("warn", "auth %s failed (code=%d msg=%q)", trimmedAction, data.Code, msg)
			}
			return auth.RespData{}, fmt.Errorf("%s (code=%d)", msg, data.Code)
		}
		if s.logs != nil {
			s.logs.Appendf("warn", "auth %s failed (code=%d)", trimmedAction, data.Code)
		}
		return auth.RespData{}, fmt.Errorf("auth failed (code=%d)", data.Code)
	}
	return data, nil
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
	case strings.Contains(msg, "session not initialized"):
		return errors.New("not connected")
	case strings.Contains(msg, "connection") && strings.Contains(msg, "closed"):
		return errors.New("connection closed")
	default:
		return err
	}
}
