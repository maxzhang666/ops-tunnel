package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/coder/websocket"
)

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		slog.Error("websocket accept failed", "err", err)
		return
	}
	defer conn.CloseNow()

	ch, cancel := s.eng.Events().Subscribe(256)
	defer cancel()

	ctx := conn.CloseRead(r.Context())

	for {
		select {
		case <-ctx.Done():
			conn.Close(websocket.StatusNormalClosure, "")
			return
		case evt, ok := <-ch:
			if !ok {
				conn.Close(websocket.StatusNormalClosure, "")
				return
			}
			data, err := json.Marshal(evt)
			if err != nil {
				slog.Error("marshal event failed", "err", err)
				continue
			}
			if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
				slog.Debug("websocket write failed", "err", err)
				return
			}
		}
	}
}
