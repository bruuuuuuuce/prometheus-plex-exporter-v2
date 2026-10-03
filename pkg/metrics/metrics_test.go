package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestPlayMetricsUseLibraryLabelsInDescriptorOrder(t *testing.T) {
	tests := []struct {
		name   string
		metric prometheus.Metric
	}{
		{
			name: "plays_total",
			metric: Play(
				1,
				"plex", "Test Server", "server-id",
				"Movies", "library-id", "movie",
				"movie", "Test Title", "", "",
				"DirectPlay", "1080", "1080", "10000",
				"Test Device", "tv", "Test User", "session-id",
			),
		},
		{
			name: "play_seconds_total",
			metric: PlayDuration(
				12,
				"plex", "Test Server", "server-id",
				"Movies", "library-id", "movie",
				"movie", "Test Title", "", "",
				"DirectPlay", "1080", "1080", "10000",
				"Test Device", "tv", "Test User", "session-id",
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var written dto.Metric
			if err := tt.metric.Write(&written); err != nil {
				t.Fatalf("write metric: %v", err)
			}

			labels := make(map[string]string, len(written.GetLabel()))
			for _, label := range written.GetLabel() {
				labels[label.GetName()] = label.GetValue()
			}

			for name, want := range map[string]string{
				"library_type": "movie",
				"library":      "Movies",
				"library_id":   "library-id",
			} {
				if got := labels[name]; got != want {
					t.Errorf("label %q = %q, want %q", name, got, want)
				}
			}
		})
	}
}
