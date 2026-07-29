package orgtime

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	isoDateRe     = regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})(?:T(\d{2}):(\d{2})(?::(\d{2}))?)?$`)
	relativeRe    = regexp.MustCompile(`^([+-]?\d+)([dwmyDWMY])$`)
	orgTimestampRe = regexp.MustCompile(`^[<\[](\d{4})-(\d{2})-(\d{2})(?:\s+\w+)?(?:\s+(\d{2}):(\d{2}))?[>\]]$`)
)

type Timestamp struct {
	Time     time.Time
	HasTime  bool
	Active   bool
}

func (ts Timestamp) String() string {
	return ts.Format(ts.Active)
}

func (ts Timestamp) Format(active bool) string {
	open, close := "[", "]"
	if active {
		open, close = "<", ">"
	}
	dow := ts.Time.Weekday().String()[:3]
	if ts.HasTime {
		return fmt.Sprintf("%s%s %s %02d:%02d%s",
			open,
			ts.Time.Format("2006-01-02"),
			dow,
			ts.Time.Hour(),
			ts.Time.Minute(),
			close,
		)
	}
	return fmt.Sprintf("%s%s %s%s", open, ts.Time.Format("2006-01-02"), dow, close)
}

func Parse(input string, anchor time.Time) (Timestamp, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return Timestamp{}, fmt.Errorf("empty date input")
	}

	loc := anchor.Location()

	if ts, ok := parseKeyword(input, anchor); ok {
		return ts, nil
	}

	if ts, ok := parseRelative(input, anchor); ok {
		return ts, nil
	}

	if ts, ok := parseISO(input, loc); ok {
		return ts, nil
	}

	if ts, ok := parseOrgTimestamp(input, loc); ok {
		return ts, nil
	}

	return Timestamp{}, fmt.Errorf("unrecognized date format: %s", input)
}

func parseKeyword(input string, anchor time.Time) (Timestamp, bool) {
	today := time.Date(anchor.Year(), anchor.Month(), anchor.Day(), 0, 0, 0, 0, anchor.Location())

	switch strings.ToLower(input) {
	case "today", "now":
		return Timestamp{Time: today, Active: true}, true
	case "tomorrow":
		return Timestamp{Time: today.AddDate(0, 0, 1), Active: true}, true
	case "yesterday":
		return Timestamp{Time: today.AddDate(0, 0, -1), Active: true}, true
	}
	return Timestamp{}, false
}

func parseRelative(input string, anchor time.Time) (Timestamp, bool) {
	m := relativeRe.FindStringSubmatch(input)
	if m == nil {
		return Timestamp{}, false
	}

	n, _ := strconv.Atoi(m[1])
	unit := strings.ToLower(m[2])
	today := time.Date(anchor.Year(), anchor.Month(), anchor.Day(), 0, 0, 0, 0, anchor.Location())

	var t time.Time
	switch unit {
	case "d":
		t = today.AddDate(0, 0, n)
	case "w":
		t = today.AddDate(0, 0, n*7)
	case "m":
		t = today.AddDate(0, n, 0)
	case "y":
		t = today.AddDate(n, 0, 0)
	default:
		return Timestamp{}, false
	}
	return Timestamp{Time: t, Active: true}, true
}

func parseISO(input string, loc *time.Location) (Timestamp, bool) {
	m := isoDateRe.FindStringSubmatch(input)
	if m == nil {
		return Timestamp{}, false
	}

	year, _ := strconv.Atoi(m[1])
	month, _ := strconv.Atoi(m[2])
	day, _ := strconv.Atoi(m[3])

	hour, minute := 0, 0
	hasTime := false
	if m[4] != "" {
		hour, _ = strconv.Atoi(m[4])
		minute, _ = strconv.Atoi(m[5])
		hasTime = true
	}

	t := time.Date(year, time.Month(month), day, hour, minute, 0, 0, loc)
	return Timestamp{Time: t, HasTime: hasTime, Active: true}, true
}

func parseOrgTimestamp(input string, loc *time.Location) (Timestamp, bool) {
	active := strings.HasPrefix(input, "<")
	m := orgTimestampRe.FindStringSubmatch(input)
	if m == nil {
		return Timestamp{}, false
	}

	year, _ := strconv.Atoi(m[1])
	month, _ := strconv.Atoi(m[2])
	day, _ := strconv.Atoi(m[3])

	hour, minute := 0, 0
	hasTime := false
	if m[4] != "" {
		hour, _ = strconv.Atoi(m[4])
		minute, _ = strconv.Atoi(m[5])
		hasTime = true
	}

	t := time.Date(year, time.Month(month), day, hour, minute, 0, 0, loc)
	return Timestamp{Time: t, HasTime: hasTime, Active: active}, true
}

func Now() Timestamp {
	return Timestamp{Time: time.Now(), HasTime: true, Active: false}
}

func NowInactive() Timestamp {
	return Timestamp{Time: time.Now(), HasTime: true, Active: false}
}

func FormatStateChange(newState, oldState string, ts Timestamp) string {
	fromPart := ""
	if oldState != "" {
		fromPart = fmt.Sprintf("from %-13q", oldState)
	} else {
		fromPart = "from \"\"           "
	}
	return fmt.Sprintf("- State %-13q %s %s", newState, fromPart, ts.Format(false))
}
