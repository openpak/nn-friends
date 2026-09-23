// Package crosspresence connects Wii U and 3DS presence to the rest of
// OpenPak through the account core's live-session registry: people signed in
// here are published as core sessions (namespace wiiu/3ds, with the title),
// and friends live somewhere else -- a Switch, Ryujinx, the other of Wii U and
// 3DS -- are rendered online here with a line saying where.
package crosspresence

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// ClientLabel names what somebody is on: the client when one is known, the
// platform otherwise. The same table lives in nx-baas and the website.
func ClientLabel(namespace, client string) string {
	switch strings.ToLower(client) {
	case "":
		switch strings.ToLower(namespace) {
		case "":
			return ""
		case "switch":
			return "Switch"
		case "wiiu":
			return "Wii U"
		case "3ds":
			return "3DS"
		default:
			return strings.ToUpper(namespace[:1]) + namespace[1:]
		}
	case "switch":
		return "Switch"
	case "ryujinx":
		return "Ryujinx"
	case "eden":
		return "Eden"
	case "citron":
		return "Citron"
	case "wiiu":
		return "Wii U"
	case "3ds":
		return "3DS"
	case "cemu":
		return "Cemu"
	case "azahar":
		return "Azahar"
	default:
		return strings.ToUpper(client[:1]) + client[1:]
	}
}

// maxMessageRunes keeps the text inside what a presence message field holds on
// either console. Neither limit is measured; 60 characters is below both
// plausible ones (the 3DS game-mode description is a 128-byte UTF-16 field).
const maxMessageRunes = 60

// Description is the line for a friend live elsewhere: "Playing Kirby on
// Ryujinx", "Playing a game on Switch", "Online on 3DS".
func Description(l Live) string {
	label := ClientLabel(l.Namespace, l.Client)
	if label == "" {
		label = "another console"
	}
	var s string
	switch name := Names.Name(l.Namespace, l.TitleID); {
	case l.TitleID == "":
		s = "Online on " + label
	case name != "":
		s = "Playing " + name + " on " + label
	default:
		s = "Playing a game on " + label
	}
	if utf8.RuneCountInString(s) > maxMessageRunes {
		s = string([]rune(s)[:maxMessageRunes])
	}
	return s
}

// TitleNames resolves title ids to names through the website's catalogue
// (GET /internal/titles/name on WEBSITE_INTERNAL_URL with WEBSITE_INTERNAL_KEY).
// Name never blocks: an unknown id starts one background fetch and answers ""
// until it lands, so a friend list never waits on the website.
type TitleNames struct {
	mu    sync.Mutex
	byKey map[string]*nameEntry
	// Fetch looks one name up; ok=false means the website could not answer.
	Fetch func(namespace, titleID string) (name string, ok bool)
}

type nameEntry struct {
	name string
	at   time.Time
	busy bool
}

const (
	nameHitTTL  = 6 * time.Hour
	nameMissTTL = 10 * time.Minute
)

// Names is the process-wide cache.
var Names = &TitleNames{Fetch: fetchTitleName}

func (t *TitleNames) Name(namespace, titleID string) string {
	if titleID == "" {
		return ""
	}
	key := strings.ToLower(namespace) + "/" + strings.ToUpper(titleID)
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.byKey == nil {
		t.byKey = map[string]*nameEntry{}
	}
	e := t.byKey[key]
	if e == nil {
		e = &nameEntry{}
		t.byKey[key] = e
	}
	ttl := nameHitTTL
	if e.name == "" {
		ttl = nameMissTTL
	}
	if (e.at.IsZero() || time.Since(e.at) > ttl) && !e.busy && t.Fetch != nil {
		e.busy = true
		go func() {
			n, ok := t.Fetch(namespace, titleID)
			t.mu.Lock()
			defer t.mu.Unlock()
			e.busy, e.at = false, time.Now()
			if ok {
				e.name = n
			}
		}()
	}
	return e.name
}

func fetchTitleName(namespace, titleID string) (string, bool) {
	site, key := os.Getenv("WEBSITE_INTERNAL_URL"), os.Getenv("WEBSITE_INTERNAL_KEY")
	if site == "" || key == "" {
		return "", false
	}
	req, err := http.NewRequest(http.MethodGet, site+"/internal/titles/name?namespace="+
		url.QueryEscape(namespace)+"&title_id="+url.QueryEscape(titleID), nil)
	if err != nil {
		return "", false
	}
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", false
	}
	var out struct {
		Name string `json:"name"`
	}
	if json.NewDecoder(resp.Body).Decode(&out) != nil {
		return "", false
	}
	return out.Name, true
}
