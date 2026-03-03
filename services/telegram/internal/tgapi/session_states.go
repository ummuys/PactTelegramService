package tgapi

type SessionState int

const (
	StateNeedPassword SessionState = iota
	StateAuthSuccessful
)
