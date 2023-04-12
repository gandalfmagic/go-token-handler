package sessions

const (
	sessionIdName    = "session_id"
	sessionStateName = "state"
)

type contextKey int

const (
	ContextKeyAccessTokenName contextKey = iota
)
