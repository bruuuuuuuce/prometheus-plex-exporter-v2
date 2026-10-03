package plex_test

import (
	"encoding/json"
	"testing"

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
