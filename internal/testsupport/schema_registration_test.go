package testsupport_test

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var (
	reDeclare = regexp.MustCompile(`(\w+)\s*=\s*testsupport\.SchemaName\(`)
	reDrop    = regexp.MustCompile(`(?s)testsupport\.DropSchemas\((.*?)\)`)
	reIdent   = regexp.MustCompile(`[A-Za-z_]\w*`)
)

// Every per-run schema a package declares must be dropped by that package's
// TestMain.
//
// The names are unique per invocation, which is what stops two concurrent runs
// treading on each other's fixtures — and it is also what makes forgetting to
// drop one invisible. A missed schema does not fail anything, collide with
// anything, or slow anything down; it accumulates one set of tables per test run
// in a shared database until somebody notices thousands of them. CI catches it
// only because a separate job counts leftover schemas after a full run, and that
// job reports a number with no indication of which package is responsible.
//
// This asserts the thing directly, in the package that owns the convention, and
// names the offender. It reads the source rather than observing behaviour
// because the failure is an omission: there is nothing to observe at runtime
// except the absence of a DROP that nobody wrote.
func TestEverySchemaDeclaredIsAlsoDropped(t *testing.T) {
	root := repoRoot(t)

	declared := map[string]map[string]string{} // package dir -> name -> file
	dropped := map[string]map[string]bool{}    // package dir -> name

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			switch info.Name() {
			case "node_modules", ".git", "bin", "webdist":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		src, dir := string(b), filepath.Dir(path)

		for _, m := range reDeclare.FindAllStringSubmatch(src, -1) {
			if declared[dir] == nil {
				declared[dir] = map[string]string{}
			}
			rel, _ := filepath.Rel(root, path)
			declared[dir][m[1]] = rel
		}
		for _, m := range reDrop.FindAllStringSubmatch(src, -1) {
			if dropped[dir] == nil {
				dropped[dir] = map[string]bool{}
			}
			for _, id := range reIdent.FindAllString(m[1], -1) {
				dropped[dir][id] = true
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk the tree: %v", err)
	}

	if len(declared) == 0 {
		t.Fatal("found no testsupport.SchemaName declarations anywhere, so this test is " +
			"asserting nothing — the pattern it matches has probably changed")
	}

	var problems []string
	for dir, names := range declared {
		for name, file := range names {
			if !dropped[dir][name] {
				rel, _ := filepath.Rel(root, dir)
				problems = append(problems, rel+": "+name+" (declared in "+file+")")
			}
		}
	}
	sort.Strings(problems)
	if len(problems) > 0 {
		t.Fatalf("these per-run schemas are created but never dropped, so every run of "+
			"their package leaves another set of tables behind in the shared test "+
			"database:\n  %s\n\nAdd each to the testsupport.DropSchemas call in that "+
			"package's TestMain.", strings.Join(problems, "\n  "))
	}
}

// repoRoot walks up until it finds the go.mod that owns this module.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 10; i++ {
		if b, err := os.ReadFile(filepath.Join(dir, "go.mod")); err == nil &&
			strings.Contains(string(b), "module github.com/glueops/apiary\n") {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("could not find the repository root from the test's working directory")
	return ""
}
