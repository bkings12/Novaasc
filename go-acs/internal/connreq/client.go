package connreq

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/icholy/digest"
	"github.com/novaacs/go-acs/internal/credprofile"
	"github.com/novaacs/go-acs/internal/device"
	"github.com/novaacs/go-acs/internal/events"
	"github.com/novaacs/go-acs/internal/tenant"
	"go.uber.org/zap"
)

type Client struct {
	deviceRepo device.Repository
	tenantRepo tenant.Repository
	resolver   *credprofile.Resolver
	hub        *events.Hub
	log        *zap.Logger
}

func NewClient(
	deviceRepo device.Repository,
	tenantRepo tenant.Repository,
	resolver *credprofile.Resolver,
	hub *events.Hub,
	log *zap.Logger,
) *Client {
	return &Client{
		deviceRepo: deviceRepo,
		tenantRepo: tenantRepo,
		resolver:   resolver,
		hub:        hub,
		log:        log,
	}
}

type WakeResult struct {
	Serial     string    `json:"serial"`
	Success    bool      `json:"success"`
	StatusCode int       `json:"status_code,omitempty"`
	CredSource string    `json:"cred_source,omitempty"`
	Error      string    `json:"error,omitempty"`
	SentAt     time.Time `json:"sent_at"`
}

func (c *Client) Wake(ctx context.Context, tenantID, serial, bodyUsername, bodyPassword string) (*WakeResult, error) {
	result := &WakeResult{
		Serial: serial,
		SentAt: time.Now(),
	}

	dev, err := c.deviceRepo.GetBySerial(ctx, tenantID, serial)
	if err != nil {
		if errors.Is(err, device.ErrNotFound) {
			result.Error = "device not registered — use pre-registration endpoint"
			return result, fmt.Errorf("device not registered: %w", err)
		}
		result.Error = "device not found"
		return result, fmt.Errorf("device not found: %w", err)
	}

	crURL := firstNonEmpty(
		dev.GetParameter("Device.ManagementServer.ConnectionRequestURL"),
		dev.GetParameter("InternetGatewayDevice.ManagementServer.ConnectionRequestURL"),
		dev.ConnectionRequestURL,
	)
	if crURL == "" {
		result.Error = "device has no ConnectionRequestURL"
		return result, fmt.Errorf("no ConnectionRequestURL for %s", serial)
	}

	t, _ := c.tenantRepo.GetByID(ctx, tenantID)
	creds := c.resolver.Resolve(ctx, dev, t, bodyUsername, bodyPassword)
	result.CredSource = creds.Source
	username := creds.Username
	password := creds.Password
	usedSerialFallback := creds.Source == "serial_fallback"

	c.log.Info("connection request",
		zap.String("tenant_id", tenantID),
		zap.String("serial", serial),
		zap.String("url", crURL),
		zap.String("cred_source", creds.Source),
	)

	// New base transport per attempt so digest.Transport's internal retry (401 → Digest) uses a clean transport.
	newBaseTransport := func() *http.Transport {
		return &http.Transport{
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
			DisableKeepAlives: true,
		}
	}

	doRequest := func(useAuth bool) (*http.Response, error) {
		base := newBaseTransport()
		var transport http.RoundTripper = base
		if useAuth && username != "" {
			transport = &digest.Transport{
				Username:  username,
				Password:  password,
				Transport: base,
			}
			c.log.Debug("connection request using Digest auth", zap.String("serial", serial), zap.String("username", username))
		}
		httpClient := &http.Client{Timeout: 15 * time.Second, Transport: transport}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, crURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "NovaACS/1.0")
		return httpClient.Do(req)
	}

	c.log.Info("sending connection request",
		zap.String("tenant_id", tenantID),
		zap.String("serial", serial),
		zap.String("url", crURL),
		zap.Bool("has_credentials", username != ""),
		zap.Bool("path_only_first", usedSerialFallback),
	)

	// When device has no connection request username (e.g. MikroTik empty): try no-auth first (path-only secret), then serial/serial on 401.
	// When device or API provided credentials: try with auth first, then no-auth on 401 (path-only).
	var resp *http.Response
	if usedSerialFallback {
		resp, err = doRequest(false)
		if err == nil && resp.StatusCode == http.StatusUnauthorized {
			resp.Body.Close()
			c.log.Debug("connection request 401 without auth, retrying with serial as credentials",
				zap.String("serial", serial),
			)
			resp, err = doRequest(true)
		}
	} else {
		resp, err = doRequest(true)
		if err == nil && resp.StatusCode == http.StatusUnauthorized {
			resp.Body.Close()
			c.log.Debug("connection request 401 with auth, retrying without credentials (path-only secret)",
				zap.String("serial", serial),
			)
			resp, err = doRequest(false)
		}
	}
	if err != nil {
		result.Error = err.Error()
		c.log.Warn("connection request failed",
			zap.String("serial", serial),
			zap.Error(err),
		)
		return result, fmt.Errorf("connection request: %w", err)
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		result.Success = true
		c.log.Info("connection request acknowledged",
			zap.String("serial", serial),
			zap.Int("status", resp.StatusCode),
		)
		if c.hub != nil {
			c.hub.Broadcast(tenantID, events.EventDeviceOnline, map[string]string{
				"serial":  serial,
				"trigger": "connection_request",
			})
		}
	} else {
		if resp.StatusCode == http.StatusUnauthorized {
			if username == "" {
				result.Error = "device returned 401 — provide ConnectionRequestUsername/ConnectionRequestPassword (device must send them in Inform, or set on device)"
			} else {
				result.Error = fmt.Sprintf("device returned 401 — check ConnectionRequestUsername/ConnectionRequestPassword for serial %s", serial)
			}
		} else {
			result.Error = fmt.Sprintf("unexpected status %d", resp.StatusCode)
		}
		c.log.Warn("connection request unexpected status",
			zap.String("serial", serial),
			zap.Int("status", resp.StatusCode),
			zap.Bool("had_credentials", username != ""),
		)
	}

	return result, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
