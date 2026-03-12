package captcha

import (
	"fmt"
	"math/rand/v2"
	"net/http"
	"strconv"

	"github.com/toulibre/libreregistration/internal/middleware"
)

const sessionKey = "captcha_answer"

// Challenge holds the question text and expected answer for a math captcha.
type Challenge struct {
	Question string
	Answer   int
}

// Generate creates a simple addition captcha and stores the answer in the session.
func Generate(w http.ResponseWriter, r *http.Request) Challenge {
	a := rand.IntN(10) + 1
	b := rand.IntN(10) + 1
	c := Challenge{
		Question: fmt.Sprintf("%d + %d", a, b),
		Answer:   a + b,
	}
	session := middleware.GetSession(r)
	if session != nil {
		session.Values[sessionKey] = c.Answer
		session.Save(r, w)
	}
	return c
}

// Verify checks that the user's answer matches the stored session answer.
// It always clears the stored answer to prevent replay.
func Verify(w http.ResponseWriter, r *http.Request) bool {
	session := middleware.GetSession(r)
	if session == nil {
		return false
	}
	expected, ok := session.Values[sessionKey].(int)
	delete(session.Values, sessionKey)
	session.Save(r, w)
	if !ok {
		return false
	}
	userAnswer, err := strconv.Atoi(r.FormValue("captcha"))
	if err != nil {
		return false
	}
	return userAnswer == expected
}

// IsHoneypotFilled returns true if the honeypot field was filled (likely a bot).
func IsHoneypotFilled(r *http.Request) bool {
	return r.FormValue("website") != ""
}
