// Package servers reads the server list cached on disk by the Proton VPN CLI.
package servers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Feature is the bitmask Proton uses to describe what a server supports.
type Feature int

const (
	SecureCore Feature = 1 << iota
	Tor
	P2P
	Streaming
	IPv6
)

// String renders the features worth showing in a list as short tags.
func (f Feature) String() string {
	var tags []string
	if f&SecureCore != 0 {
		tags = append(tags, "SC")
	}
	if f&Tor != 0 {
		tags = append(tags, "TOR")
	}
	if f&P2P != 0 {
		tags = append(tags, "P2P")
	}
	if f&Streaming != 0 {
		tags = append(tags, "STR")
	}
	return strings.Join(tags, ",")
}

// Physical is one machine behind a logical server.
type Physical struct {
	EntryIP string
	Status  int
}

// Logical is a server as the user sees it, e.g. "ES#220".
type Logical struct {
	Name         string
	EntryCountry string
	ExitCountry  string
	City         string
	Tier         int
	Features     Feature
	Load         int
	// Score is Proton's own ranking; lower is better.
	Score   float64
	Status  int
	Servers []Physical
}

// List mirrors the parts of serverlist.json we care about.
type List struct {
	LogicalServers []Logical
	MaxTier        int
}

// DefaultPath is where the Proton VPN CLI keeps its cached server list.
func DefaultPath() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "serverlist.json"
	}
	// ie ~/.cache/Proton/VPN/serverlist.json
	return filepath.Join(dir, "Proton", "VPN", "serverlist.json")
}

func Load(path string) (*List, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var l List
	if err := json.NewDecoder(f).Decode(&l); err != nil {
		return nil, err
	}
	return &l, nil
}

// Available returns the servers that are online and within the account's tier.
func (l *List) Available() []Logical {
	out := make([]Logical, 0, len(l.LogicalServers))
	for _, s := range l.LogicalServers {
		if s.Status == 1 && s.Tier <= l.MaxTier {
			out = append(out, s)
		}
	}
	return out
}
