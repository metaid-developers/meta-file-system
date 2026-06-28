package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"meta-file-system/controller/respond"
	"meta-file-system/model"
)

// HEAD counterparts for the file-content routes.
//
// Per RFC 7231 §4.3.2, a HEAD request must return the same headers as GET
// (including Content-Type, Content-Disposition, Content-Length) but no body.
// Gin does NOT auto-route HEAD to GET handlers, so without these the routes
// return Gin's native "404 page not found" for HEAD. That broke clients that
// probe availability with HEAD (e.g. the OAC --verify step), which read the
// 404 as "not indexed yet" even though the pin was servable.
//
// These handlers run the SAME metadata lookup the GET handlers use (so a
// not-indexed pin still 404s the same way), then set the headers WITHOUT
// writing the body and WITHOUT downloading the file bytes from storage. For
// the "accelerate" (OSS redirect) routes they return 200 with headers instead
// of the 307 redirect a GET issues, so a HEAD probe answers "is it available?"
// in a single round trip.

// HeadFileContent is the HEAD counterpart of GetFileContent.
func (h *IndexerQueryHandler) HeadFileContent(c *gin.Context) {
	headFileContentByPin(c, h.indexerFileService.GetFileByPinID)
}

// HeadLatestFileContentByFirstPinID is the HEAD counterpart of GetLatestFileContentByFirstPinID.
func (h *IndexerQueryHandler) HeadLatestFileContentByFirstPinID(c *gin.Context) {
	headFileContentByFirstPin(c, "firstPinId", h.indexerFileService.GetLatestFileByFirstPinID)
}

// HeadFastFileContent is the HEAD counterpart of GetFastFileContent.
func (h *IndexerQueryHandler) HeadFastFileContent(c *gin.Context) {
	headFileContentByPin(c, h.indexerFileService.GetFileByPinID)
}

// HeadLatestFastFileContentByFirstPinID is the HEAD counterpart of GetLatestFastFileContentByFirstPinID.
func (h *IndexerQueryHandler) HeadLatestFastFileContentByFirstPinID(c *gin.Context) {
	headFileContentByFirstPin(c, "firstPinId", h.indexerFileService.GetLatestFileByFirstPinID)
}

// headFileContentByPin runs the metadata lookup by pinId and responds with
// headers only (no body) on success, or the same 404 the GET handler returns
// when the pin is not indexed.
func headFileContentByPin(c *gin.Context, lookup func(string) (*model.IndexerFile, error)) {
	pinID := c.Param("pinId")
	if pinID == "" {
		respond.InvalidParam(c, "pinId is required")
		return
	}
	file, err := lookup(pinID)
	if err != nil || file == nil {
		respond.NotFound(c, "file not found")
		return
	}
	writeHeadHeaders(c, file)
	c.Status(200)
}

// headFileContentByFirstPin runs the metadata lookup by firstPinId and responds
// with headers only (no body) on success.
func headFileContentByFirstPin(c *gin.Context, param string, lookup func(string) (*model.IndexerFile, error)) {
	id := c.Param(param)
	if id == "" {
		respond.InvalidParam(c, param+" is required")
		return
	}
	file, err := lookup(id)
	if err != nil || file == nil {
		respond.NotFound(c, "file not found")
		return
	}
	writeHeadHeaders(c, file)
	c.Status(200)
}

// writeHeadHeaders sets the same Content-* headers the GET handlers set, plus
// an accurate Content-Length from the indexed metadata (no body is written).
func writeHeadHeaders(c *gin.Context, file *model.IndexerFile) {
	if file.ContentType != "" {
		c.Header("Content-Type", file.ContentType)
	}
	if file.FileName != "" {
		c.Header("Content-Disposition", "inline; filename=\""+file.FileName+"\"")
	}
	if file.FileSize > 0 {
		c.Header("Content-Length", strconv.FormatInt(file.FileSize, 10))
	}
}
