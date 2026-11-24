package countdown

import (
	"net/http"

	"github.com/joaquinrovira/notes/internal/auth/token"
	"github.com/joaquinrovira/notes/internal/store"
)

func TokenCountdown(token *token.TokenV1, fs store.Filesystem) (h http.Handler, err error) {
	t, err := CountdownInDir(fs, token.Home)
	if err != nil {
		return nil, err
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var countdown int64
		if token.NotBefore != nil {
			countdown = token.NotBefore.Unix()
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		type CountdownData struct {
			Location    string
			UnixSeconds int64
		}
		err := t.Execute(w, CountdownData{Location: token.Home, UnixSeconds: countdown})
		if err != nil {
			panic(err)
		}
	}), nil
}
