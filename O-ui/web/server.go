package web

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"

	"github.com/Hhz0823/oiwest-core/O-ui/web/api"
	"github.com/Hhz0823/oiwest-core/O-ui/web/middleware"
)

//go:embed all:frontend
var frontendFS embed.FS

func StartServer(port int) error {
	mux := http.NewServeMux()

	// Public
	mux.HandleFunc("POST /api/login", api.Login)
	mux.HandleFunc("GET /sub/{token}", api.GetSubscription)

	// Auth required
	auth := func(h http.HandlerFunc) http.Handler { return middleware.AuthMiddleware(http.HandlerFunc(h)) }

	mux.Handle("GET /api/user/info", auth(api.GetUserInfo))
	mux.Handle("POST /api/user/password", auth(api.ChangePassword))
	mux.Handle("GET /api/inbounds", auth(api.GetInbounds))
	mux.Handle("POST /api/inbounds", auth(api.AddInbound))
	mux.Handle("PUT /api/inbounds/{id}", auth(api.UpdateInbound))
	mux.Handle("DELETE /api/inbounds/{id}", auth(api.DeleteInbound))
	mux.Handle("GET /api/outbounds", auth(api.GetOutbounds))
	mux.Handle("POST /api/outbounds", auth(api.AddOutbound))
	mux.Handle("DELETE /api/outbounds/{id}", auth(api.DeleteOutbound))
	mux.Handle("GET /api/nodes", auth(api.GetNodes))
	mux.Handle("POST /api/nodes", auth(api.AddNode))
	mux.Handle("PUT /api/nodes/{id}", auth(api.UpdateNode))
	mux.Handle("DELETE /api/nodes/{id}", auth(api.DeleteNode))
	mux.Handle("POST /api/nodes/{id}/check", auth(api.CheckNode))
	mux.Handle("GET /api/stats", auth(api.GetStats))
	mux.Handle("GET /api/stats/traffic", auth(api.GetTrafficHistory))
	mux.Handle("GET /api/system", auth(api.GetSystemInfo))
	mux.Handle("GET /api/core/status", auth(api.GetCoreStatus))
	mux.Handle("POST /api/core/start", auth(api.StartCore))
	mux.Handle("POST /api/core/stop", auth(api.StopCore))
	mux.Handle("POST /api/core/restart", auth(api.RestartCore))
	mux.Handle("GET /api/settings", auth(api.GetSettings))
	mux.Handle("PUT /api/settings", auth(api.UpdateSettings))

	// Frontend
	frontend, _ := fs.Sub(frontendFS, "frontend")
	fileServer := http.FileServer(http.FS(frontend))
	mux.Handle("/", fileServer)

	handler := middleware.CORS(mux)

	addr := fmt.Sprintf(":%d", port)
	log.Printf("[O-ui] Panel running on http://0.0.0.0%s", addr)
	return http.ListenAndServe(addr, handler)
}
