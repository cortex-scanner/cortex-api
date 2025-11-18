package scanner

import (
	"cortex/repository"
)

type Protocol string

const (
	ProtocolTCP Protocol = "tcp"
	ProtocolUDP Protocol = "udp"
)

func AttachPortData(finding *repository.AssetFinding, port int, protocol Protocol) {
	finding.Type = repository.FindingTypePort
	finding.Data = map[string]any{
		"port":     port,
		"protocol": protocol,
	}
}
