package handlers

import (
	"encoding/json"
	"strings"
)

func jsonObjectOrEmpty(b []byte) json.RawMessage {
	s := strings.TrimSpace(string(b))
	if s == "" || s == "null" {
		return json.RawMessage(`{}`)
	}
	if !json.Valid(b) {
		return json.RawMessage(`{}`)
	}
	return json.RawMessage(b)
}

func jsonArrayOrEmpty(b []byte) json.RawMessage {
	s := strings.TrimSpace(string(b))
	if s == "" || s == "null" {
		return json.RawMessage(`[]`)
	}
	if !json.Valid(b) || !strings.HasPrefix(s, "[") {
		return json.RawMessage(`[]`)
	}
	return json.RawMessage(b)
}
