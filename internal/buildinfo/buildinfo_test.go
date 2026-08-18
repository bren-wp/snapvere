package buildinfo

import (
	"net/url"
	"regexp"
	"testing"
)

func TestReleaseMetadata(t *testing.T) {
	if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(Version) {
		t.Fatalf("Version must be semantic x.y.z, got %q", Version)
	}
	for name, value := range map[string]string{
		"Name": Name, "Author": Author, "Publisher": Publisher, "Domain": Domain, "Copyright": Copyright,
	} {
		if value == "" {
			t.Fatalf("%s must not be empty", name)
		}
	}
	u, err := url.Parse(Website)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		t.Fatalf("Website must be a valid HTTPS URL, got %q", Website)
	}
	if u.Host != Domain {
		t.Fatalf("Website host %q does not match Domain %q", u.Host, Domain)
	}
}
