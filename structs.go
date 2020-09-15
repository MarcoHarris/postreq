package postreq

import (
	"github.com/sethgrid/pester"
)

type Service struct {
	Client      *pester.Client
	Concurrency int64
	MaxRetries  int64
}

type Item struct {
	Name    string   `json:"name"`
	Request *Request `json:"request,omitempty"`
}

type Request struct {
	URL    *URL        `json:"url"`
	Auth   *Auth       `json:"auth,omitempty"`
	Method string      `json:"method"`
	Header []*KeyValue `json:"header,omitempty"`
}

type URL struct {
	Raw       string      `json:"raw"`
	Host      []string    `json:"host,omitempty"`
	Path      []string    `json:"path,omitempty"`
	Query     []*KeyValue `json:"query,omitempty"`
	Variables []*KeyValue `json:"variable,omitempty" mapstructure:"variable"`
}

type KeyValue struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type Auth struct {
	Type   string       `json:"type,omitempty"`
	Basic  []*AuthParam `json:"basic,omitempty"`
	Bearer []*AuthParam `json:"bearer,omitempty"`
	Digest []*AuthParam `json:"digest,omitempty"`
	OAuth1 []*AuthParam `json:"oauth1,omitempty"`
	OAuth2 []*AuthParam `json:"oauth2,omitempty"`
}

type AuthParam struct {
	Key   string      `json:"key,omitempty"`
	Value interface{} `json:"value,omitempty"`
	Type  string      `json:"type,omitempty"`
}
