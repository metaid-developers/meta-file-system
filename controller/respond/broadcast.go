package respond

import (
	"errors"

	"github.com/gin-gonic/gin"

	"meta-file-system/node"
)

// BroadcastError maps a classified node broadcast error onto a structured
// response and writes it. It is the single place handlers call when a
// broadcast step fails, so the code/slug mapping stays consistent:
//
//   - node.ErrUpstreamNodeUnreachable -> 50301 / upstream_node_unreachable
//   - node.ErrBroadcastTimeout        -> 50401 / mvc_broadcast_timeout
//   - anything else                   -> generic 50000 (ServerError)
//
// HTTP stays 200 (existing convention; the real outcome is in `code`), and
// the response carries requestId via Message so callers can correlate. The
// underlying error string is included in `message` for diagnostics.
func BroadcastError(c *gin.Context, err error) {
	if err == nil {
		ServerError(c, "broadcast failed")
		return
	}
	switch {
	case errors.Is(err, node.ErrUpstreamNodeUnreachable):
		Error(c, CodeUpstreamNodeUnreachable, err.Error())
	case errors.Is(err, node.ErrBroadcastTimeout):
		Error(c, CodeBroadcastTimeout, err.Error())
	default:
		ServerError(c, err.Error())
	}
}
