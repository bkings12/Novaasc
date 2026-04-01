package credprofile

import (
	"context"

	"github.com/novaacs/go-acs/internal/device"
	"github.com/novaacs/go-acs/internal/tenant"
	"go.uber.org/zap"
)

type Resolver struct {
	profiles   Repository
	tenantRepo tenant.Repository
	log        *zap.Logger
}

func NewResolver(profiles Repository, tenantRepo tenant.Repository, log *zap.Logger) *Resolver {
	return &Resolver{
		profiles:   profiles,
		tenantRepo: tenantRepo,
		log:        log,
	}
}

// Resolve returns the best credentials for a connection request.
// Priority: body > device stored > OUI profile > manufacturer profile > tenant default > serial fallback.
func (r *Resolver) Resolve(
	ctx context.Context,
	dev *device.Device,
	t *tenant.Tenant,
	bodyUsername, bodyPassword string,
) *ResolvedCredentials {

	if bodyUsername != "" {
		r.log.Debug("using body credentials", zap.String("serial", dev.SerialNumber))
		return &ResolvedCredentials{
			Username: bodyUsername,
			Password: bodyPassword,
			Source:   "body",
		}
	}

	if dev.ConnectionRequestUsername != "" {
		r.log.Debug("using device stored credentials", zap.String("serial", dev.SerialNumber))
		return &ResolvedCredentials{
			Username: dev.ConnectionRequestUsername,
			Password: dev.ConnectionRequestPassword,
			Source:   "device",
		}
	}

	if dev.OUI != "" {
		if p, err := r.profiles.FindByOUI(ctx, dev.TenantID, dev.OUI); err == nil && p != nil {
			r.log.Debug("using OUI profile credentials",
				zap.String("serial", dev.SerialNumber),
				zap.String("oui", dev.OUI),
				zap.String("profile", p.Name))
			return &ResolvedCredentials{
				Username: p.CRUsername,
				Password: p.CRPassword,
				Source:   "oui_profile:" + p.Name,
			}
		}
	}

	if dev.Manufacturer != "" {
		if p, err := r.profiles.FindByManufacturer(ctx, dev.TenantID, dev.Manufacturer); err == nil && p != nil {
			r.log.Debug("using manufacturer profile credentials",
				zap.String("serial", dev.SerialNumber),
				zap.String("manufacturer", dev.Manufacturer),
				zap.String("profile", p.Name))
			return &ResolvedCredentials{
				Username: p.CRUsername,
				Password: p.CRPassword,
				Source:   "manufacturer_profile:" + p.Name,
			}
		}
	}

	if t != nil && t.DefaultCRUsername != "" {
		r.log.Debug("using tenant default credentials",
			zap.String("serial", dev.SerialNumber),
			zap.String("tenant", t.Slug))
		return &ResolvedCredentials{
			Username: t.DefaultCRUsername,
			Password: t.DefaultCRPassword,
			Source:   "tenant_default",
		}
	}

	r.log.Debug("using serial fallback credentials", zap.String("serial", dev.SerialNumber))
	return &ResolvedCredentials{
		Username: dev.SerialNumber,
		Password: dev.SerialNumber,
		Source:   "serial_fallback",
	}
}
