package resources

import "testing"

func TestEmbeddedIconPresent(t *testing.T) {
	if len(Icon) < 1024 {
		t.Fatalf("embedded Snapvera icon is unexpectedly small: %d bytes", len(Icon))
	}
}
