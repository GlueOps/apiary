package agentdist

import (
	"testing"
	"testing/fstest"
)

// TestAServerWithNoAgentBinariesIsNotAnError.
//
// A development build carries none, and failing to start over it would make the
// whole control plane depend on a cross-compiled artifact nobody needs locally.
// The agent reads an empty catalogue as nothing to do.
func TestAServerWithNoAgentBinariesIsNotAnError(t *testing.T) {
	t.Parallel()

	got, err := read(fstest.MapFS{}, "v1.0.0")
	if err != nil {
		t.Fatalf("an empty distribution was an error: %v", err)
	}
	if len(got.Builds) != 0 {
		t.Fatalf("builds = %v, want none", got.Builds)
	}
	if got.Version != "v1.0.0" {
		t.Fatalf("version = %q", got.Version)
	}
}

// TestTheCatalogueCarriesADigestPerBuild.
//
// It is the only thing standing between an agent and executing whatever arrived
// over the network.
func TestTheCatalogueCarriesADigestPerBuild(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"dist/apiary-agent_linux_amd64": {Data: []byte("amd64 binary")},
		"dist/apiary-agent_linux_arm64": {Data: []byte("arm64 binary")},
	}
	got, err := read(fsys, "v2.0.0")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(got.Builds) != 2 {
		t.Fatalf("%d builds, want 2", len(got.Builds))
	}
	for _, b := range got.Builds {
		if b.SHA256 == "" {
			t.Fatalf("%s/%s has no digest, so an agent could not verify it", b.OS, b.Arch)
		}
		if b.Size == 0 {
			t.Fatalf("%s/%s reports zero bytes", b.OS, b.Arch)
		}
	}
	// Two different binaries must not share a digest, or the check proves nothing.
	if got.Builds[0].SHA256 == got.Builds[1].SHA256 {
		t.Fatal("two different binaries produced the same digest")
	}
}

// TestTheCatalogueIsStable, so two servers of the same version answer
// identically and a client can compare them.
func TestTheCatalogueIsStable(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"dist/apiary-agent_linux_arm64": {Data: []byte("b")},
		"dist/apiary-agent_linux_amd64": {Data: []byte("a")},
	}
	first, _ := read(fsys, "v1")
	for i := 0; i < 10; i++ {
		again, _ := read(fsys, "v1")
		for j := range first.Builds {
			if first.Builds[j] != again.Builds[j] {
				t.Fatalf("build order or content moved between reads: %v then %v",
					first.Builds, again.Builds)
			}
		}
	}
	if first.Builds[0].Arch != "amd64" {
		t.Fatalf("not sorted: %v", first.Builds)
	}
}

// TestFilesThatAreNotAgentBinariesAreIgnored, so a stray README or checksum file
// in the distribution directory does not become a platform.
func TestFilesThatAreNotAgentBinariesAreIgnored(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"dist/apiary-agent_linux_amd64": {Data: []byte("real")},
		"dist/README.md":                {Data: []byte("notes")},
		"dist/checksums.txt":            {Data: []byte("sums")},
		"dist/.gitkeep":                 {Data: []byte("")},
	}
	got, _ := read(fsys, "v1")
	if len(got.Builds) != 1 {
		t.Fatalf("%d builds, want 1: %v", len(got.Builds), got.Builds)
	}
}

// TestAPlatformCannotEscapeTheDistributionDirectory.
//
// The os and arch come off the URL and become a path. fs.FS refuses traversal on
// its own, but a refusal here says why — and it stops the attempt before it
// reaches the filesystem at all.
func TestAPlatformCannotEscapeTheDistributionDirectory(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct{ goos, goarch string }{
		{"../..", "amd64"},
		{"linux", "../../etc/shadow"},
		{"linux/../..", "amd64"},
		{"", "amd64"},
		{"linux", ""},
		{"LINUX", "amd64"},
		{"linux", "amd64; rm -rf /"},
	} {
		if _, err := agentFileName(tc.goos, tc.goarch); err == nil {
			t.Errorf("%q/%q was accepted as a platform", tc.goos, tc.goarch)
		}
	}

	// And the ordinary ones still work.
	for _, tc := range []struct{ goos, goarch string }{
		{"linux", "amd64"},
		{"linux", "arm64"},
	} {
		if _, err := agentFileName(tc.goos, tc.goarch); err != nil {
			t.Errorf("%q/%q was refused: %v", tc.goos, tc.goarch, err)
		}
	}
}
