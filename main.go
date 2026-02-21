package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/benstraw/whoop-garden/internal/auth"
	"github.com/benstraw/whoop-garden/internal/client"
	"github.com/benstraw/whoop-garden/internal/fetch"
	"github.com/benstraw/whoop-garden/internal/render"
)

func main() {
	loadDotEnv(".env")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "auth":
		runAuth()
	case "daily":
		runDaily(args)
	case "weekly":
		runWeekly(args)
	case "persona":
		runPersona(args)
	case "fetch-all":
		runFetchAll(args)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`whoop-garden — WHOOP data → Obsidian markdown

Usage:
  whoop-garden auth                  Authenticate with WHOOP via OAuth
  whoop-garden daily [--date DATE]   Generate daily note (default: today)
  whoop-garden weekly [--date DATE]  Generate weekly note for DATE's week
  whoop-garden persona [--days N]    Generate 30-day persona section
  whoop-garden fetch-all [--days N]  Fetch and write notes for last N days

Flags:
  --date   Date in YYYY-MM-DD format (default: today)
  --days   Number of days (default: 30)
`)
}

// loadDotEnv reads a .env file and sets environment variables.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return // .env is optional
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		// Strip surrounding quotes if present.
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}
		if key != "" {
			os.Setenv(key, val)
		}
	}
}

// outputDir returns the output directory, preferring $OBSIDIAN_VAULT_PATH/Health/WHOOP/.
func outputDir() string {
	if vault := os.Getenv("OBSIDIAN_VAULT_PATH"); vault != "" {
		return filepath.Join(vault, "Health", "WHOOP")
	}
	return "./output"
}

// ensureOutputDir creates the output directory if it doesn't exist.
func ensureOutputDir() (string, error) {
	dir := outputDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create output dir %s: %w", dir, err)
	}
	return dir, nil
}

// ensureYearDir creates a year subdirectory under baseDir if it doesn't exist.
func ensureYearDir(baseDir string, year int) (string, error) {
	dir := filepath.Join(baseDir, fmt.Sprintf("%d", year))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create year dir %s: %w", dir, err)
	}
	return dir, nil
}

// getClient loads tokens (refreshing if needed) and returns an API client.
func getClient() (*client.Client, error) {
	token, err := auth.RefreshIfNeeded()
	if err != nil {
		return nil, fmt.Errorf("authentication error: %w\nRun 'whoop-garden auth' to authenticate.", err)
	}
	return client.NewClient(token), nil
}

// parseDate parses a YYYY-MM-DD date string or returns today.
func parseDate(s string) (time.Time, error) {
	if s == "" {
		return time.Now(), nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q (expected YYYY-MM-DD): %w", s, err)
	}
	return t, nil
}

// templatesDir returns the path to the templates directory relative to the binary.
func templatesDir() string {
	if td := os.Getenv("WHOOP_TEMPLATES_DIR"); td != "" {
		return td
	}
	// Try relative to cwd first (development).
	if _, err := os.Stat("templates"); err == nil {
		return "templates"
	}
	// Fall back to next to binary.
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "templates")
}

// --- Subcommands ---

func runAuth() {
	if err := auth.StartAuthFlow(); err != nil {
		fmt.Fprintln(os.Stderr, "auth failed:", err)
		os.Exit(1)
	}
}

func runDaily(args []string) {
	fs := flag.NewFlagSet("daily", flag.ExitOnError)
	dateStr := fs.String("date", "", "date in YYYY-MM-DD format (default: today)")
	_ = fs.Parse(args)

	date, err := parseDate(*dateStr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	c, err := getClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("Fetching data for %s...\n", date.Format("2006-01-02"))
	dayData, err := fetch.GetDayData(c, date)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fetch error:", err)
		os.Exit(1)
	}

	tmplPath := filepath.Join(templatesDir(), "daily.md.tmpl")
	content, err := render.RenderDaily(dayData, tmplPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "render error:", err)
		os.Exit(1)
	}

	dir, err := ensureOutputDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	yearDir, err := ensureYearDir(dir, date.Year())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	outPath := filepath.Join(yearDir, fmt.Sprintf("daily-%s.md", date.Format("2006-01-02")))
	if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
		fmt.Fprintln(os.Stderr, "write error:", err)
		os.Exit(1)
	}

	fmt.Println("Written:", outPath)
}

func runWeekly(args []string) {
	fs := flag.NewFlagSet("weekly", flag.ExitOnError)
	dateStr := fs.String("date", "", "any date within the target week (default: this week)")
	_ = fs.Parse(args)

	date, err := parseDate(*dateStr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Find Monday of the week.
	weekday := int(date.Weekday())
	if weekday == 0 {
		weekday = 7 // treat Sunday as day 7
	}
	monday := date.AddDate(0, 0, -(weekday - 1))
	monday = time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, monday.Location())
	sunday := monday.AddDate(0, 0, 7)

	c, err := getClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("Fetching week %s → %s...\n", monday.Format("2006-01-02"), sunday.AddDate(0, 0, -1).Format("2006-01-02"))

	today := time.Now()
	var days []fetch.DayData
	for d := monday; d.Before(sunday); d = d.AddDate(0, 0, 1) {
		if d.After(today) {
			days = append(days, fetch.DayData{Date: d})
			continue
		}
		dayData, err := fetch.GetDayData(c, d)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not fetch %s: %v\n", d.Format("2006-01-02"), err)
			dayData = fetch.DayData{Date: d}
		}
		days = append(days, dayData)
	}

	stats := render.BuildWeekStats(days)
	tmplPath := filepath.Join(templatesDir(), "weekly.md.tmpl")
	content, err := render.RenderWeeklyFromStats(stats, tmplPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "render error:", err)
		os.Exit(1)
	}

	dir, err := ensureOutputDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	isoYear, isoWeek := monday.ISOWeek()
	yearDir, err := ensureYearDir(dir, isoYear)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	outPath := filepath.Join(yearDir, fmt.Sprintf("weekly-%d-W%02d.md", isoYear, isoWeek))
	if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
		fmt.Fprintln(os.Stderr, "write error:", err)
		os.Exit(1)
	}

	fmt.Println("Written:", outPath)
}

func runPersona(args []string) {
	fs := flag.NewFlagSet("persona", flag.ExitOnError)
	days := fs.Int("days", 30, "number of days to include")
	_ = fs.Parse(args)

	c, err := getClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	end := time.Now()
	start := end.AddDate(0, 0, -(*days))

	fmt.Printf("Fetching %d days of data (%s → %s)...\n",
		*days, start.Format("2006-01-02"), end.Format("2006-01-02"))

	var dayData []fetch.DayData
	for d := start; d.Before(end); d = d.AddDate(0, 0, 1) {
		dd, err := fetch.GetDayData(c, d)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not fetch %s: %v\n", d.Format("2006-01-02"), err)
			dd = fetch.DayData{Date: d}
		}
		dayData = append(dayData, dd)
	}

	content, err := render.RenderPersonaSection(dayData)
	if err != nil {
		fmt.Fprintln(os.Stderr, "render error:", err)
		os.Exit(1)
	}

	fmt.Println(content)
}

func runFetchAll(args []string) {
	fs := flag.NewFlagSet("fetch-all", flag.ExitOnError)
	days := fs.Int("days", 30, "number of days to fetch")
	_ = fs.Parse(args)

	c, err := getClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	dir, err := ensureOutputDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	tmplPath := filepath.Join(templatesDir(), "daily.md.tmpl")
	end := time.Now()
	start := end.AddDate(0, 0, -(*days))

	fmt.Printf("Fetching and writing %d daily notes...\n", *days)

	for d := start; d.Before(end); d = d.AddDate(0, 0, 1) {
		dayData, err := fetch.GetDayData(c, d)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not fetch %s: %v\n", d.Format("2006-01-02"), err)
			continue
		}
		if dayData.Cycle == nil {
			fmt.Printf("Skipped: %s (no data)\n", d.Format("2006-01-02"))
			time.Sleep(500 * time.Millisecond)
			continue
		}

		content, err := render.RenderDaily(dayData, tmplPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not render %s: %v\n", d.Format("2006-01-02"), err)
			continue
		}

		yearDir, err := ensureYearDir(dir, d.Year())
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not create year dir for %s: %v\n", d.Format("2006-01-02"), err)
			continue
		}

		outPath := filepath.Join(yearDir, fmt.Sprintf("daily-%s.md", d.Format("2006-01-02")))
		if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not write %s: %v\n", outPath, err)
			continue
		}

		fmt.Println("Written:", outPath)
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println("Done.")
}
