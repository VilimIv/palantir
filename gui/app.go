package main

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const defaultServerURL = "http://20.250.145.46:8080"

// App je bridge između Wails frontenda i tunnel logike
type App struct {
	ctx    context.Context
	tunnel *Tunnel
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.tunnel = NewTunnel(
		defaultServerURL,
		func(msg string) {
			runtime.EventsEmit(a.ctx, "log", msg)
		},
		func(peers []PeerStatus) {
			runtime.EventsEmit(a.ctx, "peers-updated", peers)
		},
		func(status string) {
			runtime.EventsEmit(a.ctx, "status-changed", status)
		},
	)
}

// --- Metode expose-ane frontendu ---

func (a *App) DoLogin(username, password string) error {
	return a.tunnel.Login(username, password)
}

func (a *App) DoRegister(username, password string) error {
	err := a.tunnel.Register(username, password)
	if err != nil {
		return err
	}
	// Auto-login nakon registracije
	return a.tunnel.Login(username, password)
}

func (a *App) DoCreateNetwork() (*CreateResult, error) {
	return a.tunnel.CreateNetwork()
}

func (a *App) DoJoinNetwork(code string) (*JoinResult, error) {
	return a.tunnel.JoinNetwork(code)
}

func (a *App) DoStartTunnel() error {
	return a.tunnel.Start()
}

func (a *App) DoStopTunnel() {
	a.tunnel.Stop()
}

func (a *App) DoGetPeers() []PeerStatus {
	return a.tunnel.GetPeers()
}

func (a *App) GetVirtualIP() string {
	return a.tunnel.VirtualIP
}

func (a *App) GetNetworkID() string {
	return a.tunnel.NetworkID
}

func (a *App) GetUsername() string {
	return a.tunnel.Username
}
