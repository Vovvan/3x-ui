package model

// ClientIpEntry is the wire/JSON shape of a single observed client IP with the
// last time it was seen (unix seconds).
type ClientIpEntry struct {
	IP        string `json:"ip"`
	Timestamp int64  `json:"timestamp"`
}
