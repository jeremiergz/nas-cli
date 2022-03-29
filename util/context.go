package util

type contextKey int

const (
	ContextKeyConsole contextKey = iota
	ContextKeyMedia
	ContextKeySFTP
	ContextKeySSH
)
