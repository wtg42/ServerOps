package ssh

import (
	"io"
	"log"
	"os"

	"golang.org/x/crypto/ssh"
)

// Client 端連線 Websocket 解析完命令後再連線到 SSH 伺服器(使用帳密認證)
func ConnectToSSHServer() {
	// SSH 連接配置
	config := &ssh.ClientConfig{
		User: "root", // 替換為伺服器的使用者名稱
		Auth: []ssh.AuthMethod{
			ssh.Password("your_password"), // 替換為對應的密碼
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 忽略主機密鑰驗證（建議開發階段使用）
	}

	// 連接到 SSH 伺服器
	client, err := ssh.Dial("tcp", "192.168.91.63:2222", config) // 替換成伺服器的地址和端口
	if err != nil {
		log.Fatalf("Failed to dial: %s", err)
	}
	defer client.Close()

	// 開啟一個新會話
	session, err := client.NewSession()
	if err != nil {
		log.Fatalf("Failed to create session: %s", err)
	}
	defer session.Close()

	// 將會話的標準輸出連接到本地的標準輸出
	session.Stdout = io.Discard
	session.Stderr = io.Discard

	// 執行命令
	err = session.Run("ls")
	if err != nil {
		log.Fatalf("Failed to run command: %s", err)
	}
}

// Client 端連線 Websocket 解析完命令後再連線到 SSH 伺服器(使用公私鑰認證)
func ConnectToSSHServerWithKey() *ssh.Client {
	// 讀取私鑰文件
	// TODO: 實作
	var err error
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("unable to get home directory: %v", err)
	}
	key, err := os.ReadFile(homeDir + "/.ssh/mse_id_rsa")
	if err != nil {
		log.Fatalf("unable to read private key: %v", err)
	}

	// 解析私鑰
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		log.Fatalf("unable to parse private key: %v", err)
	}

	// SSH 連接配置，使用公私鑰認證
	config := &ssh.ClientConfig{
		User: "root", // 使用伺服器上可接受的使用者名稱
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 忽略主機密鑰驗證（建議開發階段使用）
	}

	// 連接到 SSH 伺服器
	client, err := ssh.Dial("tcp", "192.168.91.63:2222", config) // 替換成伺服器地址和端口
	if err != nil {
		log.Fatalf("Failed to dial: %s", err)
	}
	defer client.Close()

	// 開啟一個新會話
	session, err := client.NewSession()
	if err != nil {
		log.Fatalf("Failed to create session: %s", err)
	}
	defer session.Close()

	// 將會話的標準輸出連接到本地的標準輸出
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	// 執行命令
	err = session.Run("date")
	if err != nil {
		log.Fatalf("Failed to run command: %s", err)
	}

	return client
}
