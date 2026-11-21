package tray

import (
	"errors"

	"github.com/gravity981/agent-cody/bridge-go/internal/httpapi"
)

func OpenCody(launch func(string) error) error {
	if launch == nil {
		return errors.New("browser launcher is required")
	}
	return launch(httpapi.SiteOrigin + "/")
}
