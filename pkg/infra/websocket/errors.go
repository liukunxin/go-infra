package websocket

import cws "github.com/coder/websocket"

func normalizeCloseErr(err error) error {
	if err == nil {
		return nil
	}
	code := cws.CloseStatus(err)
	switch code {
	case cws.StatusNormalClosure, cws.StatusGoingAway:
		return nil
	default:
		return err
	}
}
