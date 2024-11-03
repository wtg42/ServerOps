package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gliderlabs/ssh"
)

var wg sync.WaitGroup // 用於等待所有 goroutine 完成

// SSH Server
func main() {
	const sshport string = ":2222"
	sshServ := &ssh.Server{
		Addr: sshport,
	}

	// 啟動 SSH 伺服器，指定連接的處理程序
	sshServ.Handle(func(s ssh.Session) {
		// Track session start and end.
		wg.Add(1)
		defer wg.Done()
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
			case "log":
				// TODO: 這段改成使用 goroutine + StdoutPipe 持續讀取 tail -f
				fetchLogs(s)
			case "process":
				topCmd(s)
			default:
				io.WriteString(s, cmd[0]+" <- Unknown command.\n")
			}
		}
	})

	// Start up the SSH server, then listen for a signal.
	go func() {
		log.Printf("Starting SSH server on port %s...", sshport)
		err := sshServ.ListenAndServe()
		if err != nil {
			log.Fatal("Failed to start SSH server: ", err)
		}
	}()

	// Create a signal channel and listen for SIGINT and SIGTERM. e.g. Ctrl+C
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigc
	log.Printf("Received signal %s: shutting down ssh server...\n", sig)

	// 這段不確定是否會使用到，可能一次性的指令才會有作用。
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sshServ.Shutdown(ctx); err != nil {
		log.Fatalf("Failed to gracefully shutdown SSH server: %v", err)
	}

	wg.Wait()
	log.Printf("SSH server gracefully stoped.")
}

// fetchLogs streams logs from
// "/var/log/php.log" and "/var/log/apache/error_log".
// Use io.WriteString() to return data.
func fetchLogs(s ssh.Session) {
	// 預防指令還沒有跑完 主線程關閉
	wg.Add(1)
	defer wg.Done()

	args := []string{
		"tail",
		"-f",
		"/var/log/php.log",
		"/var/log/apache/error_log",
		// "/var/log/system.log", // for test
	}
	cmd := exec.Command(args[0], args[1:]...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Error getting stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatalf("Error starting command: %v", err)
	}

	// The tail -f cmd will block the current goroutine,
	// so we need to create a goroutine to read stdout.
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			// 輸出當前 buffer 中的資料
			io.WriteString(s, scanner.Text()+"\n")
		}

		if err := scanner.Err(); err != nil {
			log.Fatalf("Error reading stdout: %s", err)
		}
	}()

	// If session is closed, kill the process.
	<-s.Context().Done()
	if cmd.Process != nil {
		cmd.Process.Kill()
	}
}

// topCmd streams output from "top".
func topCmd(s ssh.Session) {
	// 預防指令還沒有跑完 主線程關閉
	wg.Add(1)
	defer wg.Done()

	args := []string{"top"}
	cmd := exec.Command(args[0])
	// setting stdout.
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Error getting 'top' stdout pipe: %v", err)
	}

	// Start 'top' and wait for it to finish.
	if err := cmd.Start(); err != nil {
		log.Fatalf("Error starting 'top' command: %v", err)
	}

	// 定義清屏控制碼
	clearScreenCode := []byte("\033[H\033[2J")

	// Create a scanner for 'top' stdout.
	go func() {
		reader := bufio.NewReader(stdout)
		for {
			chunk, err := reader.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				fmt.Println("Error reading:", err)
				return
			}
			// 檢查 chunk 中是否包含清屏控制碼
			if bytes.Contains(chunk, clearScreenCode) {
				fmt.Println("Clear screen detected!")
				// 輸出當前 buffer 中的資料
				io.WriteString(s, "\033[2J\033[1;1H")
			}

			io.WriteString(s, string(chunk))
		}
	}()

	// Block until 'top' finishes or session is closed.
	<-s.Context().Done()

	if cmd.Process != nil {
		cmd.Process.Kill()
	}
}
