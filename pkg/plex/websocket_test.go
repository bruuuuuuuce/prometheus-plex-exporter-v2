package plex

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	plexclient "github.com/bruuuuuuuce/go-plex-client/v2"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gorilla/websocket"
)

func TestWebsocketListenerUsesTokenWithoutLoggingPayload(t *testing.T) {
	const privateTitle = "private media title"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/:/websockets/notifications" {
			t.Errorf("websocket path = %q, want /:/websockets/notifications", got)
		}
		if got := r.Header.Get("X-Plex-Token"); got != "test-token" {
			t.Errorf("X-Plex-Token = %q, want test-token", got)
		}

		conn, err := websocket.Upgrade(w, r, nil, 1024, 1024)
		if err != nil {
			t.Errorf("upgrade websocket: %v", err)
			return
		}
		defer conn.Close()
		if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"NotificationContainer":{"type":"future.event","title":"`+privateTitle+`"}}`)); err != nil {
			t.Errorf("write websocket notification: %v", err)
			return
		}
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
	}))
	defer server.Close()

	var logOutput bytes.Buffer
	logger := level.NewFilter(kitlog.NewLogfmtLogger(&logOutput), level.AllowDebug())
	listener := plexListener{
		server: &Server{},
		conn:   &plexclient.Plex{URL: server.URL, Token: "test-token"},
		log:    logger,
	}
	if err := listener.listen(context.Background()); err != nil {
		t.Fatalf("listen: %v", err)
	}
	if got := logOutput.String(); strings.Contains(got, privateTitle) {
		t.Errorf("private payload was logged: %s", got)
	}
}

func TestPlayingNotificationIgnoresTimelineSectionIDEncoding(t *testing.T) {
	for _, tc := range []struct {
		name      string
		sectionID string
	}{
		{name: "number", sectionID: `12`},
		{name: "string", sectionID: `"12"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload := `{"NotificationContainer":{"type":"playing","TimelineEntry":[{"sectionID":` + tc.sectionID + `,"title":"Library"}],"PlaySessionStateNotification":[{"sessionKey":"abc","state":"playing"}]}}`
			var got plexclient.NotificationContainer
			if err := handleWebsocketNotification([]byte(payload), func(notification plexclient.NotificationContainer) {
				got = notification
			}, kitlog.NewNopLogger()); err != nil {
				t.Fatalf("handle notification: %v", err)
			}
			if got := got.PlaySessionStateNotification[0].SessionKey; got != "abc" {
				t.Errorf("playing session key = %q, want abc", got)
			}
		})
	}
}

func TestObservedWebsocketEventsAreIgnoredAndUnknownEventsRemainDebuggable(t *testing.T) {
	playing := 0
	var logOutput bytes.Buffer
	logger := level.NewFilter(kitlog.NewLogfmtLogger(&logOutput), level.AllowDebug())
	for _, payload := range []string{
		`{"NotificationContainer":{"type":"progress"}}`,
		`{"NotificationContainer":{"type":"status"}}`,
		`{"NotificationContainer":{"type":"provider.content.change"}}`,
		`{"NotificationContainer":{"type":"playing "}}`,
		`{"NotificationContainer":{"type":"future.event"}}`,
	} {
		if err := handleWebsocketNotification([]byte(payload), func(plexclient.NotificationContainer) {
			playing++
		}, logger); err != nil {
			t.Fatalf("handle notification: %v", err)
		}
	}
	if playing != 1 {
		t.Fatalf("playing callbacks = %d, want 1", playing)
	}
	if got := logOutput.String(); strings.Contains(got, "progress") || strings.Contains(got, "provider.content.change") || strings.Contains(got, "event=status") {
		t.Errorf("recognized events were logged as unknown: %s", got)
	}
	if got := logOutput.String(); !strings.Contains(got, "event=future.event") {
		t.Errorf("unexpected event was not logged at debug level: %s", got)
	}
}

func TestInvalidTimelineSectionIDPreservesPlayingNotification(t *testing.T) {
	payload := `{"NotificationContainer":{"type":"playing","TimelineEntry":[{"sectionID":"not-a-number"}],"PlaySessionStateNotification":[{"sessionKey":"abc","state":"playing"}]}}`
	var got plexclient.NotificationContainer
	if err := handleWebsocketNotification([]byte(payload), func(notification plexclient.NotificationContainer) {
		got = notification
	}, kitlog.NewNopLogger()); err != nil {
		t.Fatalf("handle notification: %v", err)
	}
	if got := got.PlaySessionStateNotification[0].SessionKey; got != "abc" {
		t.Errorf("playing session key = %q, want abc", got)
	}
}
