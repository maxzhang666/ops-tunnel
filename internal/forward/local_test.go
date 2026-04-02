package forward

import (
	"testing"

	"github.com/maxzhang666/ops-tunnel/internal/config"
)

func testMapping() config.Mapping {
	return config.Mapping{
		ID:      "m1",
		Listen:  config.Endpoint{Host: "127.0.0.1", Port: 0},
		Connect: config.Endpoint{Host: "127.0.0.1", Port: 5432},
	}
}

func TestLocalForwarder_InitialStatus(t *testing.T) {
	fwd := NewLocalForwarder(testMapping())
	st := fwd.Status()

	if st.MappingID != "m1" {
		t.Errorf("MappingID = %q, want %q", st.MappingID, "m1")
	}
	if st.State != "stopped" {
		t.Errorf("State = %q, want %q", st.State, "stopped")
	}
	if st.ActiveConns != 0 {
		t.Errorf("ActiveConns = %d, want 0", st.ActiveConns)
	}
	if st.TotalConns != 0 {
		t.Errorf("TotalConns = %d, want 0", st.TotalConns)
	}
}
