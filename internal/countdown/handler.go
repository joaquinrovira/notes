package countdown

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/joaquinrovira/notes/internal/auth/token"
	"github.com/joaquinrovira/notes/internal/store"
)

func Handler(log *slog.Logger, t *token.Cypher, content store.Content) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		payload, ok := token.Extract(t, r)
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		switch token := payload.(type) {
		case *token.TokenV1:
			h, err := TokenCountdown(token, content.Data())
			if err != nil {
				log.Error("custom countdown failure", slog.String("err", err.Error()))
				RenderCountdown(*token, Default(content.Static()), w)
				return
			}
			h.ServeHTTP(w, r)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func TestHandler(log *slog.Logger, fs store.Filesystem) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw := r.URL.Query().Get("s")

		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			v = int64((36 * time.Hour).Seconds())
		}

		token := token.NewTokenV1()
		token.NotBefore = new(time.Now().Add(time.Duration(v) * time.Second))

		h, err := TokenCountdown(token, fs)
		if err != nil {
			log.Error("custom countdown failure", slog.String("err", err.Error()))
			RenderCountdown(*token, Default(fs), w)
			return
		}
		h.ServeHTTP(w, r)
	}
}
