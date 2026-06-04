package collab

import "errors"

var (
	// ErrDuplicate 事件已存在（幂等去重拦截）。
	ErrDuplicate = errors.New("collab: duplicate event")
	// ErrSessionClosed 会话已关闭，不再接受写入。
	ErrSessionClosed = errors.New("collab: session is closed")
	// ErrSessionNotFound 会话不存在。
	ErrSessionNotFound = errors.New("collab: session not found")
	// ErrSessionExists 会话已存在，不能重复创建。
	ErrSessionExists = errors.New("collab: session already exists")
	// ErrEmptySessionID session_id 不能为空。
	ErrEmptySessionID = errors.New("collab: session_id is required")
)
