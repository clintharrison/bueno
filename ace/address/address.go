// Package address holds MAC-address functions.
package address

import (
	"fmt"
	"strings"
)

type Address struct {
	Bytes [6]uint8
}

func errInvalidAddress(addr string) error {
	return fmt.Errorf("invalid address format: %s", addr)
}

func NewFromString(addr string) (Address, error) {
	parts := make([]string, 6)
	if len(addr) == 17 && strings.Contains(addr, ":") {
		parts = strings.Split(addr, ":")
		if len(parts) != 6 {
			return Address{}, errInvalidAddress(addr)
		}
	} else if len(addr) == 12 && !strings.Contains(addr, ":") {
		for i := 0; i < 6; i++ {
			parts[i] = addr[i*2 : i*2+2]
		}
	} else {
		return Address{}, errInvalidAddress(addr)
	}
	var address [6]uint8
	for i, part := range parts {
		var b byte
		n, err := fmt.Sscanf(part, "%02x", &b)
		if n != 1 || err != nil {
			return Address{}, errInvalidAddress(addr)
		}
		address[i] = b
	}
	return Address{Bytes: address}, nil
}

func (a Address) ToString() string {
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
		a.Bytes[0], a.Bytes[1], a.Bytes[2],
		a.Bytes[3], a.Bytes[4], a.Bytes[5])
}
