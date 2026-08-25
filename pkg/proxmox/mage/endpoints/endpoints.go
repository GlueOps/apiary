package endpoints

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	upstreamURL = "https://pve.proxmox.com/pve-docs/api-viewer/apidoc.js"
	cacheDir    = ".cache/pve-api"
)

var intentionallyExcluded = map[string]string{
	"get /nodes/{}/qemu/{}/mtunnelwebsocket":         "URL-builder helper (MigrationTunnelWebSocketPath); caller drives the websocket dialer",
	"get /nodes/{}/lxc/{}/mtunnelwebsocket":          "URL-builder helper (MigrationTunnelWebSocketPath); caller drives the websocket dialer",
	"get /nodes/{}/storage/{}/file-restore/download": "streaming binary download; needs custom io.Reader plumbing, not the JSON Get helper",
}

type methodInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type node struct {
	Text     string                `json:"text"`
	Info     map[string]methodInfo `json:"info"`
	Children []node                `json:"children"`
}

type endpoint struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func Sync() error {
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}
	fmt.Printf("Fetching %s\n", upstreamURL)
	if err := download(upstreamURL, filepath.Join(cacheDir, "apidoc.js")); err != nil {
		return err
	}
	return Extract()
}

func Extract() error {
	cache := filepath.Join(cacheDir, "apidoc.js")
	raw, err := os.ReadFile(cache)
	if err != nil {
		return fmt.Errorf("read %s: %w (run `mage endpoints:sync` first)", cache, err)
	}
	jsonBlob, err := extractSchemaJSON(raw)
	if err != nil {
		return err
	}
	var schema []node
	if err := json.Unmarshal(jsonBlob, &schema); err != nil {
		return fmt.Errorf("parse apidoc.js schema: %w", err)
	}

	out := flatten(schema)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Method < out[j].Method
	})

	jsonBytes, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "endpoints.json"), jsonBytes, 0o644); err != nil {
		return err
	}
	var txt bytes.Buffer
	for _, e := range out {
		fmt.Fprintf(&txt, "%-7s %s\n", e.Method, e.Path)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "endpoints.txt"), txt.Bytes(), 0o644); err != nil {
		return err
	}
	printSummary(out)
	return nil
}

func Coverage() error {
	endpointsFile := filepath.Join(cacheDir, "endpoints.json")
	if _, err := os.Stat(endpointsFile); err != nil {
		if err := Sync(); err != nil {
			return err
		}
	}

	raw, err := os.ReadFile(endpointsFile)
	if err != nil {
		return err
	}
	var schema []endpoint
	if err := json.Unmarshal(raw, &schema); err != nil {
		return err
	}

	schemaKey := make(map[string]struct{}, len(schema))
	schemaByArea := map[string]map[string]struct{}{}
	for _, e := range schema {
		k := key(e.Method, normalizePath(e.Path))
		schemaKey[k] = struct{}{}
		a := areaOf(normalizePath(e.Path))
		if schemaByArea[a] == nil {
			schemaByArea[a] = map[string]struct{}{}
		}
		schemaByArea[a][k] = struct{}{}
	}

	for k := range intentionallyExcluded {
		if _, ok := schemaKey[k]; !ok {
			return fmt.Errorf("intentionallyExcluded entry %q is not in the upstream schema — update or remove it", k)
		}
	}

	calls, err := scanCallSites(".")
	if err != nil {
		return err
	}

	covered := map[string]struct{}{}
	var unmatched []extractedCall
	matched, missed := 0, 0
	for _, c := range calls {
		k := key(c.Method, c.Path)
		if _, ok := schemaKey[k]; ok {
			matched++
			covered[k] = struct{}{}
		} else {
			missed++
			unmatched = append(unmatched, c)
		}
	}
	sort.Slice(unmatched, func(i, j int) bool {
		if unmatched[i].File != unmatched[j].File {
			return unmatched[i].File < unmatched[j].File
		}
		return unmatched[i].Line < unmatched[j].Line
	})

	var uncovered []string
	excludedHits := 0
	for _, e := range schema {
		k := key(e.Method, normalizePath(e.Path))
		if _, ok := covered[k]; ok {
			continue
		}
		if _, ok := intentionallyExcluded[k]; ok {
			excludedHits++
			continue
		}
		uncovered = append(uncovered, fmt.Sprintf("%-7s %s", e.Method, e.Path))
	}
	sort.Strings(uncovered)
	if err := os.WriteFile(
		filepath.Join(cacheDir, "coverage.txt"),
		[]byte(strings.Join(uncovered, "\n")+"\n"),
		0o644,
	); err != nil {
		return err
	}

	type areaRow struct {
		area                         string
		total, cov, excluded, missed int
	}
	var rows []areaRow
	for area, keys := range schemaByArea {
		row := areaRow{area: area, total: len(keys)}
		for k := range keys {
			if _, ok := covered[k]; ok {
				row.cov++
				continue
			}
			if _, ok := intentionallyExcluded[k]; ok {
				row.excluded++
			}
		}
		row.missed = row.total - row.cov - row.excluded
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].total != rows[j].total {
			return rows[i].total > rows[j].total
		}
		return rows[i].area < rows[j].area
	})

	var areaBuf bytes.Buffer
	fmt.Fprintf(&areaBuf, "%-25s %8s %8s %8s %8s %8s\n", "area", "covered", "total", "excluded", "missing", "pct")
	for _, r := range rows {

		denom := r.total - r.excluded
		fmt.Fprintf(&areaBuf, "%-25s %8d %8d %8d %8d %7.1f%%\n",
			r.area, r.cov, r.total, r.excluded, r.missed, pct(r.cov, denom))
	}
	if err := os.WriteFile(
		filepath.Join(cacheDir, "coverage_by_area.txt"),
		areaBuf.Bytes(),
		0o644,
	); err != nil {
		return err
	}

	fmt.Printf("\ngo-proxmox endpoint coverage — %s\n", time.Now().Format("2006-01-02"))
	fmt.Printf("  schema endpoints   : %d\n", len(schema))
	fmt.Printf("  call sites scanned : %d (matched=%d, no-match=%d → match rate %.1f%%)\n",
		len(calls), matched, missed, pct(matched, len(calls)))

	considered := len(schema) - excludedHits
	fmt.Printf("  unique covered     : %d / %d  →  %.1f%%  (excluded by design: %d)\n",
		len(covered), considered, pct(len(covered), considered), excludedHits)
	fmt.Println()
	fmt.Print(areaBuf.String())
	if len(unmatched) > 0 {
		fmt.Printf("\ncall sites that don't resolve to a schema endpoint (%d):\n", len(unmatched))
		fmt.Println("  (real coverage findings — either stale Go code, or one Go call covering multiple schema endpoints)")
		for _, c := range unmatched {
			fmt.Printf("  %s:%d  %s %s\n", c.File, c.Line, c.Method, c.Path)
		}
	}
	fmt.Printf("\nwrote:\n  %s/coverage.txt           # %d uncovered endpoints\n  %s/coverage_by_area.txt\n",
		cacheDir, len(uncovered), cacheDir)
	return nil
}

func scanCallSites(root string) ([]extractedCall, error) {
	var out []extractedCall
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".claude", ".cache", "examples", "tests", "audit", "mage":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() {
			if cerr := f.Close(); cerr != nil {
				fmt.Fprintln(os.Stderr, "close:", cerr)
			}
		}()
		s := bufio.NewScanner(f)
		s.Buffer(make([]byte, 1<<16), 1<<20)

		urlVars := map[string]string{}

		pathVars := map[string]string{}

		var pendingMethod string
		var pendingLine int
		for line := 1; s.Scan(); line++ {
			text := s.Text()

			if pendingMethod != "" {
				trimmed := strings.TrimSpace(text)
				if trimmed == "" || strings.HasPrefix(trimmed, "//") {
					continue
				}
				if pathLit := resolvePath(text, urlVars, pathVars); pathLit != "" {
					out = append(out, extractedCall{
						Method: pendingMethod,
						Path:   normalizePath(pathLit),
						File:   filepath.ToSlash(path),
						Line:   pendingLine,
					})
				}
				pendingMethod = ""
				pendingLine = 0

			}

			if am := urlAssignRE.FindStringSubmatch(text); am != nil {
				if p := extractPathExpr(am[2]); p != "" {
					urlVars[am[1]] = p
				}
			}
			if pm := pathAssignRE.FindStringSubmatch(text); pm != nil {
				if p := extractPathExpr(pm[2]); p != "" {
					pathVars[pm[1]] = p
				}
			}

			if m := callRE.FindStringSubmatch(text); m != nil {
				method := m[1]
				rest := m[2]
				pathLit := resolvePath(rest, urlVars, pathVars)
				if pathLit != "" {
					out = append(out, extractedCall{
						Method: method,
						Path:   normalizePath(pathLit),
						File:   filepath.ToSlash(path),
						Line:   line,
					})
				}
				continue
			}

			if wm := wsCallRE.FindStringSubmatch(text); wm != nil {
				rest := wm[2]
				pathLit := resolvePath(rest, urlVars, pathVars)
				if pathLit == "" {
					continue
				}

				out = append(out, extractedCall{
					Method: "GET",
					Path:   normalizePath(pathLit),
					File:   filepath.ToSlash(path),
					Line:   line,
				})
				continue
			}

			if dm := diridxRE.FindStringSubmatch(text); dm != nil {
				pathLit := resolvePath(dm[1], urlVars, pathVars)
				if pathLit == "" {
					continue
				}
				out = append(out, extractedCall{
					Method: "GET",
					Path:   normalizePath(pathLit),
					File:   filepath.ToSlash(path),
					Line:   line,
				})
				continue
			}

			if um := uploadRE.FindStringSubmatch(text); um != nil {
				if pathLit := resolvePath(um[1], urlVars, pathVars); pathLit != "" {
					out = append(out, extractedCall{
						Method: "POST",
						Path:   normalizePath(pathLit),
						File:   filepath.ToSlash(path),
						Line:   line,
					})
					continue
				}

				pendingMethod = "POST"
				pendingLine = line
			}
		}
		return s.Err()
	})
	return out, err
}

func resolvePath(expr string, urlVars, pathVars map[string]string) string {
	if p := extractPathExpr(expr); p != "" {
		return p
	}
	trimmed := strings.TrimSpace(expr)
	if um := urlStringRE.FindStringSubmatch(trimmed); um != nil {
		return urlVars[um[1]]
	}
	if vm := varRefRE.FindStringSubmatch(trimmed); vm != nil {
		return pathVars[vm[1]]
	}
	return ""
}

func extractPathExpr(expr string) string {
	if sm := sprintfRE.FindStringSubmatch(expr); sm != nil {
		return sm[1]
	}
	if t := strings.TrimSpace(expr); strings.HasPrefix(t, `"`) {
		if end := strings.Index(t[1:], `"`); end > 0 {
			return t[1 : 1+end]
		}
	}
	return ""
}

type extractedCall struct {
	Method, Path, File string
	Line               int
}

var (
	callRE   = regexp.MustCompile(`\b\w+\.(Get|Post|Put|Delete)(?:WithParams)?\s*\(\s*ctx\s*,\s*(.*?)$`)
	wsCallRE = regexp.MustCompile(`\b\w+\.(VNCWebSocket|TermWebSocket)\s*\(\s*(.*?)$`)

	diridxRE = regexp.MustCompile(`\b\w+\.\w*[Dd]iridx\s*\(\s*ctx\s*,\s*(.*?)$`)

	uploadRE    = regexp.MustCompile(`\bclient\.Upload\w*\s*\(\s*(.*?)$`)
	sprintfRE   = regexp.MustCompile(`fmt\.Sprintf\(\s*"([^"]+)"`)
	urlAssignRE = regexp.MustCompile(`\b(\w+)\s*:?=\s*url\.URL\{\s*Path:\s*(.+)\}`)

	pathAssignRE = regexp.MustCompile(`\b(\w+)\s*:?=\s*(fmt\.Sprintf\(\s*"[^"]+"|"/[^"]*")`)
	urlStringRE  = regexp.MustCompile(`^(\w+)\.String\(\)`)
	varRefRE     = regexp.MustCompile(`^(\w+)\s*[,)]`)
	braceRE      = regexp.MustCompile(`\{[^}]+\}`)
	fmtVerbRE    = regexp.MustCompile(`%[sdvqxXt]`)

	volidRE = regexp.MustCompile(`\{\}:\{\}/\{\}`)
)

func normalizePath(p string) string {
	if i := strings.IndexByte(p, '?'); i >= 0 {
		p = p[:i]
	}
	p = braceRE.ReplaceAllString(p, "{}")
	p = fmtVerbRE.ReplaceAllString(p, "{}")
	p = volidRE.ReplaceAllString(p, "{}")
	if len(p) > 1 {
		p = strings.TrimRight(p, "/")
	}
	return p
}

func key(method, path string) string {
	return strings.ToLower(method) + " " + path
}

func areaOf(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		return "(root)"
	}
	if parts[0] == "nodes" && len(parts) >= 3 {
		return "nodes/" + parts[2]
	}
	return parts[0]
}

func pct(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return float64(n) * 100 / float64(d)
}

func extractSchemaJSON(raw []byte) ([]byte, error) {
	start := bytes.IndexByte(raw, '[')
	endIdx := bytes.Index(raw, []byte("\n]\n;"))
	if start < 0 || endIdx <= start {
		return nil, fmt.Errorf("could not locate JSON array boundaries in apidoc.js")
	}
	return raw[start : endIdx+2], nil
}

func flatten(schema []node) []endpoint {
	var out []endpoint
	var walk func(n node, path string)
	walk = func(n node, path string) {
		here := path
		if n.Text != "" {
			here = path + "/" + n.Text
		}
		for method, meta := range n.Info {
			desc := meta.Description
			if i := strings.IndexByte(desc, '\n'); i >= 0 {
				desc = desc[:i]
			}
			out = append(out, endpoint{
				Method:      method,
				Path:        here,
				Name:        meta.Name,
				Description: strings.TrimSpace(desc),
			})
		}
		for _, c := range n.Children {
			walk(c, here)
		}
	}
	for _, top := range schema {
		walk(top, "")
	}
	return out
}

func download(url, dst string) (err error) {
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close response body: %w", cerr)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close %s: %w", dst, cerr)
		}
	}()
	_, err = io.Copy(f, resp.Body)
	return err
}

func printSummary(out []endpoint) {
	uniqPaths := map[string]struct{}{}
	areas := map[string]int{}
	methodCounts := map[string]int{}
	for _, e := range out {
		uniqPaths[e.Path] = struct{}{}
		methodCounts[e.Method]++
		areas[topArea(e.Path)]++
	}
	fmt.Printf("\nProxmox VE API snapshot — %s\n", time.Now().Format("2006-01-02"))
	fmt.Printf("  endpoints   : %d\n", len(out))
	fmt.Printf("  unique paths: %d\n", len(uniqPaths))
	fmt.Printf("  methods     :")
	for _, m := range sortedKeys(methodCounts) {
		fmt.Printf(" %s=%d", m, methodCounts[m])
	}
	fmt.Println()
	fmt.Printf("  top areas   :")
	for _, a := range sortedKeys(areas) {
		fmt.Printf(" %s=%d", a, areas[a])
	}
	fmt.Println()
	fmt.Printf("\nwrote:\n  %s/endpoints.json\n  %s/endpoints.txt\n", cacheDir, cacheDir)
}

func topArea(path string) string {
	p := strings.TrimPrefix(path, "/")
	if i := strings.IndexByte(p, '/'); i >= 0 {
		return p[:i]
	}
	return p
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
