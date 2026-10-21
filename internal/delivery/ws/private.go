package ws

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
)

func tableKey(private bool) string {
	if !private {
		return ""
	}
	b := make([]byte, 18)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func (r *Room) Private() bool {
	return r.private
}

func (r *Room) Key() string {
	return r.key
}

func (r *Room) Member(playerID string) bool {
	return r.Holds(playerID) || r.Reserved(playerID) || r.Invited(playerID)
}

func (r *Room) MayEnter(playerID, key string) bool {
	if !r.private || r.Member(playerID) {
		return true
	}
	return key != "" && subtle.ConstantTimeCompare([]byte(key), []byte(r.key)) == 1
}
