package main

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var defaultServerURL = "http://20.250.145.46:8080"

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
		func(msg string) { runtime.EventsEmit(a.ctx, "log", msg) },
		func(peers []PeerStatus) { runtime.EventsEmit(a.ctx, "peers-updated", peers) },
		func(status string) { runtime.EventsEmit(a.ctx, "status-changed", status) },
	)
}

func (a *App) DoLogin(username, password string) error {
	return a.tunnel.Login(username, password)
}

func (a *App) DoRegister(username, password string) error {
	err := a.tunnel.Register(username, password)
	if err != nil {
		return err
	}
	return a.tunnel.Login(username, password)
}

// CreateAndStart kreira mrežu I odmah pokreće tunel
func (a *App) DoCreateAndStart() (*CreateResult, error) {
	result, err := a.tunnel.CreateNetwork()
	if err != nil {
		return nil, err
	}
	err = a.tunnel.Start()
	if err != nil {
		return nil, err
	}
	return result, nil
}

// JoinAndStart pridruži se mreži I odmah pokreće tunel
func (a *App) DoJoinAndStart(code string) (*JoinResult, error) {
	result, err := a.tunnel.JoinNetwork(code)
	if err != nil {
		return nil, err
	}
	err = a.tunnel.Start()
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (a *App) DoLeaveNetwork() {
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

func (a *App) IsInNetwork() bool {
	return a.tunnel.running
}

func (a *App) IsLoggedIn() bool {
	return a.tunnel.Token != ""
}

func (a *App) GetServerURL() string {
	return a.tunnel.ServerURL
}

func (a *App) SetServerURL(url string) {
	a.tunnel.ServerURL = url
}
