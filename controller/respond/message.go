package respond

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Message unified response structure
type Message struct {
	Code           int         `json:"code"`
	Message        string      `json:"message"`
	ProcessingTime int64       `json:"processingTime"`
	RequestId      string      `json:"requestId,omitempty"`
	ErrorCode      string      `json:"errorCode,omitempty"`
	Data           interface{} `json:"data"`
}

// Response response structure (for Swagger)
// @Description Unified API response structure
type Response struct {
	Code           int         `json:"code" example:"0" description:"Response code: 0=success, 40000=param error, 40400=not found, 50000=server error, 50301=upstream node unreachable, 50401=broadcast timeout"`
	Message        string      `json:"message" example:"success" description:"Response message"`
	ProcessingTime int64       `json:"processingTime" example:"123" description:"Request processing time (milliseconds)"`
	RequestId      string      `json:"requestId,omitempty" example:"9b1c..." description:"Per-request id echoed for tracing"`
	ErrorCode      string      `json:"errorCode,omitempty" example:"upstream_node_unreachable" description:"Machine-readable error slug for classified failures (e.g. upstream_node_unreachable, mvc_broadcast_timeout)"`
	Data           interface{} `json:"data" description:"Response data"`
}

// HTTP status code constants
const (
	CodeSuccess      = 0     // Success
	CodeInvalidParam = 40000 // Parameter error
	CodeNotFound     = 40400 // Resource not found
	CodeServerError  = 50000 // Server error

	// Classified broadcast failure codes. Carried in the `code` field with a
	// matching machine-readable slug in `errorCode`, so callers (e.g. OAC)
	// can distinguish a dead node from a generic server error without parsing
	// free-text messages.
	CodeUpstreamNodeUnreachable = 50301 // errorCode: upstream_node_unreachable
	CodeBroadcastTimeout        = 50401 // errorCode: mvc_broadcast_timeout
)

// Machine-readable error slugs, paired with the codes above.
const (
	ErrorCodeUpstreamNodeUnreachable = "upstream_node_unreachable"
	ErrorCodeBroadcastTimeout        = "mvc_broadcast_timeout"
)

// Success message constants
const (
	MsgSuccess = "success"
	MsgFailed  = "failed"
)

// Context key holding the per-request id.
const requestIDKey = "request_id"

// HeaderNameRequestID is the request/response header carrying the request id.
const HeaderNameRequestID = "X-Request-Id"

// Success return success response
func Success(c *gin.Context, data interface{}) {
	SuccessWithMsg(c, MsgSuccess, data)
}

// Success return success response
func SuccessWithCode(c *gin.Context, code int, data interface{}) {
	processingTime := getProcessingTime(c)
	c.JSON(200, Message{
		Code:           code,
		Message:        MsgSuccess,
		ProcessingTime: processingTime,
		RequestId:      getRequestID(c),
		Data:           data,
	})
}

// SuccessWithMsg return success response (custom message)
func SuccessWithMsg(c *gin.Context, message string, data interface{}) {
	processingTime := getProcessingTime(c)
	c.JSON(200, Message{
		Code:           CodeSuccess,
		Message:        message,
		ProcessingTime: processingTime,
		RequestId:      getRequestID(c),
		Data:           data,
	})
}

// Error return error response
func Error(c *gin.Context, code int, message string) {
	ErrorWithData(c, code, message, nil)
}

// ErrorWithData return error response (with data)
func ErrorWithData(c *gin.Context, code int, message string, data interface{}) {
	processingTime := getProcessingTime(c)
	c.JSON(200, Message{
		Code:           code,
		Message:        message,
		ProcessingTime: processingTime,
		RequestId:      getRequestID(c),
		ErrorCode:      errorCodeForCode(code),
		Data:           data,
	})
}

// InvalidParam return parameter error response
func InvalidParam(c *gin.Context, message string) {
	Error(c, CodeInvalidParam, message)
}

// NotFound return resource not found response
func NotFound(c *gin.Context, message string) {
	Error(c, CodeNotFound, message)
}

// ServerError return server error response
func ServerError(c *gin.Context, message string) {
	Error(c, CodeServerError, message)
}

// errorCodeForCode maps a numeric code to its machine-readable slug. Empty
// for codes with no slug (generic 50000, param, not-found).
func errorCodeForCode(code int) string {
	switch code {
	case CodeUpstreamNodeUnreachable:
		return ErrorCodeUpstreamNodeUnreachable
	case CodeBroadcastTimeout:
		return ErrorCodeBroadcastTimeout
	}
	return ""
}

// getProcessingTime calculate request processing time (milliseconds)
func getProcessingTime(c *gin.Context) int64 {
	if startTime, exists := c.Get("start_time"); exists {
		if t, ok := startTime.(time.Time); ok {
			return time.Since(t).Milliseconds()
		}
	}
	return 0
}

// getRequestID returns the per-request id stored by RequestIDMiddleware.
func getRequestID(c *gin.Context) string {
	if v, exists := c.Get(requestIDKey); exists {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return ""
}

// TimingMiddleware timing middleware
func TimingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("start_time", time.Now())
		c.Next()
	}
}

// RequestIDMiddleware ensures every request has a stable id. It honors an
// incoming X-Request-Id header when present, otherwise generates a UUID,
// stores it in the gin context, and echoes it back in the response header.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader(HeaderNameRequestID)
		if rid == "" {
			rid = uuid.NewString()
		}
		c.Set(requestIDKey, rid)
		c.Header(HeaderNameRequestID, rid)
		c.Next()
	}
}
