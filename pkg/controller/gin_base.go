package controller

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/liukunxin/go-infra/internal/consts"
	kerr "github.com/liukunxin/go-infra/pkg/errors"
	"github.com/liukunxin/go-infra/pkg/trace"
	"net/http"
)

type GinBase struct {
}

type CommonResponse struct {
	Code    int32       `json:"code"`
	Msg     string      `json:"msg"`
	Data    interface{} `json:"data,omitempty"`
	TraceID string      `json:"trace_id"`
}

func (*GinBase) SuccessResponse(ctx *gin.Context, data interface{}) {
	resp := &CommonResponse{
		Code:    0,
		Msg:     "success",
		Data:    data,
		TraceID: trace.GetTraceID(ctx.Request.Context()),
	}
	ctx.Set(consts.ResponseCode, resp.Code)
	ctx.Set(consts.ResponseMsg, resp.Msg)
	ctx.JSON(http.StatusOK, resp)
}

func (*GinBase) ErrorResponse(ctx *gin.Context, err error) {
	errorResponseWithData(ctx, err, nil)
}

func (*GinBase) ErrorResponseWithData(ctx *gin.Context, err error, data interface{}) {
	errorResponseWithData(ctx, err, data)
}

func errorResponseWithData(ctx *gin.Context, err error, data interface{}) {
	var er *kerr.Error
	errMsg := err.Error()
	var code int32 = -1
	httpStatus := http.StatusInternalServerError
	if ok := errors.As(err, &er); ok {
		errMsg = er.Error()
		code = int32(er.Code)
		httpStatus = er.Status.HTTPCode()
	}
	resp := CommonResponse{
		Code:    code,
		Msg:     errMsg,
		Data:    data,
		TraceID: trace.GetTraceID(ctx.Request.Context()),
	}
	ctx.Set(consts.ResponseCode, resp.Code)
	ctx.Set(consts.ResponseMsg, resp.Msg)
	ctx.JSON(httpStatus, resp)
}
