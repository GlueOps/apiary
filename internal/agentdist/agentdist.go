package agentdist

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
	"sync"
)

// Distributing the agent binary from the control plane.
//
// It is not a convenience. A hypervisor in a managed estate has no route to the
// internet — that is the point of the estate — so the control plane is the only
// host it can fetch anything from, and without this an upgrade means an operator
// copying a binary to every machine by hand. Which is also how a fleet ends up
// running four different agent versions.
//
// The binaries are embedded, so they ship inside the server binary and there is
// no second thing to deploy or keep in sync. The cost is honest and worth
// stating: each architecture adds its own size to the server binary.
//
// ITS OWN PACKAGE rather than the SPA's embedded tree, which is where this
// started. internal/api imports the handlers, so a handler reaching back into
// internal/api for these bytes is an import cycle — and separating them is
// better regardless: the agent distribution and the web assets have nothing to
// do with each other beyond both being embedded.
//
// SERVED THROUGH AN AUTHENTICATED ENDPOINT rather than as a static asset. A
// static path would hand the fleet's agent binary to anyone who could reach the
// login page, and under the SPA's catch-all it would collide with routing the
// first time somebody visited /agent/... in a browser.

// distDir is where agent binaries live inside this package's embedded tree.
//
// The directory is committed with a .gitkeep and its contents are gitignored:
// //go:embed fails to COMPILE on a missing directory, so an empty one is what
// keeps a checkout buildable before any agent has been cross-compiled into it.
const distDir = "dist"

//go:embed all:dist
var embedded embed.FS

// Build is one downloadable binary.
type Build struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
	Size int64  `json:"size"`
	// SHA256 is what the agent verifies the download against before it installs
	// anything. Without it an upgrade is "execute whatever arrived", and a
	// truncated download is a binary that either fails to start or, worse,
	// starts.
	SHA256 string `json:"sha256"`
}

// Dist is the catalogue an agent reads to decide whether to upgrade.
type Dist struct {
	// Version is the control plane's own version, which is what these binaries
	// were built alongside. An agent compares its own against it.
	//
	// Deliberately the SERVER's version rather than one read out of the binary:
	// the two ship together from one release, and parsing a version back out of
	// an executable is guesswork that fails silently.
	Version string  `json:"version"`
	Builds  []Build `json:"builds"`
}

var (
	distOnce sync.Once
	distVal  Dist
	distErr  error
)

// Catalogue lists the agent binaries this server can serve.
//
// Computed ONCE. The digest of a 20MB file is not something to recompute per
// request, and the embedded tree cannot change while the process runs.
func Catalogue(version string) (Dist, error) {
	distOnce.Do(func() {
		distVal, distErr = read(embedded, version)
	})
	return distVal, distErr
}

func read(fsys fs.FS, version string) (Dist, error) {
	out := Dist{Version: version, Builds: []Build{}}

	entries, err := fs.ReadDir(fsys, distDir)
	if err != nil {
		// NOT an error. A server built without agent binaries is a normal
		// development build, and failing to start over it would make the SPA
		// depend on a cross-compiled artifact nobody needs locally. The endpoint
		// says the catalogue is empty and the agent declines to upgrade.
		return out, nil
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		goos, goarch, ok := parseAgentName(e.Name())
		if !ok {
			continue
		}
		body, err := fs.ReadFile(fsys, path.Join(distDir, e.Name()))
		if err != nil {
			return Dist{}, fmt.Errorf("agent dist: read %s: %w", e.Name(), err)
		}
		sum := sha256.Sum256(body)
		out.Builds = append(out.Builds, Build{
			OS: goos, Arch: goarch, Size: int64(len(body)),
			SHA256: "sha256:" + hex.EncodeToString(sum[:]),
		})
	}
	// Sorted so the catalogue is stable between builds and two servers of the
	// same version answer identically.
	sort.Slice(out.Builds, func(i, j int) bool {
		if out.Builds[i].OS != out.Builds[j].OS {
			return out.Builds[i].OS < out.Builds[j].OS
		}
		return out.Builds[i].Arch < out.Builds[j].Arch
	})
	return out, nil
}

// Binary returns the bytes for one os/arch.
func Binary(goos, goarch string) ([]byte, error) {
	name, err := agentFileName(goos, goarch)
	if err != nil {
		return nil, err
	}
	return fs.ReadFile(embedded, path.Join(distDir, name))
}

// agentFileName is the one place the naming convention lives, so the reader and
// the build that produces the files cannot disagree about it.
func agentFileName(goos, goarch string) (string, error) {
	if !safeIdent(goos) || !safeIdent(goarch) {
		// Rejected rather than sanitised: these become a path, and a caller
		// supplying "../.." is asking for a file outside the distribution tree.
		// fs.FS refuses traversal on its own, but a refusal here says why.
		return "", fmt.Errorf("agent dist: %q/%q is not a platform", goos, goarch)
	}
	return "apiary-agent_" + goos + "_" + goarch, nil
}

func parseAgentName(name string) (goos, goarch string, ok bool) {
	rest, found := strings.CutPrefix(name, "apiary-agent_")
	if !found {
		return "", "", false
	}
	goos, goarch, found = strings.Cut(rest, "_")
	if !found || !safeIdent(goos) || !safeIdent(goarch) {
		return "", "", false
	}
	return goos, goarch, true
}

// safeIdent allows only what a GOOS or GOARCH can contain, which excludes every
// character that could leave the distribution directory.
func safeIdent(s string) bool {
	if s == "" || len(s) > 16 {
		return false
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

// Stamp is the version `just agent-dist` stamped into these binaries, or "" when
// none have been built into this server.
//
// It exists because nothing else can answer the question. The version is applied
// with `-X main.version`, which is only readable by executing the binary, and
// embed.FS carries no modification times — so a server has no way to tell
// whether the agents it is about to install came from its own source or from
// whatever was last cross-compiled into the tree.
//
// That is not hypothetical. The dist directory is gitignored and produced by one
// recipe that nothing depends on, so editing agent code and running the server
// ships the previous build with no warning at all. A change that made a
// dist-upgrade stop requiring a Proxmox token was committed at 18:24 and was
// still missing from every host onboarded hours afterwards, because the binaries
// were built at 17:24.
func Stamp() string {
	b, err := embedded.ReadFile(distDir + "/stamp")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// WarnIfStale reports a mismatch between the embedded agents and this server.
//
// A warning rather than a refusal: agents built from a different commit are
// usually a developer mid-change, and refusing to start would make the common
// case worse than the problem. What matters is that it is SAID — the failure
// this prevents is silent, and costs its whole diagnosis before anyone thinks to
// check a build artefact.
//
// Empty stamp means no agents are embedded at all, which Catalogue already
// reports usefully at the point an upgrade is attempted; saying it again at
// every startup would train people to ignore this line.
func WarnIfStale(serverVersion string, warn func(msg string, args ...any)) {
	stamp := Stamp()
	if stamp == "" || stamp == strings.TrimSpace(serverVersion) {
		return
	}
	warn("the embedded agent binaries were built from a different version than this "+
		"server, so onboarding and upgrades will install agents that do not match this "+
		"source; run `just agent-dist` and restart",
		"agent_build", stamp, "server_version", serverVersion)
}
