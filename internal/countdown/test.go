package countdown

import (
	"log/slog"
	"net/http"
	"strconv"
	"text/template"
	"time"

	"github.com/joaquinrovira/notes/internal/auth/token"
	"github.com/joaquinrovira/notes/internal/store"
)

func TestHandler(log *slog.Logger, fs store.Filesystem) http.HandlerFunc {
	tmpl, err := template.New("countdown").Parse(CountdownPage)
	if err != nil {
		panic(err)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract Token
		raw := r.URL.Query().Get("s")

		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			v = int64((36 * time.Hour).Seconds())
		}

		token := token.NewTokenV1()
		token.NotBefore = new(time.Now().Add(time.Duration(v) * time.Second))

		h, err := customCountdown(token, fs)
		if err != nil {
			log.Error("custom countdown failure", slog.String("err", err.Error()))
			countdownToken(*token, tmpl, w)
			return
		}
		h.ServeHTTP(w, r)
	}
}
