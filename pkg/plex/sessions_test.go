package plex

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/bruuuuuuuce/go-plex-client/v2"
	"github.com/prometheus/client_golang/prometheus"
)

func TestUpdateStateStopsPlaybackMetricsIncreasing(t *testing.T) {
	for _, newState := range []sessionState{stateBuffering, statePaused} {
		t.Run(string(newState), func(t *testing.T) {
			activeSessions := newTestActiveSessions()

			time.Sleep(10 * time.Millisecond)
			if updated := activeSessions.updateState("176", newState); !updated {
				t.Fatal("updateState() = false, want true")
			}

			first := gatherPlaybackMetrics(t, activeSessions)
			time.Sleep(20 * time.Millisecond)
			second := gatherPlaybackMetrics(t, activeSessions)

			for metricName, firstValue := range first {
				if firstValue <= 0 {
					t.Errorf("%s after %s = %v, want a positive value", metricName, newState, firstValue)
				}
				if secondValue := second[metricName]; secondValue != firstValue {
					t.Errorf("%s increased after %s: first %v, second %v", metricName, newState, firstValue, secondValue)
				}
			}
		})
	}
}

func TestUpdateStateDoesNotReviveStoppedSession(t *testing.T) {
	for _, staleState := range []sessionState{stateBuffering, statePaused, statePlaying} {
		t.Run(string(staleState), func(t *testing.T) {
			activeSessions := newTestActiveSessions()
			if updated := activeSessions.updateState("176", stateStopped); !updated {
				t.Fatal("stopped updateState() = false, want true")
			}
			cached := activeSessions.sessions["176"]
			cached.lastUpdate = time.Now().Add(-sessionTimeout - time.Second)
			activeSessions.sessions["176"] = cached

			if updated := activeSessions.updateState("176", staleState); updated {
				t.Errorf("stale %s updateState() = true, want false", staleState)
			}
			if got := activeSessions.sessions["176"].state; got != stateStopped {
				t.Errorf("state after stale %s = %q, want %q", staleState, got, stateStopped)
			}

			activeSessions.pruneOldSessions()
			if _, ok := activeSessions.sessions["176"]; ok {
				t.Errorf("session remained cached after stale %s notification", staleState)
			}
		})
	}
}

func newTestActiveSessions() *sessions {
	server := &Server{Name: "test-server", ID: "test-server-id"}
	server.libraries = []*Library{{Name: "Movies", ID: "1", Type: "movie"}}
	activeSessions := &sessions{
		sessions: map[string]session{},
		server:   server,
	}
	activeSessions.Update("176", statePlaying, &plex.Metadata{
		Media: []plex.Media{{
			Bitrate:         1000,
			VideoResolution: "1080",
			Part:            []plex.Part{{Decision: "directplay"}},
		}},
		Player: plex.Player{Device: "test-device", Product: "test-player"},
		User:   plex.User{Title: "test-user"},
	}, &plex.Metadata{
		LibrarySectionID: json.Number("1"),
		Media:            []plex.Media{{VideoResolution: "1080"}},
		RatingKey:        "3147",
		Title:            "test-media",
		Type:             "movie",
	})
	return activeSessions
}

func gatherPlaybackMetrics(t *testing.T, collector prometheus.Collector) map[string]float64 {
	t.Helper()

	registry := prometheus.NewPedanticRegistry()
	registry.MustRegister(collector)
	metricFamilies, err := registry.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}

	values := make(map[string]float64, 2)
	for _, family := range metricFamilies {
		switch family.GetName() {
		case "play_seconds_total", "estimated_transmit_bytes_total":
			if len(family.Metric) != 1 || family.Metric[0].Counter == nil {
				t.Fatalf("%s metrics = %v, want one counter", family.GetName(), family.Metric)
			}
			values[family.GetName()] = family.Metric[0].Counter.GetValue()
		}
	}

	for _, metricName := range []string{"play_seconds_total", "estimated_transmit_bytes_total"} {
		if _, ok := values[metricName]; !ok {
			t.Errorf("metric %s was not collected", metricName)
		}
	}
	return values
}
