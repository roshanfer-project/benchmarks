package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	reRqTotal     = regexp.MustCompile(`^envoy_http_downstream_rq_total\{envoy_http_conn_manager_prefix="([^"]+)"\}\s+(\S+)`)
	reRqCompleted = regexp.MustCompile(`^envoy_http_downstream_rq_completed\{envoy_http_conn_manager_prefix="([^"]+)"\}\s+(\S+)`)
)

var statIDPrefixes = []string{"ingress_", "inbound_"}

func main() {
	adminURL := envOr("ENVOY_ADMIN_URL", "http://127.0.0.1:9901")
	outPath := envOr("OUTPUT_PATH", "/tmp/envoy_stats.csv")
	appName := os.Getenv("APP_NAME")
	intervalMs := envIntOr("POLL_INTERVAL_MS", 200)

	statsURL := adminURL + "/stats/prometheus"
	interval := time.Duration(intervalMs) * time.Millisecond

	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		log.Fatal(err)
	}
	f, err := os.Create(outPath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	client := &http.Client{Timeout: 2 * time.Second}
	var startWall time.Time
	started := false
	headerWritten := false
	var listeners []string

	log.Printf("envoy-stats-exporter: app=%s interval=%s url=%s out=%s", appName, interval, statsURL, outPath)

	for {
		loopStart := time.Now()
		if !started {
			startWall = loopStart
			started = true
		}

		reqBy, respBy, ok := fetchCounters(client, statsURL)
		if !ok {
			elapsed := time.Since(loopStart)
			if sleep := interval - elapsed; sleep > 0 {
				time.Sleep(sleep)
			}
			continue
		}

		ids := discoverStatIDs(reqBy)
		if len(ids) == 0 {
			elapsed := time.Since(loopStart)
			if sleep := interval - elapsed; sleep > 0 {
				time.Sleep(sleep)
			}
			continue
		}
		if !headerWritten {
			listeners = ids
			if err := writeMultiHeader(w, listeners); err != nil {
				log.Fatal(err)
			}
			w.Flush()
			headerWritten = true
		}
		row := buildMultiRow(loopStart, startWall, listeners, reqBy, respBy)

		if err := w.Write(row); err != nil {
			log.Printf("write error: %v", err)
		} else {
			w.Flush()
			f.Sync()
		}

		elapsed := time.Since(loopStart)
		if sleep := interval - elapsed; sleep > 0 {
			time.Sleep(sleep)
		}
	}
}

func writeMultiHeader(w *csv.Writer, listeners []string) error {
	header := []string{"timestamp", "relative_time_s"}
	for _, id := range listeners {
		header = append(header, id+"-Req", id+"-Resp")
	}
	return w.Write(header)
}

func buildMultiRow(loopStart, startWall time.Time, listeners []string, reqBy, respBy map[string]float64) []string {
	rel := loopStart.Sub(startWall).Seconds()
	row := []string{
		loopStart.UTC().Format("2006-01-02T15:04:05.000000"),
		fmt.Sprintf("%.3f", rel),
	}
	for _, id := range listeners {
		key := statKeyForID(id, reqBy)
		row = append(row,
			fmt.Sprintf("%.0f", reqBy[key]),
			fmt.Sprintf("%.0f", respBy[key]),
		)
	}
	return row
}

func statKeyForID(id string, reqBy map[string]float64) string {
	for _, pfx := range statIDPrefixes {
		key := pfx + id
		if _, ok := reqBy[key]; ok {
			return key
		}
	}
	return id
}

func fetchCounters(client *http.Client, url string) (map[string]float64, map[string]float64, bool) {
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("fetch error: %v", err)
		return nil, nil, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("fetch status: %d", resp.StatusCode)
		return nil, nil, false
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("read error: %v", err)
		return nil, nil, false
	}
	reqBy, respBy := parseCounters(string(body))
	return reqBy, respBy, true
}

func parseCounters(text string) (map[string]float64, map[string]float64) {
	reqBy := make(map[string]float64)
	respBy := make(map[string]float64)
	for _, line := range splitLines(text) {
		if m := reRqTotal.FindStringSubmatch(line); m != nil {
			reqBy[m[1]], _ = strconv.ParseFloat(m[2], 64)
			continue
		}
		if m := reRqCompleted.FindStringSubmatch(line); m != nil {
			respBy[m[1]], _ = strconv.ParseFloat(m[2], 64)
		}
	}
	return reqBy, respBy
}

func discoverStatIDs(reqBy map[string]float64) []string {
	var ids []string
	seen := make(map[string]bool)
	for key := range reqBy {
		for _, pfx := range statIDPrefixes {
			if !strings.HasPrefix(key, pfx) {
				continue
			}
			id := key[len(pfx):]
			if id == "" || seen[id] {
				break
			}
			seen[id] = true
			ids = append(ids, id)
			break
		}
	}
	sort.Strings(ids)
	return ids
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envIntOr(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
