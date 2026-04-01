package api

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	fws "github.com/gofiber/websocket/v2"
	"github.com/novaacs/go-acs/internal/events"
	"go.uber.org/zap"
)

func (h *Handler) WSUpgrade(c *fiber.Ctx) error {
	if !fws.IsWebSocketUpgrade(c) {
		return fiber.NewError(426, "websocket upgrade required")
	}
	return c.Next()
}

func (h *Handler) WSHandler(hub *events.Hub) fiber.Handler {
	return fws.New(func(conn *fws.Conn) {
		tokenStr := conn.Query("token")
		if tokenStr == "" {
			tokenStr = strings.TrimPrefix(conn.Headers("Authorization"), "Bearer ")
		}

		if tokenStr == "" {
			_ = conn.WriteMessage(fws.TextMessage, []byte(`{"error":"token required"}`))
			_ = conn.Close()
			return
		}

		claims, err := h.authSvc.ValidateAccessToken(tokenStr)
		if err != nil {
			_ = conn.WriteMessage(fws.TextMessage, []byte(`{"error":"invalid token"}`))
			_ = conn.Close()
			return
		}

		h.log.Info("ws client authenticated",
			zap.String("tenant_id", claims.TenantID),
			zap.String("email", claims.Email),
		)

		_ = conn.WriteJSON(fiber.Map{
			"type":      "connected",
			"tenant_id": claims.TenantID,
			"message":   "NovaACS event stream ready",
		})

		hub.ServeWS(conn, claims.TenantID)
	})
}
