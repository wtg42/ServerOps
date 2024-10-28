package main

import (
	"bufio"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/coder/websocket"
	"github.com/wtg42/ServerOps/backend/ssh"
)

// API Server
func main() {
	mux := http.NewServeMux()

	// TODO: 在這裡新增你要的路由跟功能
	// Logs Service router
	mux.HandleFunc("/logs", handleLogsService)

	// 設定靜態文件路由，提供 dist 目錄中的文件
	// 編譯 WebOps 專案後 把 dist 目錄放在根目錄
	fs := http.FileServer(http.Dir("./public")) // dist 是 Astro 編譯後的目錄
	mux.Handle("/", fs)                         // 根目錄將提供靜態文件

	// 啟動伺服器
	log.Println("Server started on :8080")
	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

// 開啟 Websocket 處理跟網頁端的溝通
func handleLogsService(w http.ResponseWriter, r *http.Request) {
	wscon, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // 視需要配置選項
	})
	if err != nil {
		log.Printf("failed to accept websocket connection: %v", err)
		return
	}
	defer wscon.Close(websocket.StatusInternalError, "unexpected close")

	ctx := context.Background()
	var con *ssh.Connection

	type Message struct {
		Target string `json:"target"`
		Data   string `json:"data"`
		Type   string `json:"type"`
	}
	// 開始接收 Websocket 請求
	for {
		typ, data, err := wscon.Read(ctx)
		if err != nil {
			log.Printf("error reading from websocket: %v", err)

			// If we receive a close message, close the connection.
			statusCode := websocket.CloseStatus(err)
			if statusCode != -1 {
				wscon.Close(statusCode, err.Error())
			}

			return
		}

		if typ == websocket.MessageText {
			// json structure
			var v Message
			err = json.Unmarshal(data, &v)
			if err != nil {
				log.Printf("error reading from json data: %v", err)
				return
			}
			log.Printf("Json contents is : %v", v)
			if con == nil {
				targetIP := strings.Builder{}
				targetIP.WriteString(v.Target)
				targetIP.WriteString(":2222")
				con = ssh.InitConnectionInstance(targetIP.String())
				defer con.Client.Close() // 先 Seesion 在 Client
				targetIP.Reset()
			}

			// 新的會話處理
			con.NewSession()
			defer con.Session.Close()

			stdoutPipe, err := con.Session.StdoutPipe()
			if err != nil {
				log.Printf("error getting stdout pipe: %v", err)
			}

			stderrPipe, err := con.Session.StderrPipe()
			if err != nil {
				log.Printf("error getting stderr pipe: %v", err)
			}

			err = con.Session.Start("logs")
			if err != nil {
				log.Printf("error running command: %v", err)
			}

			// Remember that Start() and Wait() are asynchronous,
			// and you can use a goroutine to handle the data in between.
			go func() {
				scanner := bufio.NewScanner(stdoutPipe)
				for scanner.Scan() {
					log.Printf("TEXT:: %v", scanner.Text())
					// 將標準輸出的每一行發送到 WebSocket 客戶端
					if err := wscon.Write(ctx, websocket.MessageText, scanner.Bytes()); err != nil {
						log.Printf("error writing to websocket: %v", err)
						continue
					}
				}

				if scanner.Err() != nil {
					log.Printf("error reading from stdout: %v", scanner.Err())
				}
			}()

			go func() {
				scanner := bufio.NewScanner(stderrPipe)
				for scanner.Scan() {
					// 將標準輸出的每一行發送到 WebSocket 客戶端
					if err := wscon.Write(ctx, websocket.MessageText, scanner.Bytes()); err != nil {
						log.Printf("error writing to websocket: %v", err)
						continue
					}
				}

				if scanner.Err() != nil {
					log.Printf("error reading from stdout: %v", scanner.Err())
				}
			}()

			err = con.Session.Wait()
			if err != nil {
				log.Printf("error running command: %v", err)
			}
		} else {
			log.Printf("received:%v %s", typ, data)
		}
	}
}
