package api

import "github.com/gin-gonic/gin"

type Response struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
	Msg  string      `json:"msg"`
}

func JSON(c *gin.Context, httpStatus int, code int, data interface{}, msg string) {
	c.JSON(httpStatus, Response{
		Code: code,
		Data: data,
		Msg:  msg,
	})
}

func Success(c *gin.Context, data interface{}) {
	JSON(c, 200, 200, data, "ok")
}

func Created(c *gin.Context, data interface{}) {
	JSON(c, 201, 201, data, "created")
}

func Error(c *gin.Context, httpStatus int, code int, msg string) {
	JSON(c, httpStatus, code, nil, msg)
}
