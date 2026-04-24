package main

import (
	"embed"
	"log"
	"net/http"

	"github.com/sakagamijun/rawmanga-download-go/internal/download"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()
	assetHandler := app.AssetHandler()

	if err := wails.Run(&options.App{
		Title:     "KLZ9 Downloader",
		Width:     1440,
		Height:    960,
		MinWidth:  1180,
		MinHeight: 760,
		AssetServer: &assetserver.Options{
			Assets:  assets,
			Handler: assetHandler,
			Middleware: func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
					if download.IsLibraryAssetRequest(request.URL.Path) {
						assetHandler.ServeHTTP(writer, request)
						return
					}

					next.ServeHTTP(writer, request)
				})
			},
		},
		BackgroundColour: &options.RGBA{R: 243, G: 247, B: 252, A: 1},
		Mac: &mac.Options{
			TitleBar: mac.TitleBarHidden(),
			Preferences: &mac.Preferences{
				FullscreenEnabled: mac.Enabled,
			},
		},
		OnStartup: app.startup,
		Bind: []interface{}{
			app,
		},
	}); err != nil {
		log.Fatal(err)
	}
}
