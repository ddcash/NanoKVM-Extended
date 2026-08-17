package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"NanoKVM-Server/proto"
)

// Sessions exist so a login can be revoked. The token itself is a JWT and says
// nothing about whether it is still wanted, so before this a stolen one stayed
// valid until it expired and there was no way to see it had been taken.
//
// Each token now carries a session id, and a token whose session is gone is
// refused. The store is held in memory and written to disk on change: the check
// runs on every authenticated request, so it must not touch the disk, and it
// never runs on the video or HID paths.
const (
	SessionFile = "/etc/kvm/sessions.json"

	// Bounds the file, and an attacker cannot grow it without valid logins.
	maxSessions = 50

	// How stale lastSeen may get before it is written. Every request would
	// otherwise mean a disk write on an SD card.
	lastSeenInterval = 5 * time.Minute
)

type Session struct {
	Id        string    `json:"id"`
	Username  string    `json:"username"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"userAgent"`
	CreatedAt time.Time `json:"createdAt"`
	LastSeen  time.Time `json:"lastSeen"`
}

var (
	sessionMu    sync.RWMutex
	sessions     map[string]*Session
	sessionsPath = SessionFile
	sessionsRead bool
)

// LoadSessions reads the store at startup. A store that cannot be read starts
// empty, which logs everyone out rather than failing open.
func LoadSessions() {
	sessionMu.Lock()
	defer sessionMu.Unlock()

	sessions = make(map[string]*Session)
	sessionsRead = true

	data, err := os.ReadFile(sessionsPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Errorf("failed to read sessions, starting empty: %s", err)
		}
		return
	}

	var stored []*Session
	if err := json.Unmarshal(data, &stored); err != nil {
		log.Errorf("failed to parse sessions, starting empty: %s", err)
		return
	}

	for _, session := range stored {
		if session.Id == "" {
			continue
		}
		sessions[session.Id] = session
	}

	log.Debugf("loaded %d sessions", len(sessions))
}

// CreateSession records a login and returns the id to embed in the token.
func CreateSession(c *gin.Context, username string) string {
	sessionMu.Lock()
	defer sessionMu.Unlock()

	ensureLoadedLocked()

	// Oldest first, so a flood of logins cannot push out the newest.
	if len(sessions) >= maxSessions {
		evictOldestLocked()
	}

	now := time.Now()
	session := &Session{
		Id:        uuid.NewString(),
		Username:  username,
		CreatedAt: now,
		LastSeen:  now,
	}
	if c != nil {
		session.IP = GetClientIP(c)
		session.UserAgent = c.GetHeader("User-Agent")
	}

	sessions[session.Id] = session
	persistLocked()

	return session.Id
}

// TouchSession reports whether the session is still valid, and refreshes
// lastSeen. Called on every authenticated request, so the common path takes a
// read lock and writes nothing.
func TouchSession(id string) bool {
	if id == "" {
		return false
	}

	sessionMu.RLock()
	if !sessionsRead {
		sessionMu.RUnlock()
		LoadSessions()
		sessionMu.RLock()
	}
	session, ok := sessions[id]
	stale := ok && time.Since(session.LastSeen) > lastSeenInterval
	sessionMu.RUnlock()

	if !ok {
		return false
	}
	if !stale {
		return true
	}

	sessionMu.Lock()
	if current, still := sessions[id]; still {
		current.LastSeen = time.Now()
		persistLocked()
	}
	sessionMu.Unlock()

	return true
}

func RemoveSession(id string) {
	if id == "" {
		return
	}

	sessionMu.Lock()
	defer sessionMu.Unlock()

	ensureLoadedLocked()

	if _, ok := sessions[id]; ok {
		delete(sessions, id)
		persistLocked()
	}
}

func (s *Service) GetSessions(c *gin.Context) {
	var rsp proto.Response

	current, _ := c.Get("sessionId")
	currentId, _ := current.(string)

	sessionMu.RLock()
	list := make([]proto.SessionInfo, 0, len(sessions))
	for _, session := range sessions {
		list = append(list, proto.SessionInfo{
			Id:        session.Id,
			Username:  session.Username,
			IP:        session.IP,
			UserAgent: session.UserAgent,
			CreatedAt: session.CreatedAt.Unix(),
			LastSeen:  session.LastSeen.Unix(),
			Current:   session.Id == currentId,
		})
	}
	sessionMu.RUnlock()

	// Newest first: the one someone is looking for is usually recent.
	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt > list[j].CreatedAt
	})

	rsp.OkRspWithData(c, &proto.GetSessionsRsp{Sessions: list})
}

func (s *Service) RevokeSession(c *gin.Context) {
	var req proto.RevokeSessionReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	current, _ := c.Get("sessionId")
	currentId, _ := current.(string)

	if req.All {
		revokeAllExcept(currentId)
		Audit(c, "session_revoke_all", nil)
		rsp.OkRsp(c)
		return
	}

	if req.Id == "" {
		rsp.ErrRsp(c, -2, "invalid arguments")
		return
	}
	// Revoking your own session would just be a confusing logout.
	if req.Id == currentId {
		rsp.ErrRsp(c, -3, "use logout to end the current session")
		return
	}

	RemoveSession(req.Id)
	Audit(c, "session_revoke", log.Fields{"session": req.Id})

	rsp.OkRsp(c)
}

func revokeAllExcept(keep string) {
	sessionMu.Lock()
	defer sessionMu.Unlock()

	ensureLoadedLocked()

	for id := range sessions {
		if id != keep {
			delete(sessions, id)
		}
	}
	persistLocked()
}

func ensureLoadedLocked() {
	if sessions == nil {
		sessions = make(map[string]*Session)
	}
}

func evictOldestLocked() {
	var oldestId string
	var oldest time.Time

	for id, session := range sessions {
		if oldestId == "" || session.LastSeen.Before(oldest) {
			oldestId, oldest = id, session.LastSeen
		}
	}

	if oldestId != "" {
		delete(sessions, oldestId)
	}
}

// persistLocked writes the store. The caller holds the write lock.
func persistLocked() {
	list := make([]*Session, 0, len(sessions))
	for _, session := range sessions {
		list = append(list, session)
	}

	data, err := json.Marshal(list)
	if err != nil {
		log.Errorf("failed to marshal sessions: %s", err)
		return
	}

	if err := writeSessionFile(data); err != nil {
		log.Errorf("failed to write sessions: %s", err)
	}
}

func writeSessionFile(data []byte) error {
	dir := filepath.Dir(sessionsPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, ".sessions.json.*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	// 0600: knowing which sessions exist is not something to hand out.
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmpPath, sessionsPath)
}
