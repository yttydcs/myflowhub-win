package file

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/yttydcs/myflowhub-core/eventbus"
	protocol "github.com/yttydcs/myflowhub-server/protocol/file"
	"github.com/yttydcs/myflowhub-win/internal/services/logs"
	sessionsvc "github.com/yttydcs/myflowhub-win/internal/services/session"
	"github.com/yttydcs/myflowhub-win/internal/services/transport"
	"github.com/yttydcs/myflowhub-win/internal/storage"
)

type FileService struct {
	session *sessionsvc.SessionService
	logs    *logs.LogService
	store   *storage.Store
	bus     eventbus.IBus

	mu        sync.RWMutex
	localNode uint32
	hubID     uint32

	state      *fileState
	busTokens  []busToken
}

func New(session *sessionsvc.SessionService, logsSvc *logs.LogService, store *storage.Store, bus eventbus.IBus) *FileService {
	svc := &FileService{
		session: session,
		logs:    logsSvc,
		store:   store,
		bus:     bus,
		state:   newFileState(),
	}
	svc.bindBus()
	return svc
}

func (s *FileService) Close() {
	s.unbindBus()
	s.stopJanitor()
}

func (s *FileService) SetIdentity(nodeID, hubID uint32) {
	s.mu.Lock()
	s.localNode = nodeID
	s.hubID = hubID
	s.mu.Unlock()
}

func (s *FileService) List(ctx context.Context, sourceID, hubID, targetID uint32, dir string, recursive bool) error {
	req := protocol.ReadReq{Op: protocol.OpList, Target: targetID, Dir: strings.TrimSpace(dir), Recursive: recursive}
	return s.Read(ctx, sourceID, hubID, req)
}

func (s *FileService) ListSimple(sourceID, hubID, targetID uint32, dir string, recursive bool) error {
	return s.List(context.Background(), sourceID, hubID, targetID, dir, recursive)
}

func (s *FileService) ReadText(ctx context.Context, sourceID, hubID, targetID uint32, dir, name string, maxBytes uint32) error {
	req := protocol.ReadReq{Op: protocol.OpReadText, Target: targetID, Dir: strings.TrimSpace(dir), Name: strings.TrimSpace(name), MaxBytes: maxBytes}
	return s.Read(ctx, sourceID, hubID, req)
}

func (s *FileService) ReadTextSimple(sourceID, hubID, targetID uint32, dir, name string, maxBytes uint32) error {
	return s.ReadText(context.Background(), sourceID, hubID, targetID, dir, name, maxBytes)
}

func (s *FileService) Pull(ctx context.Context, sourceID, hubID uint32, req protocol.ReadReq) error {
	req.Op = protocol.OpPull
	return s.Read(ctx, sourceID, hubID, req)
}

func (s *FileService) PullSimple(sourceID, hubID uint32, req protocol.ReadReq) error {
	return s.Pull(context.Background(), sourceID, hubID, req)
}

func (s *FileService) Offer(ctx context.Context, sourceID, hubID uint32, req protocol.WriteReq) error {
	req.Op = protocol.OpOffer
	return s.Write(ctx, sourceID, hubID, req)
}

func (s *FileService) OfferSimple(sourceID, hubID uint32, req protocol.WriteReq) error {
	return s.Offer(context.Background(), sourceID, hubID, req)
}

func (s *FileService) Read(ctx context.Context, sourceID, hubID uint32, req protocol.ReadReq) error {
	if strings.TrimSpace(req.Op) == "" {
		return errors.New("op is required")
	}
	if req.Target == 0 {
		return errors.New("target is required")
	}
	if req.Target == sourceID {
		return s.handleLocalRead(req)
	}
	if hubID == 0 {
		return errors.New("hub_id is required")
	}
	payload, err := transport.EncodeMessage(protocol.ActionRead, req)
	if err != nil {
		return err
	}
	return s.sendCtrl(ctx, sourceID, hubID, payload, "read", req.Op)
}

func (s *FileService) Write(ctx context.Context, sourceID, hubID uint32, req protocol.WriteReq) error {
	if strings.TrimSpace(req.Op) == "" {
		return errors.New("op is required")
	}
	if strings.TrimSpace(req.Name) == "" {
		return errors.New("name is required")
	}
	if hubID == 0 {
		return errors.New("hub_id is required")
	}
	payload, err := transport.EncodeMessage(protocol.ActionWrite, req)
	if err != nil {
		return err
	}
	return s.sendCtrl(ctx, sourceID, hubID, payload, "write", req.Op)
}

func (s *FileService) Send(ctx context.Context, sourceID, hubID uint32, action string, data any) error {
	payload, err := transport.EncodeMessage(action, data)
	if err != nil {
		return err
	}
	return s.sendCtrl(ctx, sourceID, hubID, payload, action, "")
}

func (s *FileService) sendCtrl(_ context.Context, sourceID, targetID uint32, payload []byte, action, op string) error {
	if s.session == nil {
		return errors.New("session service not initialized")
	}
	ctrlPayload := make([]byte, 1+len(payload))
	ctrlPayload[0] = protocol.KindCtrl
	copy(ctrlPayload[1:], payload)
	if err := s.session.SendCommand(protocol.SubProtoFile, sourceID, targetID, ctrlPayload); err != nil {
		return err
	}
	if s.logs != nil {
		if op != "" {
			s.logs.Appendf("info", "file %s sent op=%s", action, op)
		} else {
			s.logs.Appendf("info", "file %s sent", action)
		}
	}
	return nil
}
