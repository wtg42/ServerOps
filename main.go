package main

import (
	"context"
	"log"
	"net/http"

	"github.com/coder/websocket"
)

func main() {
	mux := http.NewServeMux()

	// 使用新的路由匹配方式來處理 WebSocket 請求
	mux.HandleFunc("/ws", handleWebSocket)

	// 啟動伺服器
	log.Println("Server started on :8080")
	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // 視需要配置選項
	})
	if err != nil {
		log.Printf("failed to accept websocket connection: %v", err)
		return
	}
	defer c.Close(websocket.StatusInternalError, "unexpected close")

	// 建立一個 10 秒超時的 context
	ctx := context.Background()

	// var v interface{}
	// err = wsjson.Read(ctx, c, &v)
	count := 0
	for {
		count = count + 1
		typ, data, err := c.Read(ctx)
		if err != nil {
			log.Printf("error reading from websocket: %v", err)
			return
		}
		log.Printf("received:%v %s", typ, data)

		if count == 10 {
			// 正常關閉 WebSocket 連接
			c.Close(websocket.StatusNormalClosure, "normal closure")
		}
	}

}
