package node

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"

	"github.com/imroc/req"
	"github.com/tidwall/gjson"
)

type ClientInterface interface {
	Call(path string, request []interface{}) (*gjson.Result, error)
}

// Broadcast error classification.
//
// RPC transport failures fall into two categories that callers (handlers,
// task workers) can inspect via errors.Is/errors.As to return structured
// error codes (e.g. upstream_node_unreachable / mvc_broadcast_timeout)
// instead of a generic 500:
//
//   - ErrUpstreamNodeUnreachable: the node could not be reached at all
//     (connection refused, no route, DNS failure, dial timeout, EOF). This
//     is the failure mode seen when a stale/decommissioned node address is
//     configured (e.g. the 172.31.168.215 i/o timeout outage).
//   - ErrBroadcastTimeout: the request was sent but the node did not
//     respond within the overall HTTP timeout.
//
// Errors returned by the RPC server itself (e.g. "txn-already-known",
// validation errors) are NOT classified here and remain raw so existing
// duplicate-tx handling still works.
var (
	ErrUpstreamNodeUnreachable = errors.New("upstream node unreachable")
	ErrBroadcastTimeout        = errors.New("broadcast timed out")
)

// RPC HTTP timeouts. Kept tight so a dead node fails in seconds instead of
// hanging on the default 30s dial / 2m overall inherited from imroc/req.
const (
	rpcDialTimeout    = 5 * time.Second
	rpcRequestTimeout = 15 * time.Second
)

// A Client is a Bitcoin RPC client. It performs RPCs over HTTP using json
// request and responses. A Client must be configured with a secret token
// to authenticate with other Cores on the network.
type Client struct {
	URL         string
	AccessToken string
	Debug       bool
	client      *req.Req
}

type Response struct {
	Code    int         `json:"code, omitempty"`
	Error   interface{} `json:"error, omitempty"`
	Result  interface{} `json:"result, omitempty"`
	Message string      `json:"message, omitempty"`
	Id      string      `json:"id, omitempty"`
}

func NewClientNode(url string, accessToken string, debug bool) *Client {
	cli := &Client{
		URL:         url,
		AccessToken: accessToken,
		Debug:       debug,
	}

	api := req.New()
	// Wire a custom http.Client with explicit dial + overall timeouts so a
	// dead/unreachable node fails in seconds instead of hanging on the
	// library defaults (30s dial / 2m overall).
	api.SetClient(&http.Client{
		Timeout: rpcRequestTimeout,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   rpcDialTimeout,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	})
	cli.client = api
	return cli
}

func (cl *Client) Call(path string, request []interface{}) (*gjson.Result, error) {

	var body = make(map[string]interface{}, 0)

	if cl.client == nil {
		return nil, errors.New("Api url is not setup. ")
	}

	authHeader := req.Header{
		"Accept":        "Application/json",
		"Authorization": "Basic " + cl.AccessToken,
	}

	//json-rpc
	body["jsonrpc"] = "1.0"
	body["id"] = "1"
	body["method"] = path
	body["params"] = request

	if cl.Debug { //debug
		//log.Std.Info("Start Request API...")
		fmt.Println("Start Request API...")
	}

	r, err := cl.client.Post(cl.URL, req.BodyJSON(body), authHeader)

	if cl.Debug { //debug
		//log.Std.Info("Request API Completed")
		fmt.Println("Request API Completed")
	}

	if cl.Debug { //debug
		//log.Std.Info("%+v", r)
		fmt.Printf("%+v \n", r)
	}

	if err != nil {
		return nil, classifyBroadcastError(err)
	}

	resp := gjson.ParseBytes(r.Bytes())
	err = IsError(&resp)
	if err != nil {
		return nil, err
	}

	result := resp.Get("result")
	return &result, nil
}

// See 2 (end of page 4) http://www.ietf.org/rfc/rfc2617.txt
// "To receive authorization, the client sends the userid and password,
// separated by a single colon (":") character, within a base64
// encoded string in the credentials."
// It is not meant to be urlencoded.
func BasicAuth(userName, password string) string {
	auth := userName + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

// isError
func IsError(result *gjson.Result) error {
	/*
		{
			"result": null,
			"error": {
				"code": -8,
				"message": "Block height out of range"
			},
			"id": "foo"
		}
	*/
	var err error
	if !result.Get("error").IsObject() {
		if !result.Get("result").Exists() {
			return errors.New("Response is empty. ")
		}
		return nil
	}

	errInfo := fmt.Sprintf("[%d]%s",
		result.Get("error.code").Int(),
		result.Get("error.message").String())
	err = errors.New(errInfo)
	return err
}

// classifyBroadcastError maps a transport-level error from the HTTP client
// onto the typed broadcast errors so callers can return structured codes.
// RPC server errors (e.g. "txn-already-known") are returned unchanged.
//
// Order matters: a dial-phase failure (refused / DNS / dial-timeout / EOF)
// means the node could not be reached at all → ErrUpstreamNodeUnreachable.
// Only an overall request deadline (node accepted the connection but never
// replied) → ErrBroadcastTimeout. This distinction matches the production
// outage, where a stale address produced a dial i/o timeout that must read
// as "unreachable", not "slow".
func classifyBroadcastError(err error) error {
	if err == nil {
		return nil
	}

	// Dial-phase / connection-level failures (the stale-node signature).
	if isUnreachableOpError(err) {
		return fmt.Errorf("%w: %s: %v", ErrUpstreamNodeUnreachable, "node unreachable", err)
	}

	// Overall request timeout (http.Client.Timeout exceeded) surfaces as a
	// net/http deadline via the url.Error wrapper.
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if urlErr.Timeout() || errors.Is(urlErr.Err, context.DeadlineExceeded) {
			return fmt.Errorf("%w: %s: %v", ErrBroadcastTimeout, "rpc request deadline", err)
		}
		return err
	}

	return err
}

// isUnreachableOpError reports whether err represents a failure to even
// contact the node (refused, reset, host unknown, network unreachable,
// dial timeout, EOF). These are the signatures of a stale/dead address.
func isUnreachableOpError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.EHOSTUNREACH) ||
		errors.Is(err, syscall.ENETUNREACH) {
		return true
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		// Dial-phase i/o timeout ("dial tcp ...: i/o timeout") means the
		// host never answered — treat as unreachable.
		return true
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	return false
}

// IsDuplicateBroadcastError reports whether err is a benign "transaction
// already known/exists" RPC response, in which case a re-broadcast after a
// partial success should be treated as success. Shared here so both the
// sync and async upload paths use one definition.
func IsDuplicateBroadcastError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already") || strings.Contains(msg, "known") ||
		strings.Contains(msg, "exists") || strings.Contains(msg, "spent")
}
