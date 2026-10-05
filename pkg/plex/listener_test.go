package plex_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gorilla/websocket"

	plexexporter "github.com/grafana/plexporter/pkg/plex"
)

func TestListenerHandlesSessionsMissingFromSnapshot(t *testing.T) {
	var sessionRequests atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/status/sessions", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch sessionRequests.Add(1) {
		case 1:
			_, _ = w.Write([]byte(`{"MediaContainer":{"size":1,"Metadata":[{"sessionKey":"176","User":{"id":"1","title":"test-user"},"Player":{"device":"test-device","product":"test-player"},"Media":[{"bitrate":1000,"videoResolution":"1080","Part":[{"decision":"directplay"}]}]}]}}`))
		case 3:
			_, _ = w.Write([]byte(`{"MediaContainer":{"size":1,"Metadata":[{"sessionKey":"177","User":{"id":"1","title":"test-user"},"Player":{"device":"test-device","product":"test-player"},"Media":[{"bitrate":1000,"videoResolution":"1080","Part":[{"decision":"directplay"}]}]}]}}`))
		default:
			_, _ = w.Write([]byte(`{"MediaContainer":{"size":0,"Metadata":[]}}`))
		}
	})
	mux.HandleFunc("/library/metadata/3147", metadataHandler("3147"))
	mux.HandleFunc("/library/metadata/3148", metadataHandler("3148"))
	mux.HandleFunc("/:/websockets/notifications", func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Upgrade(w, r, nil, 1024, 1024)
		if err != nil {
			t.Errorf("upgrade websocket: %v", err)
			return
		}
		defer conn.Close()

		payloads := []string{
			`{"NotificationContainer":{"type":"playing","PlaySessionStateNotification":[{"sessionKey":"176","state":"playing","ratingKey":"3147"}]}}`,
			`{"NotificationContainer":{"type":"playing","PlaySessionStateNotification":[{"sessionKey":"176","state":"buffering","ratingKey":"3147"}]}}`,
			`{"NotificationContainer":{"type":"playing","PlaySessionStateNotification":[{"sessionKey":"missing","state":"buffering","ratingKey":"3149"},{"sessionKey":"177","state":"buffering","ratingKey":"3148"}]}}`,
			`{"NotificationContainer":{"type":"playing","PlaySessionStateNotification":[{"sessionKey":"stopped-before-seen","state":"stopped"}]}}`,
			`{"NotificationContainer":{"type":"playing","PlaySessionStateNotification":[{"sessionKey":"stopped-before-seen","state":"buffering","ratingKey":"3150"}]}}`,
		}
		for _, payload := range payloads {
			if err := conn.WriteMessage(websocket.TextMessage, []byte(payload)); err != nil {
				t.Errorf("write websocket notification: %v", err)
				return
			}
		}
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
	})

	fakePlex := httptest.NewServer(mux)
	defer fakePlex.Close()
	serverURL, err := url.Parse(fakePlex.URL)
	if err != nil {
		t.Fatalf("parse fake Plex URL: %v", err)
	}

	var logOutput bytes.Buffer
	logger := level.NewFilter(kitlog.NewLogfmtLogger(&logOutput), level.AllowDebug())
	server := &plexexporter.Server{URL: serverURL, Token: "test-token"}
	if err := server.Listen(context.Background(), logger); err != nil {
		t.Fatalf("Listen() error = %v", err)
	}

	logs := logOutput.String()
	if strings.Contains(logs, "error handling OnPlaying event") {
		t.Errorf("session snapshot race logged as an error: %s", logs)
	}
	if !strings.Contains(logs, `msg="updated cached session missing from Plex session snapshot" sessionKey=176 state=buffering`) {
		t.Errorf("cached buffering session was not updated: %s", logs)
	}
	if !strings.Contains(logs, `msg="Received PlaySessionStateNotification" SessionKey=177`) {
		t.Errorf("notification after unknown session was not processed: %s", logs)
	}
	if !strings.Contains(logs, `msg="ignored session missing from Plex session snapshot" sessionKey=stopped-before-seen state=buffering`) {
		t.Errorf("unknown stopped session was cached: %s", logs)
	}
}

func metadataHandler(ratingKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"MediaContainer":{"size":1,"Metadata":[{"ratingKey":"` + ratingKey + `","title":"test-media","librarySectionID":"1","Media":[{"videoResolution":"1080"}]}]}}`))
	}
}
