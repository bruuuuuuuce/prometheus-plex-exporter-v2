package plex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bruuuuuuuce/go-plex-client/v2"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gorilla/websocket"
)

var (
	ErrAlreadyListening = errors.New("already listening")
)

type plexListener struct {
	server         *Server
	conn           *plex.Plex
	activeSessions *sessions
	log            log.Logger
}

func (s *Server) Listen(ctx context.Context, log log.Logger) error {
	s.mtx.Lock()
	if s.listener != nil {
		s.mtx.Unlock()
		return ErrAlreadyListening
	}

	conn, err := plex.New(s.URL.String(), s.Token)
	if err != nil {
		s.mtx.Unlock()
		return fmt.Errorf("failed to connect to %s: %w", s.URL.String(), err)
	}

	s.listener = &plexListener{
		server:         s,
		conn:           conn,
		activeSessions: NewSessions(ctx, s),
		log:            log,
	}

	s.mtx.Unlock()

	return s.listener.listen(ctx)
}

var ignoredNotificationTypes = map[string]struct{}{
	"activity":                  {},
	"backgroundProcessingQueue": {},
	"preference":                {},
	"progress":                  {},
	"provider.content.change":   {},
	"reachability":              {},
	"status":                    {},
	"transcode.end":             {},
	"transcodeSession.end":      {},
	"transcodeSession.start":    {},
	"transcodeSession.update":   {},
	"update.statechange":        {},
}

type websocketNotification struct {
	NotificationContainer struct {
		PlaySessionStateNotification []plex.PlaySessionStateNotification `json:"PlaySessionStateNotification"`
		Type                         string                              `json:"type"`
	} `json:"NotificationContainer"`
}

func (l *plexListener) listen(ctx context.Context) error {
	plexURL, err := url.Parse(l.conn.URL)
	if err != nil {
		return err
	}

	scheme := "ws"
	if plexURL.Scheme == "https" {
		scheme = "wss"
	}
	websocketURL := url.URL{Scheme: scheme, Host: plexURL.Host, Path: "/:/websockets/notifications"}
	header := http.Header{"X-Plex-Token": []string{l.conn.Token}}
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, websocketURL.String(), header)
	if err != nil {
		return err
	}
	defer conn.Close()

	level.Info(l.log).Log("msg", "Successfully connected", "machineID", l.server.ID, "server", l.server.Name)

	done := make(chan error, 1)
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				done <- err
				return
			}

			if err := handleWebsocketNotification(message, l.onPlayingHandler, l.log); err != nil {
				level.Warn(l.log).Log("msg", "failed to decode Plex websocket notification", "err", err)
			}
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			level.Debug(l.log).Log("msg", "closing Plex websocket connection")
			if err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
				return err
			}
			select {
			case err := <-done:
				if isNormalWebsocketClose(err) {
					return nil
				}
				return err
			case <-time.After(time.Second):
				level.Debug(l.log).Log("msg", "Plex websocket close timed out; forcing connection closed")
				return nil
			}
		case err := <-done:
			if isNormalWebsocketClose(err) {
				return nil
			}
			return err
		case now := <-ticker.C:
			if err := conn.WriteMessage(websocket.TextMessage, []byte(now.String())); err != nil {
				return err
			}
		}
	}
}

func handleWebsocketNotification(message []byte, onPlaying func(plex.NotificationContainer), log log.Logger) error {
	var notification websocketNotification
	if err := json.Unmarshal(message, &notification); err != nil {
		return err
	}

	eventType := strings.TrimSpace(notification.NotificationContainer.Type)
	switch eventType {
	case "playing":
		onPlaying(plex.NotificationContainer{
			PlaySessionStateNotification: notification.NotificationContainer.PlaySessionStateNotification,
			Type:                         eventType,
		})
	default:
		if _, ok := ignoredNotificationTypes[eventType]; !ok {
			level.Debug(log).Log("msg", "unknown websocket event name", "event", eventType)
		}
	}

	return nil
}

func isNormalWebsocketClose(err error) bool {
	var closeErr *websocket.CloseError
	return errors.As(err, &closeErr) && closeErr.Code == websocket.CloseNormalClosure
}

func getSessionByID(sessions plex.CurrentSessions, sessionID string) *plex.Metadata {
	for _, session := range sessions.MediaContainer.Metadata {
		if sessionID == session.SessionKey {
			return &session
		}
	}
	return nil
}

func (l *plexListener) onPlayingHandler(c plex.NotificationContainer) {
	err := l.onPlaying(c)
	if err != nil {
		level.Error(l.log).Log("msg", "error handling OnPlaying event", "eventType", c.Type, "err", err)
	}
}

func (l *plexListener) onPlaying(c plex.NotificationContainer) error {
	sessions, err := l.conn.GetSessions()
	if err != nil {
		return fmt.Errorf("error fetching sessions: %w", err)
	}

	for _, n := range c.PlaySessionStateNotification {
		if sessionState(n.State) == stateStopped {
			// When the session is stopped we can't look up the user info or media anymore.
			l.activeSessions.Update(n.SessionKey, sessionState(n.State), nil, nil)
			continue
		}

		session := getSessionByID(sessions, n.SessionKey)
		if session == nil {
			return fmt.Errorf("error getting session with key %s %+v", n.SessionKey, n)
		}

		metadata, err := l.conn.GetMetadata(n.RatingKey)
		if err != nil {
			return fmt.Errorf("error fetching metadata for key %s: %w", n.RatingKey, err)
		}

		level.Info(l.log).Log("msg", "Received PlaySessionStateNotification",
			"SessionKey", n.SessionKey,
			"userName", session.User.Title,
			"userID", session.User.ID,
			"state", n.State,
			"mediaTitle", metadata.MediaContainer.Metadata[0].Title,
			"mediaID", metadata.MediaContainer.Metadata[0].RatingKey,
			"timestamp", time.Duration(time.Millisecond)*time.Duration(n.ViewOffset))

		l.activeSessions.Update(n.SessionKey, sessionState(n.State), session, &metadata.MediaContainer.Metadata[0])
	}

	return nil
}
