package ui

import (
	"context"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/UltimateForm/protonshell/internal/servers"
	"github.com/UltimateForm/protonshell/internal/vpn"
)

const (
	tickInterval  = time.Second
	statusTimeout = 15 * time.Second
	actionTimeout = 60 * time.Second
)

type (
	tickMsg time.Time
	// listMsg carries a freshly loaded list; list is nil when the file on
	// disk has not changed since the last load.
	listMsg struct {
		list *servers.List
		mod  time.Time
		err  error
	}
	statusMsg struct {
		status vpn.Status
		err    error
	}
	actionMsg struct {
		out string
		err error
	}
)

func tick() tea.Cmd {
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// loadList reloads the server list if the file changed after known.
func loadList(path string, known time.Time) tea.Cmd {
	return func() tea.Msg {
		fi, err := os.Stat(path)
		if err != nil {
			return listMsg{err: err}
		}
		if !fi.ModTime().After(known) {
			return listMsg{}
		}
		l, err := servers.Load(path)
		if err != nil {
			return listMsg{err: err}
		}
		return listMsg{list: l, mod: fi.ModTime()}
	}
}

func fetchStatus() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), statusTimeout)
		defer cancel()
		st, err := vpn.GetStatus(ctx)
		return statusMsg{status: st, err: err}
	}
}

func connect(server string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), actionTimeout)
		defer cancel()
		out, err := vpn.Connect(ctx, server)
		return actionMsg{out: out, err: err}
	}
}

func disconnect() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), actionTimeout)
		defer cancel()
		out, err := vpn.Disconnect(ctx)
		return actionMsg{out: out, err: err}
	}
}
