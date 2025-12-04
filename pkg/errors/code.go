package errors

// ---------------------------
// Code 类型，可根据业务定义
// ---------------------------

type Code int

const (
	CodeInternalCallFailed  Code = 50005
	CodeUserContextCanceled Code = 50006
	CodeUnknown             Code = 0
)
