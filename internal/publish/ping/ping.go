// Package ping tells update services that a site has changed, with the
// XML-RPC weblogUpdates calls blog engines have long sent once something is
// published. A service such as Explore reads them to fetch a site sooner
// than it otherwise would.
package ping

import (
	"bytes"
	"cmp"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kite-plus/kite/internal/buildinfo"
)

// Timeout is how long a service has to answer a ping.
const Timeout = 10 * time.Second

// maxAnswer bounds what is read of an answer, which is a few lines.
const maxAnswer = 64 << 10

// Site is what a ping says about a site.
type Site struct {
	Title string
	URL   string

	// Feed is the address of the site's feed, or "" for a site with none.
	Feed string
}

// Method is the call a ping makes: extendedPing, which names the feed, or
// the plain ping for a site with no feed to name.
func (s Site) Method() string {
	if s.Feed != "" {
		return "weblogUpdates.extendedPing"
	}
	return "weblogUpdates.ping"
}

// params are the call's: the site's name and address, and for extendedPing
// the page that changed, which for a whole site is its address, and the feed.
func (s Site) params() []string {
	if s.Feed != "" {
		return []string{s.Title, s.URL, s.URL, s.Feed}
	}
	return []string{s.Title, s.URL}
}

type methodCall struct {
	XMLName xml.Name `xml:"methodCall"`
	Method  string   `xml:"methodName"`
	Params  []param  `xml:"params>param"`
}

type param struct {
	Value string `xml:"value>string"`
}

// body is the XML-RPC call a ping sends for a site.
func body(site Site) ([]byte, error) {
	call := methodCall{Method: site.Method()}
	for _, p := range site.params() {
		call.Params = append(call.Params, param{Value: p})
	}
	// On one line: some XML-RPC readers take the text around a value's
	// <string> for the value.
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	if err := xml.NewEncoder(&buf).Encode(call); err != nil {
		return nil, err
	}
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

// Send pings one service and returns what it said back. A ping fails when
// the service cannot be reached within Timeout, answers with an HTTP error or
// an XML-RPC fault, or reports an error of its own with flerror.
func Send(ctx context.Context, client *http.Client, endpoint string, site Site) (string, error) {
	call, err := body(site)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(call))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "text/xml")
	req.Header.Set("User-Agent", "kite/"+buildinfo.Version)

	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return "", fmt.Errorf("no answer within %s", Timeout)
		}
		// The address is the caller's to name; the error says it again.
		if u, ok := errors.AsType[*url.Error](err); ok {
			err = u.Err
		}
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("answered %s", resp.Status)
	}
	return read(io.LimitReader(resp.Body, maxAnswer))
}

// value is an XML-RPC value: a struct, a typed scalar such as <boolean>, or
// text, which is a string.
type value struct {
	Members []member `xml:"struct>member"`
	Typed   *scalar  `xml:",any"`
	Text    string   `xml:",chardata"`
}

type scalar struct {
	XMLName xml.Name
	Text    string `xml:",chardata"`
}

type member struct {
	Name  string `xml:"name"`
	Value value  `xml:"value"`
}

func (v value) String() string {
	if v.Typed != nil {
		return strings.TrimSpace(v.Typed.Text)
	}
	return strings.TrimSpace(v.Text)
}

// member is the value of a struct's member by name, "" when there is none.
func (v value) member(name string) string {
	for _, m := range v.Members {
		if m.Name == name {
			return m.Value.String()
		}
	}
	return ""
}

type methodResponse struct {
	XMLName xml.Name `xml:"methodResponse"`
	Params  []value  `xml:"params>param>value"`
	Fault   *struct {
		Value value `xml:"value"`
	} `xml:"fault"`
}

// read makes sense of a service's answer: a struct whose flerror says whether
// the ping was taken and whose message says why, or a fault.
func read(answer io.Reader) (string, error) {
	dec := xml.NewDecoder(answer)
	// What matters in an answer is ASCII, so one that names another charset
	// is read as it is rather than refused.
	dec.CharsetReader = func(_ string, input io.Reader) (io.Reader, error) { return input, nil }
	var r methodResponse
	if err := dec.Decode(&r); err != nil {
		return "", errors.New("the answer is not an XML-RPC response")
	}
	if r.Fault != nil {
		return "", fmt.Errorf("fault %s: %s", cmp.Or(r.Fault.Value.member("faultCode"), "?"),
			cmp.Or(r.Fault.Value.member("faultString"), "no reason given"))
	}
	if len(r.Params) == 0 {
		return "", errors.New("the answer holds no result")
	}
	result := r.Params[0]
	if len(result.Members) == 0 {
		return result.String(), nil
	}
	message := result.member("message")
	switch strings.ToLower(result.member("flerror")) {
	case "1", "true":
		return "", errors.New(cmp.Or(message, "the service reports an error"))
	}
	return message, nil
}
