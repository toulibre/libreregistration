package middleware

import (
	"net/http"
)

func SetFlash(w http.ResponseWriter, r *http.Request, kind string, message string) {
	session := GetSession(r)
	if session == nil {
		return
	}
	session.AddFlash(message, kind)
	session.Save(r, w)
}

func GetFlashes(w http.ResponseWriter, r *http.Request, kind string) []string {
	session := GetSession(r)
	if session == nil {
		return nil
	}
	flashes := session.Flashes(kind)
	if len(flashes) > 0 {
		session.Save(r, w)
	}
	var messages []string
	for _, f := range flashes {
		if s, ok := f.(string); ok {
			messages = append(messages, s)
		}
	}
	return messages
}
