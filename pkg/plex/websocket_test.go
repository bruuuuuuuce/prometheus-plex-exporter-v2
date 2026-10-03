package plex_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gorilla/websocket"
	plexclient "github.com/jrudio/go-plex-client"
)

func TestTimelineSectionIDAcceptsNumberAndString(t *testing.T) {
	for _, tc := range []struct {
		name      string
		sectionID string
	}{
		{name: "number", sectionID: `12`},
		{name: "string", sectionID: `"12"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload := `{"NotificationContainer":{"type":"playing","TimelineEntry":[{"sectionID":` + tc.sectionID + `,"title":"Library"}],"PlaySessionStateNotification":[{"sessionKey":"abc","state":"playing"}]}}`
			var notification plexclient.WebsocketNotification
			if err := json.Unmarshal([]byte(payload), &notification); err != nil {
				t.Fatalf("decode notification: %v", err)
			}
			if got := notification.TimelineEntry[0].SectionID; got != 12 {
				t.Errorf("sectionID = %d, want 12", got)
			}
			if got := notification.TimelineEntry[0].Title; got != "Library" {
				t.Errorf("title = %q, want Library", got)
			}
			if got := notification.PlaySessionStateNotification[0].SessionKey; got != "abc" {
				t.Errorf("playing session key = %q, want abc", got)
			}
		})
	}
}

func TestObservedWebsocketEventsAreIgnoredAndUnknownEventsRemainDebuggable(t *testing.T) {
	playing := make(chan struct{}, 1)
	var logOutput bytes.Buffer
	logger := level.NewFilter(kitlog.NewLogfmtLogger(&logOutput), level.AllowDebug())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Upgrade(w, r, nil, 1024, 1024)
		if err != nil {
			t.Errorf("upgrade websocket: %v", err)
			return
		}
		defer conn.Close()
		for _, payload := range []string{
			`{"NotificationContainer":{"type":"progress"}}`,
			`{"NotificationContainer":{"type":"status"}}`,
			`{"NotificationContainer":{"type":"provider.content.change"}}`,
			`{"NotificationContainer":{"type":"playing "}}`,
			`{"NotificationContainer":{"type":"future.event"}}`,
		} {
			if err := conn.WriteMessage(websocket.TextMessage, []byte(payload)); err != nil {
				t.Errorf("write websocket notification: %v", err)
				return
			}
		}
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
	}))
	defer server.Close()

	client := &plexclient.Plex{URL: server.URL}
	interrupt := make(chan os.Signal)
	finished := make(chan struct{}, 1)
	events := plexclient.NewNotificationEvents()
	events.OnPlaying(func(plexclient.NotificationContainer) { playing <- struct{}{} })
	client.SubscribeToNotificationsWithLogger(events, interrupt, func(error) { finished <- struct{}{} }, logger)

	select {
	case <-playing:
	case <-time.After(2 * time.Second):
		t.Fatal("playing event with trailing whitespace was not dispatched")
	}
	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		t.Fatal("websocket did not finish after test notifications")
	}
	if got := logOutput.String(); strings.Contains(got, "progress") || strings.Contains(got, "provider.content.change") || strings.Contains(got, "event=status") {
		t.Errorf("recognized events were logged as unknown: %s", got)
	}
	if got := logOutput.String(); !strings.Contains(got, "event=future.event") {
		t.Errorf("unexpected event was not logged at debug level: %s", got)
	}
	close(interrupt)
}

func TestInvalidTimelineSectionIDPreservesPlayingNotification(t *testing.T) {
	var notification plexclient.WebsocketNotification
	payload := `{"NotificationContainer":{"type":"playing","TimelineEntry":[{"sectionID":"not-a-number"}],"PlaySessionStateNotification":[{"sessionKey":"abc","state":"playing"}]}}`
	if err := json.Unmarshal([]byte(payload), &notification); err != nil {
		t.Fatalf("decode notification: %v", err)
	}
	if got := notification.TimelineEntry[0].SectionID; got != 0 {
		t.Errorf("invalid sectionID = %d, want 0", got)
	}
	if got := notification.PlaySessionStateNotification[0].SessionKey; got != "abc" {
		t.Errorf("playing session key = %q, want abc", got)
	}
}
