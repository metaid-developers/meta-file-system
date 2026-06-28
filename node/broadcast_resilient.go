package node

import (
	"errors"
	"fmt"
	"time"

	"meta-file-system/conf"
)

// Resilient broadcast knobs.
//
// These are package-level vars (not config) so the behavior is predictable
// and tests can shrink the retry sleep. The retry only applies to the
// uploader's BroadcastTxResilient path; the indexer keeps using the plain
// single-shot BroadcastTx.
var (
	broadcastRetryAttempts = 2                      // total tries on the primary node (1 = no retry)
	broadcastRetryBackoff  = 500 * time.Millisecond // base backoff between attempts
	broadcastRetryJitter   = 250 * time.Millisecond // added to base backoff
)

// BroadcastTxResilient broadcasts a raw transaction with retry and an
// optional fallback node. It is the uploader-facing entry point; the
// indexer continues to use the single-shot node.BroadcastTx unchanged.
//
// Behavior:
//  1. Try the primary node up to broadcastRetryAttempts times, with backoff,
//     but ONLY on transient/network failures. A duplicate-tx ("already
//     known") response is treated as success. RPC validation errors (e.g.
//     bad-txns, txn-mempool-conflict) are returned immediately without
//     retry, since retrying cannot help.
//  2. If the primary is exhausted and a fallback URL is configured for the
//     chain, build a one-off client on the fallback URL and try once.
//  3. The returned error is classified (ErrUpstreamNodeUnreachable /
//     ErrBroadcastTimeout / raw RPC error) so handlers can map it to a
//     structured code.
func BroadcastTxResilient(chain, txHex string) (string, error) {
	// Primary: retry transient failures only.
	var lastErr error
	for attempt := 1; attempt <= broadcastRetryAttempts; attempt++ {
		txID, err := broadcastOnce(chain, txHex)
		if err == nil {
			return txID, nil
		}
		if IsDuplicateBroadcastError(err) {
			// Duplicate broadcast is effectively success.
			return "", nil
		}
		lastErr = err
		// Don't retry on RPC server errors (validation rejects). Only retry
		// when the node itself was unreachable or timed out.
		if !isTransientBroadcastError(err) {
			break
		}
		if attempt < broadcastRetryAttempts {
			time.Sleep(broadcastRetryBackoff + broadcastRetryJitter)
		}
	}

	// Fallback node, if configured.
	if cfg, ok := conf.RpcConfigMap[chain]; ok && cfg.FallbackUrl != "" {
		txID, err := broadcastFallback(cfg.FallbackUrl, cfg.Username, cfg.Password, txHex)
		if err == nil {
			return txID, nil
		}
		if IsDuplicateBroadcastError(err) {
			return "", nil
		}
		lastErr = err
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("broadcast failed with no error")
	}
	return "", lastErr
}

// broadcastOnce performs a single primary-node broadcast. It is a package
// var so tests can stub it without standing up a real RPC node; production
// code delegates to the cached client controller.
var broadcastOnce = func(chain, txHex string) (string, error) {
	return BroadcastTx(chain, txHex)
}

// broadcastFallback is the fallback-node broadcast. Package var so tests
// can stub it; production delegates to broadcastOnURL.
var broadcastFallback = broadcastOnURL

// isTransientBroadcastError reports whether err is worth retrying: a network
// reachability failure or a request timeout. RPC server errors (duplicate
// tx, validation rejects) are NOT transient.
func isTransientBroadcastError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrUpstreamNodeUnreachable) ||
		errors.Is(err, ErrBroadcastTimeout) ||
		isUnreachableOpError(err)
}

// broadcastOnURL sends sendrawtransaction to an arbitrary URL using a
// freshly built (uncached) client. Used for the fallback node so it does
// not pollute the per-chain ClientMap.
func broadcastOnURL(url, user, pass, txHex string) (string, error) {
	token := BasicAuth(user, pass)
	cli := NewClientNode(url, token, false)
	request := []interface{}{txHex, false}
	result, err := cli.Call("sendrawtransaction", request)
	if err != nil {
		return "", err
	}
	return result.String(), nil
}
