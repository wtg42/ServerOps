package main

import (
	"fmt"
	"io"
	"log"
	"os/exec"

	"github.com/gliderlabs/ssh"
)

// SSH Server
func main() {
	// 啟動 SSH 伺服器，指定連接的處理程序
	ssh.Handle(func(s ssh.Session) {
		// 取得使用者名稱
		user := s.User()

		// 打印訊息，表示成功連接
		fmt.Fprintf(s, "Hello %s\n", user)
		// 回傳訊息給連接的使用者
		io.WriteString(s, "Welcome to the Go SSH server!\n")

		// 解析用戶傳來的字串
		cmd := s.Command()
		if len(cmd) > 0 {
			switch cmd[0] {
			case "logs":
				// TODO: 這段改成使用 goroutine + StdoutPipe 持續讀取 tail -f
				out, err := exec.Command("date").Output()
				if err != nil {
					io.WriteString(s, "Failed to run date command.\n")
				}
				s.Write(out)
			default:
				io.WriteString(s, cmd[0]+" <- Unknown command.\n")
			}
		}
	})

	// 啟動伺服器，監聽 2222 埠口
	log.Println("Starting SSH server on port 2222...")
	err := ssh.ListenAndServe(":2222", nil)
	if err != nil {
		log.Fatal("Failed to start SSH server: ", err)
	}
}
