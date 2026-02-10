package session

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	core "github.com/yttydcs/myflowhub-core"
	"github.com/yttydcs/myflowhub-core/header"
)

var traceSeq atomic.Uint32
var traceSeqInit sync.Once

func nextTraceID() uint32 {
	traceSeqInit.Do(func() {
		var seed [4]byte
		if _, err := rand.Read(seed[:]); err != nil {
			traceSeq.Store(uint32(time.Now().UnixNano()))
			return
		}
		traceSeq.Store(binary.BigEndian.Uint32(seed[:]))
	})
	v := traceSeq.Add(1)
	if v == 0 {
		v = traceSeq.Add(1)
	}
	return v
}

type Session struct {
	mu      sync.Mutex
	conn    net.Conn
	codec   header.HeaderTcpCodec
	baseCtx context.Context
	ctx     context.Context
	cancel  context.CancelFunc
	onFrame func(core.IHeader, []byte)
	onError func(error)
}

func New(ctx context.Context, onFrame func(core.IHeader, []byte), onError func(error)) *Session {
	if ctx == nil {
		ctx = context.Background()
	}
	cctx, cancel := context.WithCancel(ctx)
	return &Session{baseCtx: ctx, ctx: cctx, cancel: cancel, codec: header.HeaderTcpCodec{}, onFrame: onFrame, onError: onError}
}

func (s *Session) Connect(addr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn != nil {
		return errors.New("已经连接")
	}
	if s.ctx == nil || s.ctx.Err() != nil {
		base := s.baseCtx
		if base == nil {
			base = context.Background()
		}
		s.ctx, s.cancel = context.WithCancel(base)
	}
	dialer := net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(s.ctx, "tcp", addr)
	if err != nil {
		return err
	}
	s.conn = conn
	go s.readLoop()
	return nil
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
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn != nil {
		_ = s.conn.Close()
		s.conn = nil
	}
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *Session) Send(hdr core.IHeader, payload []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn == nil {
		return fmt.Errorf("尚未连接")
	}
	if hdr == nil {
		return errors.New("header is required")
	}
	if hdr.GetHopLimit() == 0 {
		hdr.WithHopLimit(header.DefaultHopLimit)
	}
	if hdr.GetTraceID() == 0 {
		hdr.WithTraceID(nextTraceID())
	}
	frame, err := s.codec.Encode(hdr, payload)
	if err != nil {
		return err
	}
	_, err = s.conn.Write(frame)
	return err
}

func (s *Session) readLoop() {
	reader := bufio.NewReader(s.conn)
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}
		hdr, payload, err := s.codec.Decode(reader)
		if err != nil {
			if s.onError != nil {
				s.onError(err)
			}
			return
		}
		if s.onFrame != nil {
			s.onFrame(hdr, payload)
		}
	}
}
