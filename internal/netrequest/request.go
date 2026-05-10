package netrequest

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

var (
	ErrInvalidRequest = errors.New("invalid client request")
	ErrInvalidHeader  = errors.New("invalid request header")
	ErrNetLogActive   = errors.New("netLog is already capturing")
	ErrNetLogInactive = errors.New("netLog is not capturing")
)

type RedirectPolicy string

const (
	RedirectFollow RedirectPolicy = "follow"
	RedirectError  RedirectPolicy = "error"
	RedirectManual RedirectPolicy = "manual"
)

type UploadBody struct {
	ContentType string
	Bytes       []byte
}

type RequestOptions struct {
	Method           string
	URL              string
	Headers          http.Header
	SessionPartition string
	RedirectPolicy   RedirectPolicy
	UploadBody       *UploadBody
}

type ClientRequest struct {
	Method           string
	URL              string
	Headers          http.Header
	SessionPartition string
	RedirectPolicy   RedirectPolicy
	UploadBody       *UploadBody
	ended            bool
}

func NewClientRequest(options RequestOptions) (*ClientRequest, error) {
	normalized, err := normalizeRequest(options)
	if err != nil {
		return nil, err
	}
	return &normalized, nil
}

func (r *ClientRequest) End() {
	r.ended = true
}

func (r *ClientRequest) Ended() bool {
	return r.ended
}

func (r *ClientRequest) ParsedURL() (*url.URL, error) {
	parsed, err := url.Parse(r.URL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("%w: URL", ErrInvalidRequest)
	}
	return parsed, nil
}

func normalizeRequest(options RequestOptions) (ClientRequest, error) {
	method := strings.ToUpper(strings.TrimSpace(options.Method))
	if method == "" {
		method = http.MethodGet
	}
	if strings.ContainsAny(method, " \t\r\n") {
		return ClientRequest{}, fmt.Errorf("%w: method", ErrInvalidRequest)
	}
	parsed, err := url.Parse(options.URL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ClientRequest{}, fmt.Errorf("%w: URL", ErrInvalidRequest)
	}
	headers, err := cloneHeaders(options.Headers)
	if err != nil {
		return ClientRequest{}, err
	}
	policy := options.RedirectPolicy
	if policy == "" {
		policy = RedirectFollow
	}
	if policy != RedirectFollow && policy != RedirectError && policy != RedirectManual {
		return ClientRequest{}, fmt.Errorf("%w: redirect policy", ErrInvalidRequest)
	}
	var upload *UploadBody
	if options.UploadBody != nil {
		body := *options.UploadBody
		body.ContentType = strings.TrimSpace(body.ContentType)
		body.Bytes = append([]byte(nil), options.UploadBody.Bytes...)
		upload = &body
	}
	return ClientRequest{
		Method:           method,
		URL:              parsed.String(),
		Headers:          headers,
		SessionPartition: strings.TrimSpace(options.SessionPartition),
		RedirectPolicy:   policy,
		UploadBody:       upload,
	}, nil
}

func cloneHeaders(headers http.Header) (http.Header, error) {
	cloned := make(http.Header, len(headers))
	for name, values := range headers {
		if strings.TrimSpace(name) == "" || strings.ContainsAny(name, ":\r\n") {
			return nil, fmt.Errorf("%w: name", ErrInvalidHeader)
		}
		for _, value := range values {
			if strings.ContainsAny(value, "\r\n") {
				return nil, fmt.Errorf("%w: value", ErrInvalidHeader)
			}
			cloned.Add(name, value)
		}
	}
	return cloned, nil
}

type NetLog struct {
	active bool
	path   string
	events []string
}

func (n *NetLog) StartLogging(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("%w: log path", ErrInvalidRequest)
	}
	if n.active {
		return ErrNetLogActive
	}
	n.active = true
	n.path = path
	n.events = append(n.events, "start:"+path)
	return nil
}

func (n *NetLog) StopLogging() (string, error) {
	if !n.active {
		return "", ErrNetLogInactive
	}
	n.active = false
	n.events = append(n.events, "stop:"+n.path)
	return n.path, nil
}

func (n *NetLog) IsCurrentlyLogging() bool {
	return n.active
}

func (n *NetLog) Events() []string {
	return append([]string(nil), n.events...)
}
