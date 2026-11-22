package device

import (
	"errors"
	"fmt"
	"time"

	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

type SerialSystem struct{}

func (SerialSystem) ListPorts() ([]PortInfo, error) {
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return nil, fmt.Errorf("enumerate serial ports: %w", err)
	}
	result := make([]PortInfo, 0, len(ports))
	for _, port := range ports {
		result = append(result, PortInfo{Device: port.Name, VID: port.VID})
	}
	return result, nil
}

func (SerialSystem) Open(name string) (Port, error) {
	port, err := serial.Open(name, &serial.Mode{
		BaudRate: 115200,
		InitialStatusBits: &serial.ModemOutputBits{
			DTR: false,
			RTS: false,
		},
	})
	if err != nil {
		return nil, err
	}
	if err := port.SetReadTimeout(time.Millisecond); err != nil {
		return nil, failOpen(port, "set serial read timeout", err)
	}
	if err := port.SetDTR(false); err != nil {
		return nil, failOpen(port, "disable serial DTR", err)
	}
	if err := port.SetRTS(false); err != nil {
		return nil, failOpen(port, "disable serial RTS", err)
	}
	return port, nil
}

func failOpen(port serial.Port, operation string, cause error) error {
	if closeErr := port.Close(); closeErr != nil {
		cause = errors.Join(cause, fmt.Errorf("close partially opened port: %w", closeErr))
	}
	return fmt.Errorf("%s: %w", operation, cause)
}
