package tray

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image/color"

	"github.com/gravity981/agent-cody/bridge-go/internal/bridge"
	"github.com/gravity981/agent-cody/bridge-go/internal/sessions"
)

const iconSize = 32

func ColorForState(state sessions.State) (color.RGBA, bool) {
	switch state {
	case sessions.IDLE, sessions.WORKING:
		return color.RGBA{B: 255, A: 255}, true
	case sessions.THINKING:
		return color.RGBA{R: 255, G: 160, A: 255}, true
	case sessions.APPROVAL:
		return color.RGBA{R: 255, G: 60, A: 255}, true
	case sessions.COMPLETED:
		return color.RGBA{G: 255, A: 255}, true
	case sessions.ERROR:
		return color.RGBA{R: 255, A: 255}, true
	default:
		return color.RGBA{}, false
	}
}

func Icon(state sessions.State) ([]byte, error) {
	color, ok := ColorForState(state)
	if !ok {
		return nil, fmt.Errorf("no tray color for state %q", state)
	}

	maskSize := iconSize / 8 * iconSize
	imageSize := 40 + iconSize*iconSize*4 + maskSize
	file := make([]byte, 22+imageSize)
	binary.LittleEndian.PutUint16(file[2:4], 1)
	binary.LittleEndian.PutUint16(file[4:6], 1)
	entry := file[6:22]
	entry[0], entry[1] = iconSize, iconSize
	binary.LittleEndian.PutUint16(entry[4:6], 1)
	binary.LittleEndian.PutUint16(entry[6:8], 32)
	binary.LittleEndian.PutUint32(entry[8:12], uint32(imageSize))
	binary.LittleEndian.PutUint32(entry[12:16], 22)

	dib := file[22:]
	binary.LittleEndian.PutUint32(dib[0:4], 40)
	binary.LittleEndian.PutUint32(dib[4:8], iconSize)
	binary.LittleEndian.PutUint32(dib[8:12], iconSize*2)
	binary.LittleEndian.PutUint16(dib[12:14], 1)
	binary.LittleEndian.PutUint16(dib[14:16], 32)
	binary.LittleEndian.PutUint32(dib[20:24], uint32(iconSize*iconSize*4))

	pixels := dib[40 : 40+iconSize*iconSize*4]
	mask := dib[40+iconSize*iconSize*4:]
	center, radius := iconSize/2, iconSize/2-2
	for y := 0; y < iconSize; y++ {
		for x := 0; x < iconSize; x++ {
			dx, dy := x-center, y-center
			offset := ((iconSize-1-y)*iconSize + x) * 4
			if dx*dx+dy*dy <= radius*radius {
				pixels[offset] = color.B
				pixels[offset+1] = color.G
				pixels[offset+2] = color.R
				pixels[offset+3] = 255
				continue
			}
			maskRow := (iconSize - 1 - y) * (iconSize / 8)
			mask[maskRow+x/8] |= 1 << (7 - x%8)
		}
	}
	return file, nil
}

func Tooltip(status bridge.Status) string {
	device := "not connected"
	if status.Device.Connected && status.Device.Port != nil {
		device = "connected: " + *status.Device.Port
	}
	return fmt.Sprintf(
		"Cody %s | %s | %s | %d sessions",
		status.Bridge.Version,
		status.State,
		device,
		len(status.Sessions),
	)
}

func ValidateIcon(icon []byte) error {
	if len(icon) < 22 || binary.LittleEndian.Uint16(icon[2:4]) != 1 ||
		binary.LittleEndian.Uint16(icon[4:6]) != 1 {
		return errors.New("invalid ICO header")
	}
	entry := icon[6:22]
	if entry[0] != iconSize || entry[1] != iconSize ||
		binary.LittleEndian.Uint16(entry[6:8]) != 32 ||
		binary.LittleEndian.Uint32(entry[12:16]) != 22 {
		return errors.New("invalid ICO entry")
	}
	if uint32(len(icon)-22) != binary.LittleEndian.Uint32(entry[8:12]) {
		return errors.New("invalid ICO image size")
	}
	return nil
}
