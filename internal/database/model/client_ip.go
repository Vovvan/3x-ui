package model

import (
	"cmp"
	"encoding/json"
	"slices"
)

// ClientIpEntry is the wire/JSON shape of a single observed client IP with the
// last time it was seen (unix seconds).
type ClientIpEntry struct {
	IP        string `json:"ip"`
	Timestamp int64  `json:"timestamp"`
}

func SortClientIpEntries(entries []ClientIpEntry) {
	slices.SortFunc(entries, func(a, b ClientIpEntry) int {
		if order := cmp.Compare(a.Timestamp, b.Timestamp); order != 0 {
			return order
		}
		return cmp.Compare(a.IP, b.IP)
	})
}

func MarshalClientIpEntries(entries []ClientIpEntry) (string, error) {
	if len(entries) == 0 {
		return "[]", nil
	}
	SortClientIpEntries(entries)
	encoded, err := json.Marshal(entries)
	return string(encoded), err
}
