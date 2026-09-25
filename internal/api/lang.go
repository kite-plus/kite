package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/kite-plus/kite/internal/render/theme"
)

// language is the language a request asks to be answered in, from its
// Accept-Language header: the tag weighted highest, the first of equals. It
// is empty when the request names none.
//
// Only what a theme says about itself is translated this way. Everything
// else the API returns is data, which is the same in any language, or a
// message meant for whoever reads a log.
func language(r *http.Request) string {
	best, weight := "", 0.0
	for part := range strings.SplitSeq(r.Header.Get("Accept-Language"), ",") {
		tag, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		tag = strings.TrimSpace(tag)
		if tag == "" || tag == "*" {
			continue
		}
		q := 1.0
		if value, ok := strings.CutPrefix(strings.TrimSpace(params), "q="); ok {
			if parsed, err := strconv.ParseFloat(value, 64); err == nil {
				q = parsed
			}
		}
		if q > weight {
			best, weight = tag, q
		}
	}
	return best
}

// described is a theme's manifest in the language a request asks for.
func described(th *theme.Theme, r *http.Request) theme.Manifest {
	return th.Localized(language(r))
}
