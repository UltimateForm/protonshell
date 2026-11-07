// Package vpn drives the official protonvpn CLI.
package vpn

import (
	"context"
	"errors"
	"os/exec"
	"strings"
)

const bin = "protonvpn"

// Status is the parsed output of `protonvpn status`.
type Status struct {
	Connected bool
	// Server is the logical server name, e.g. "ES#220".
	Server   string
	Location string
	Protocol string
}

func GetStatus(ctx context.Context) (Status, error) {
	out, err := run(ctx, "status")
	if err != nil {
		return Status{}, err
	}
	return parseStatus(out), nil
}

func Connect(ctx context.Context, server string) (string, error) {
	return run(ctx, "connect", server)
}

func Disconnect(ctx context.Context) (string, error) {
	return run(ctx, "disconnect")
}

func parseStatus(out string) Status {
	var st Status
	for _, line := range strings.Split(out, "\n") {
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		val = strings.TrimSpace(val)
		switch strings.TrimSpace(key) {
		case "Status":
			st.Connected = val == "Connected"
		case "Server":
			// "ES#220 in Madrid, Spain"
			st.Server, st.Location, _ = strings.Cut(val, " in ")
		case "Protocol":
			st.Protocol = val
		}
	}
	return st
}

func run(ctx context.Context, args ...string) (string, error) {
	b, err := exec.CommandContext(ctx, bin, args...).CombinedOutput()
	out := strings.TrimSpace(string(b))
	if err != nil {
		if out != "" {
			return out, errors.New(lastLine(out))
		}
		return out, err
	}
	return out, nil
}

func lastLine(s string) string {
	if i := strings.LastIndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[i+1:])
	}
	return s
}
