package session

import (
	"context"
	"time"

	core "github.com/yttydcs/myflowhub-core"
	"github.com/yttydcs/myflowhub-core/header"
	sdksession "github.com/yttydcs/myflowhub-sdk/session"
)

type Session struct {
	sess *sdksession.Session
}

func New(ctx context.Context, onFrame func(core.IHeader, []byte), onError func(error)) *Session {
	return &Session{sess: sdksession.New(ctx, onFrame, onError)}
}

func (s *Session) Connect(addr string) error {
	if s == nil || s.sess == nil {
		return sdksession.ErrNotConnected
	}
	return s.sess.Connect(addr)
}

func (s *Session) Login(nodeName string) error {
	if nodeName == "" {
		nodeName = "debugclient"
	}
	hdr := (&header.HeaderTcp{}).
		WithMajor(header.MajorCmd).
		WithSubProto(2).
		WithSourceID(1).
		WithTargetID(1).
		WithMsgID(uint32(time.Now().UnixNano()))
	payload := []byte(nodeName)
	return s.Send(hdr, payload)
}

func (s *Session) Close() {
	if s == nil || s.sess == nil {
		return
	}
	s.sess.Close()
}

func (s *Session) Send(hdr core.IHeader, payload []byte) error {
	if s == nil || s.sess == nil {
		return sdksession.ErrNotConnected
	}
	// 兼容：保留 Win 侧显式补齐逻辑，但底层实际由 SDK 统一处理。
	if hdr != nil {
		if hdr.GetHopLimit() == 0 {
			hdr.WithHopLimit(header.DefaultHopLimit)
		}
	}
	return s.sess.Send(hdr, payload)
}
