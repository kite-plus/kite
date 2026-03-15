package service

import "github.com/amigoer/kite-blog/internal/config"

type SystemService struct {
	cfg *config.Config
}

func NewSystemService(cfg *config.Config) *SystemService {
	return &SystemService{cfg: cfg}
}

func (s *SystemService) HealthStatus() map[string]string {
	renderMode := config.RenderModeClassic
	if s != nil && s.cfg != nil && s.cfg.RenderMode != "" {
		renderMode = s.cfg.RenderMode
	}

	return map[string]string{
		"status":      "ok",
		"render_mode": renderMode,
	}
}
