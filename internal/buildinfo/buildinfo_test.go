package buildinfo

import "testing"

func TestServiceVersion(t *testing.T) {
	oldRelease := ReleaseVersion
	oldCommit := Commit
	t.Cleanup(func() {
		ReleaseVersion = oldRelease
		Commit = oldCommit
	})

	ReleaseVersion = "2026.04.0"
	Commit = "04fc78e"

	if got := ServiceVersion(); got != "2026.04.0+04fc78e" {
		t.Fatalf("expected stamped version, got %q", got)
	}
}

func TestServiceVersionWithoutCommit(t *testing.T) {
	oldRelease := ReleaseVersion
	oldCommit := Commit
	t.Cleanup(func() {
		ReleaseVersion = oldRelease
		Commit = oldCommit
	})

	ReleaseVersion = "2026.04.0"
	Commit = ""

	if got := ServiceVersion(); got != "2026.04.0" {
		t.Fatalf("expected release version only, got %q", got)
	}
}
