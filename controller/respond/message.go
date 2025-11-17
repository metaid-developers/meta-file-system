package respond

import (
	"time"

	"github.com/gin-gonic/gin"
)

// Message unified response structure
type Message struct {
	Code           int         `json:"code"`
	Message        string      `json:"message"`
	ProcessingTime int64       `json:"processingTime"`
	Data           interface{} `json:"data"`
}

// Response response structure (for Swagger)
// @Description Unified API response structure
type Response struct {
	Code           int         `json:"code" example:"0" description:"Response code: 0=success, 40000=param error, 40400=not found, 50000=server error"`
	Message        string      `json:"message" example:"success" description:"Response message"`
	ProcessingTime int64       `json:"processingTime" example:"123" description:"Request processing time (milliseconds)"`
	Data           interface{} `json:"data" description:"Response data"`
}

// HTTP status code constants
const (
	CodeSuccess      = 0     // Success
	CodeInvalidParam = 40000 // Parameter error
	CodeNotFound     = 40400 // Resource not found
	CodeServerError  = 50000 // Server error
)

// Success message constants
const (
	MsgSuccess = "success"
	MsgFailed  = "failed"
)

// Success return success response
func Success(c *gin.Context, data interface{}) {
	SuccessWithMsg(c, MsgSuccess, data)
}

// SuccessWithMsg return success response (custom message)
func SuccessWithMsg(c *gin.Context, message string, data interface{}) {
	processingTime := getProcessingTime(c)
	c.JSON(200, Message{
		Code:           CodeSuccess,
		Message:        message,
		ProcessingTime: processingTime,
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

// getProcessingTime calculate request processing time (milliseconds)
func getProcessingTime(c *gin.Context) int64 {
	if startTime, exists := c.Get("start_time"); exists {
		if t, ok := startTime.(time.Time); ok {
			return time.Since(t).Milliseconds()
		}
	}
	return 0
}

// TimingMiddleware timing middleware
func TimingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("start_time", time.Now())
		c.Next()
	}
}
