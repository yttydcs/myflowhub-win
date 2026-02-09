package auth

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/yttydcs/myflowhub-server/protocol/auth"
	"github.com/yttydcs/myflowhub-win/internal/services/logs"
	sessionsvc "github.com/yttydcs/myflowhub-win/internal/services/session"
	"github.com/yttydcs/myflowhub-win/internal/services/transport"
)

const defaultNodeKeysPath = "config/node_keys.json"

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

func (s *AuthService) Register(ctx context.Context, sourceID, targetID uint32, deviceID string) error {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return errors.New("device_id is required")
	}
	pub, err := s.EnsureKeys()
	if err != nil {
		return err
	}
	payload, err := transport.EncodeMessage(auth.ActionRegister, auth.RegisterData{
		DeviceID: deviceID,
		PubKey:   pub,
		NodePub:  pub,
	})
	if err != nil {
		return err
	}
	if err := s.send(ctx, sourceID, targetID, payload); err != nil {
		return err
	}
	s.logs.Appendf("info", "auth register sent device=%s", deviceID)
	return nil
}

func (s *AuthService) RegisterSimple(sourceID, targetID uint32, deviceID string) error {
	return s.Register(context.Background(), sourceID, targetID, deviceID)
}

func (s *AuthService) Login(ctx context.Context, sourceID, targetID uint32, deviceID string, nodeID uint32) error {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return errors.New("device_id is required")
	}
	if nodeID == 0 {
		return errors.New("node_id is required")
	}
	login, err := s.SignLogin(deviceID, nodeID)
	if err != nil {
		return err
	}
	payload, err := transport.EncodeMessage(auth.ActionLogin, login)
	if err != nil {
		return err
	}
	if err := s.send(ctx, sourceID, targetID, payload); err != nil {
		return err
	}
	s.logs.Appendf("info", "auth login sent device=%s node=%d", deviceID, nodeID)
	return nil
}

func (s *AuthService) LoginSimple(sourceID, targetID uint32, deviceID string, nodeID uint32) error {
	return s.Login(context.Background(), sourceID, targetID, deviceID, nodeID)
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
