package main

import (
	"encoding/base64"
	_ "embed"
)

//go:embed assets/hr.png
var hrFlagBytes []byte

//go:embed assets/gb.png
var gbFlagBytes []byte

//go:embed assets/palantir-saruman.gif
var sarumanGifBytes []byte

func (a *App) GetHrFlag() string {
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(hrFlagBytes)
}

func (a *App) GetGbFlag() string {
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(gbFlagBytes)
}

func (a *App) GetSarumanGif() string {
	return "data:image/gif;base64," + base64.StdEncoding.EncodeToString(sarumanGifBytes)
}
