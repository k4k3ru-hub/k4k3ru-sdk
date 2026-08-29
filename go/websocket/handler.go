package websocket

import k4k3ruWebSocket "github.com/k4k3ru-hub/websocket/go"

type messageReceiver interface {
	HandleMessage([]byte)
	HandleClose()
}

type sessionHandler struct {
	receiver messageReceiver
}

func (h *sessionHandler) HandleMessage(_ k4k3ruWebSocket.SessionContext, message []byte) {
	if h == nil || h.receiver == nil {
		return
	}
	h.receiver.HandleMessage(append([]byte(nil), message...))
}

func (h *sessionHandler) HandleClose(k4k3ruWebSocket.SessionContext) {
	if h == nil || h.receiver == nil {
		return
	}
	h.receiver.HandleClose()
}
