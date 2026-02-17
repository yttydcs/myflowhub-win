package session

import (
	"context"
	"errors"
	"time"

	core "github.com/yttydcs/myflowhub-core"
	"github.com/yttydcs/myflowhub-core/header"
	sdkawait "github.com/yttydcs/myflowhub-sdk/await"
)

var ErrSessionNotInitialized = errors.New("session not initialized")

type Session struct {
	client *sdkawait.Client
}

func New(ctx context.Context, onFrame func(core.IHeader, []byte), onError func(error)) *Session {
	c := sdkawait.NewClient(ctx, nil, onError)
	if onFrame != nil {
		c.SetOnFrame(onFrame)
	}
	return &Session{client: c}
}

func (s *Session) Connect(addr string) error {
	if s == nil || s.client == nil {
		return ErrSessionNotInitialized
	}
	return s.client.Connect(addr)
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
	if s == nil || s.client == nil {
		return
	}
	s.client.Close()
}

func (s *Session) Send(hdr core.IHeader, payload []byte) error {
	if s == nil || s.client == nil {
		return ErrSessionNotInitialized
	}
	// 兼容：保留 Win 侧显式补齐逻辑，但底层实际由 SDK 统一处理。
	if hdr != nil {
		if hdr.GetHopLimit() == 0 {
			hdr.WithHopLimit(header.DefaultHopLimit)
		}
	}
	return s.client.Send(hdr, payload)
}

func (s *Session) SendAndAwait(ctx context.Context, hdr core.IHeader, payload []byte, expectAction string) (sdkawait.Response, error) {
	if s == nil || s.client == nil {
		return sdkawait.Response{}, ErrSessionNotInitialized
	}
	return s.client.SendAndAwait(ctx, hdr, payload, expectAction)
}
