package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/coder/websocket"
)

func main() {
	mux := http.NewServeMux()

	// 使用新的路由匹配方式來處理 WebSocket 請求
	mux.HandleFunc("/ws", handleWebSocket)

	// 設定靜態文件路由，提供 dist 目錄中的文件
	// 編譯 WebOps 專案後 把 dist 目錄放在根目錄
	fs := http.FileServer(http.Dir("./dist")) // dist 是 Astro 編譯後的目錄
	mux.Handle("/", fs)                       // 根目錄將提供靜態文件

	// 啟動伺服器
	log.Println("Server started on :8080")
	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

// 開啟 Websocket 處理跟網頁端的溝通
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // 視需要配置選項
	})
	if err != nil {
		log.Printf("failed to accept websocket connection: %v", err)
		return
	}
	defer c.Close(websocket.StatusInternalError, "unexpected close")

	ctx := context.Background()

	for {
		typ, data, err := c.Read(ctx)
		if err != nil {
			log.Printf("error reading from websocket: %v", err)
			log.Println("=>", websocket.CloseStatus(err))
			return
		}

		if typ == websocket.MessageText {
			// json structure
			var v interface{}
			err = json.Unmarshal(data, &v)
			if err != nil {
				log.Printf("error reading from websocket: %v", err)
				return
			}
			log.Printf("Json contents is : %v", v)
		} else {
			log.Printf("received:%v %s", typ, data)
		}
	}
}
