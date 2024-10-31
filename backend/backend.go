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
	cryptoSSH "golang.org/x/crypto/ssh" // 使用別名因為 ssh 名稱衝突
)

// color for log message.
type code = string
type ColorCode struct {
	Red   code
	Green code
	Reset code
}

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
	color := ColorCode{
		Red:   "\033[31m",
		Green: "\033[32m",
		Reset: "\033[0m",
	}
	wscon, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // 視需要配置選項
	})
	if err != nil {
		log.Printf(color.Red+"failed to accept websocket connection: %v"+color.Reset, err)
		return
	}
	defer wscon.Close(websocket.StatusInternalError, "unexpected close")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var con *ssh.Connection

	// Frontend Json Structure.
	type Message struct {
		Target string `json:"target"`
		Data   string `json:"data"`
		Type   string `json:"type"`
	}
	// 開始接收 Websocket 請求
	for {
		log.Println(color.Green + "Waiting for message..." + color.Reset)
		typ, data, err := wscon.Read(ctx)
		if err != nil {
			statusCode := websocket.CloseStatus(err)
			if statusCode != -1 {
				log.Printf(color.Red+"WebSocket closed with status: %d, reason: %v"+color.Reset, statusCode, err)
			} else {
				log.Printf(color.Red+"WebSocket closed with an unknown reason, error: %v"+color.Reset, err)
			}
			return
		}

		if typ == websocket.MessageText {
			// json structure
			var v Message
			err = json.Unmarshal(data, &v)
			if err != nil {
				log.Printf(color.Red+"error reading from json data: %v"+color.Reset, err)
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

			// TODO: Use PTY to handle the command.
			if err := con.Session.RequestPty("xterm-256color", 120, 80, cryptoSSH.TerminalModes{}); err != nil {
				log.Fatalf(color.Red+"Request for pseudo terminal failed: %s"+color.Reset, err)
			}

			stdoutPipe, err := con.Session.StdoutPipe()
			if err != nil {
				log.Printf(color.Red+"error getting stdout pipe: %v"+color.Reset, err)
			}

			stderrPipe, err := con.Session.StderrPipe()
			if err != nil {
				log.Printf(color.Red+"error getting stderr pipe: %v"+color.Reset, err)
			}

			err = con.Session.Start("logs")
			if err != nil {
				log.Printf(color.Red+"error running command `logs`: %v"+color.Reset, err)
			}

			// Remember that Start() and Wait() are asynchronous,
			// and you can use a goroutine to handle the data in between.
			go func() {
				scanner := bufio.NewScanner(stdoutPipe)
				for scanner.Scan() {
					log.Printf("TEXT:: %v", scanner.Text())
					// 將標準輸出的每一行發送到 WebSocket 客戶端
					if err := wscon.Write(ctx, websocket.MessageText, scanner.Bytes()); err != nil {
						log.Printf(color.Red+"error writing to websocket: %v"+color.Reset, err)
						continue
					}
				}

				if scanner.Err() != nil {
					log.Printf(color.Red+"error reading from stdout: %v"+color.Reset, scanner.Err())
				}
			}()

			go func() {
				scanner := bufio.NewScanner(stderrPipe)
				for scanner.Scan() {
					// 將標準輸出的每一行發送到 WebSocket 客戶端
					if err := wscon.Write(ctx, websocket.MessageText, scanner.Bytes()); err != nil {
						log.Printf(color.Red+"error writing to websocket: %v"+color.Reset, err)
						continue
					}
				}

				if scanner.Err() != nil {
					log.Printf(color.Red+"error reading from stdout: %v"+color.Reset, scanner.Err())
				}
			}()

			go func() {
				err = con.Session.Wait()
				if err != nil {
					log.Printf(color.Red+"error running command on Session.Wait(): %v"+color.Reset, err)
				}
			}()
		} else {
			log.Printf("received:%v %s", typ, data)
		}
	}
}
