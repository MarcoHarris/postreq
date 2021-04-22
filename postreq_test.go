package postreq

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-test/deep"
	"github.com/sethgrid/pester"
	"github.com/stretchr/testify/assert"
)

var inputJSON = `
{
    "request": {
        "name": "test",
        "request": {
            "auth": {
                "type": "basic",
                "basic": [
                    {
                        "key": "username",
                        "value": "{{username}}",
                        "type": "string"
                    }
                ]
            },
            "method": "GET",
            "header": [
				{
					"key": "myHeader",
					"value": "{{clientId}}",
					"type": "text"
				}
            ],
            "url": {
                "raw": "{{host}}/status/200?api_key={{apiKey}}",
                "host": [
                    "{{host}}"
                ],
                "path": [
                    "status",
                    "200"
                ],
                "query": [
                    {
                        "key": "api_key",
                        "value": "{{apiKey}}"
                    },
                    {
                        "key": "page",
                        "value": "{{page}}"
                    }
                ],
                "variable": []
            }
        },
        "response": []
    },
    "params": {
        "host": "https://httpbin.org",
        "username": "user234",
        "password": "password123",
        "accessToken": "1234passwordtoken",
        "clientId": "asdf2323fff",
        "apiKey": "l337",
        "id": "1234",
        "page": "1"
    }
}
`

type rawInput struct {
	Request json.RawMessage `json:"request"`
	Params  json.RawMessage `json:"params"`
}

var input rawInput
var inputErr = json.Unmarshal([]byte(inputJSON), &input)

var inputParams map[string]string
var inputParamsErr = json.Unmarshal(input.Params, &inputParams)

var item Item
var itemErr = json.Unmarshal([]byte(input.Request), &item)

var server *httptest.Server

func Test_replacePlaceholder(t *testing.T) {
	type args struct {
		placeholder string
		params      map[string]string
	}
	tests := []struct {
		name  string
		args  args
		want  string
		want1 bool
	}{
		{"replace env variable", args{"{{host}}", inputParams}, "https://httpbin.org", true},
		{"replace path variable", args{":id", inputParams}, "1234", true},
		{"replace non variable", args{"api", inputParams}, "api", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := replacePlaceholder(tt.args.placeholder, tt.args.params)
			assert.Equal(t, got, tt.want)
			assert.Equal(t, got1, tt.want1)
		})
	}
}

func Test_isPlaceholder(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"test env variable", args{"{{host}}"}, true},
		{"test path variable", args{":id"}, true},
		{"test non variable", args{"api"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPlaceholder(tt.args.value)
			assert.Equal(t, got, tt.want)

		})
	}
}

func Test_getValue(t *testing.T) {
	type args struct {
		value  string
		params map[string]string
	}
	tests := []struct {
		name  string
		args  args
		want  string
		want1 bool
	}{
		{"replace env variable", args{"{{host}}", inputParams}, "https://httpbin.org", true},
		{"replace path variable", args{":id", inputParams}, "1234", true},
		{"non variable", args{"api", inputParams}, "api", true},
		{"replacement not available", args{"{{date}}", inputParams}, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := getValue(tt.args.value, tt.args.params)
			assert.Equal(t, got, tt.want)
			assert.Equal(t, got1, tt.want1)
		})
	}
}

func Test_generateEndpoint(t *testing.T) {
	type args struct {
		hosts  []string
		paths  []string
		params map[string]string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"generate endpoint from input", args{item.Request.URL.Host, item.Request.URL.Path, inputParams}, "https://httpbin.org/status/200"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateEndpoint(tt.args.hosts, tt.args.paths, tt.args.params)
			assert.Equal(t, got, tt.want)
		})
	}
}

func Test_generateAuth(t *testing.T) {
	type args struct {
		inputAuthType string
		params        map[string]string
	}
	tests := []struct {
		name  string
		args  args
		want  string
		want1 bool
	}{
		{"generate auth from input", args{item.Request.Auth.Type, inputParams}, "Basic dXNlcjIzNDpwYXNzd29yZDEyMw==", true},
		{"generate bearer auth", args{"bearer", inputParams}, "Bearer 1234passwordtoken", true},
		{"generate unsupported auth", args{"digest", inputParams}, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := generateAuth(tt.args.inputAuthType, tt.args.params)
			assert.Equal(t, got, tt.want)
			assert.Equal(t, got1, tt.want1)
		})
	}
}

func Test_generateRequest(t *testing.T) {
	type args struct {
		item   Item
		params map[string]string
	}

	expectedOutput, _ := http.NewRequest("GET", "https://httpbin.org/status/200?api_key=l337&page=1", bytes.NewBuffer(nil))
	expectedOutput.Header.Set("Authorization", "Basic dXNlcjIzNDpwYXNzd29yZDEyMw==")
	expectedOutput.Header.Set("Myheader", "asdf2323fff")
	tests := []struct {
		name string
		args args
		want *http.Request
	}{
		{"test", args{item, inputParams}, expectedOutput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateRequest(tt.args.item, tt.args.params)

			if diff := deep.Equal(got, tt.want); diff != nil {
				t.Error(diff)
			}
		})
	}
}

func TestService_Do(t *testing.T) {
	type fields struct {
		Client      *pester.Client
		Concurrency int64
		MaxRetries  int64
	}
	type args struct {
		input  string
		params map[string]string
	}
	client := pester.New()
	invalidParams := map[string]string{
		"host":     "magnet://asd.com",
		"username": "user",
		"apiKey":   "l337",
		"id":       "1234",
		"page":     "1",
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []byte
		want1   int
		wantErr bool
	}{
		{"test invalid json", fields{client, 3, 5}, args{"{", inputParams}, nil, 0, true},
		{"test payload", fields{client, 3, 5}, args{string(input.Request), inputParams}, []byte{}, 200, false},
		{"test invalid URL", fields{client, 3, 5}, args{string(input.Request), invalidParams}, nil, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Service{
				Client:      tt.fields.Client,
				Concurrency: tt.fields.Concurrency,
				MaxRetries:  tt.fields.MaxRetries,
			}
			_, got, got1, err := c.Do(tt.args.input, tt.args.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.Do() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, got, tt.want)
			assert.Equal(t, got1, tt.want1)
		})
	}
}

func TestNewService(t *testing.T) {
	type args struct {
		concurrency int
		maxRetries  int
	}
	client := pester.New()
	client.Concurrency = 3
	client.MaxRetries = 5
	client.Backoff = pester.ExponentialBackoff
	client.KeepLog = true
	service := &Service{
		Client: client,
	}
	tests := []struct {
		name string
		args args
		want *Service
	}{
		{"get a new service", args{3, 5}, service},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got := NewService(tt.args.concurrency, tt.args.maxRetries)
			if diff := deep.Equal(got, tt.want); diff != nil {
				t.Error(diff)
			}
		})
	}
}
