package common

type DatabaseArgMarkers int

const (
	PositionalDbArgs DatabaseArgMarkers = iota
	NumberedDbArgs
)
