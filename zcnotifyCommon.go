package main

import (
	"fmt"
	"github.com/grandcat/zeroconf"
	"time"
)

type ServiceChangeType int

const (
	ADD    ServiceChangeType = iota
	REMOVE                   = iota
	MODIFY                   = iota
)

func (sct ServiceChangeType) MarshalJSON() ([]byte, error) {
	var bytes []byte
	switch sct {
	case ADD:
		bytes = []byte(`"ADD"`)
		break
	case REMOVE:
		bytes = []byte(`"REMOVE"`)
		break
	case MODIFY:
		bytes = []byte(`"MODIFY"`)
		break
	default:
		panic(fmt.Sprintf("unknown service change type %s", sct))
	}

	return bytes, nil
}

func (sct ServiceChangeType) String() string {
	var sctStr string
	switch sct {
	case ADD:
		sctStr = "ADD"
		break
	case REMOVE:
		sctStr = "REMOVE"
		break
	case MODIFY:
		sctStr = "MODIFY"
		break
	default:
		panic(fmt.Sprintf("unknown service change type %s", sct))
	}

	return sctStr
}

// ServiceEntryChange is a type which encapsulates information about a group
// member along with the type of change and the time at which the event occured
// on the network.
type ServiceEntryChange struct {
	ChangeType ServiceChangeType     `json:"changeType"`
	Timestamp  time.Time             `json:"timestamp"`
	Entry      zeroconf.ServiceEntry `json:"entry"`
}

func (sec ServiceEntryChange) String() string {
	return fmt.Sprintf("Service %s %q @ %s: (h: %s, 4: %s, 6: %s, ttl: %d)",
		sec.ChangeType.String(),
		sec.Entry.Instance,
		sec.Timestamp.Format(time.RFC3339),
		sec.Entry.HostName,
		sec.Entry.AddrIPv4,
		sec.Entry.AddrIPv6,
		sec.Entry.TTL)
}
