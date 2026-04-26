package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kite-plus/kite/internal/errcodes"
)

// Response is the unified JSON envelope returned by every API endpoint.
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// PagedData wraps a page of items together with pagination metadata.
type PagedData struct {
	Items interface{} `json:"items"`
	Total int64       `json:"total"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
}

// Success writes a 200 OK response with code 0.
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    int(errcodes.Success),
		Message: "success",
		Data:    data,
	})
}

// Created writes a 201 Created response with code 0.
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Code:    int(errcodes.Success),
		Message: "success",
		Data:    data,
	})
}

// Fail writes a non-successful response. errCode is the business code from
// the [errcodes] package (httpCode is taken as-is — it must agree with the
// canonical mapping in [errcodes.Catalog]).
func Fail(c *gin.Context, httpCode int, errCode int, message string) {
	c.JSON(httpCode, Response{
		Code:    errCode,
		Message: message,
		Data:    nil,
	})
}

// BadRequest writes a 400 response with the generic bad-request code.
func BadRequest(c *gin.Context, message string) {
	Fail(c, http.StatusBadRequest, int(errcodes.BadRequest), message)
}

// Unauthorized writes a 401 response with the generic unauthorized code.
func Unauthorized(c *gin.Context, message string) {
	Fail(c, http.StatusUnauthorized, int(errcodes.Unauthorized), message)
}

// Forbidden writes a 403 response with the generic forbidden code.
func Forbidden(c *gin.Context, message string) {
	Fail(c, http.StatusForbidden, int(errcodes.Forbidden), message)
}

// NotFound writes a 404 response with the generic not-found code.
func NotFound(c *gin.Context, message string) {
	Fail(c, http.StatusNotFound, int(errcodes.NotFound), message)
}

// ServerError writes a 500 response with the generic internal-error code.
func ServerError(c *gin.Context, message string) {
	Fail(c, http.StatusInternalServerError, int(errcodes.InternalError), message)
}

// Paged writes a 200 response wrapping a page of items and pagination metadata.
func Paged(c *gin.Context, items interface{}, total int64, page, size int) {
	Success(c, PagedData{
		Items: items,
		Total: total,
		Page:  page,
		Size:  size,
	})
}
