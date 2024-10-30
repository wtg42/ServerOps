package ssh

import (
	"io"
	"log"
	"os"

	"golang.org/x/crypto/ssh"
)

type Connection struct {
	Ip      string
	Session *ssh.Session
	Client  *ssh.Client
}

// Client 端連線 Websocket 解析完命令後再連線到 SSH 伺服器(使用帳密認證)
func ConnectToSSHServer(ip string, pw string) {
	// SSH 連接配置
	config := &ssh.ClientConfig{
		User: "root", // 替換為伺服器的使用者名稱
		Auth: []ssh.AuthMethod{
			ssh.Password(pw), // 明碼密碼認證 (建議開發階段使用)
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 忽略主機密鑰驗證（建議開發階段使用）
	}

	// 連接到 SSH 伺服器
	client, err := ssh.Dial("tcp", ip, config) // 替換成伺服器的地址和端口
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
// TODO: Rewrite it as an init function
func connectToSSHServerWithKey(ip string) *ssh.Client {
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
	client, err := ssh.Dial("tcp", ip, config) // 替換成伺服器地址和端口
	if err != nil {
		log.Fatalf("Failed to dial: %s", err)
	}

	return client
}

func InitConnectionInstance(ipPort string) *Connection {
	client := connectToSSHServerWithKey(ipPort)
	session, err := client.NewSession()
	if err != nil {
		log.Fatalf("Failed to create session: %s", err)
	}
	return &Connection{
		Ip:      ipPort,
		Client:  client,
		Session: session,
	}
}

// Close 關閉 SSH 連接和 Session，以避免資源洩露
func (c *Connection) Close() {
	if c.Client != nil {
		c.Client.Close()
	}

	if c.Session != nil {
		c.Session.Close()
	}
}

// Session cannot be used multiple times for a single connection.
func (c *Connection) NewSession() {
	if c.Session != nil {
		c.Session.Close()
	}
	var err error
	c.Session, err = c.Client.NewSession()
	if err != nil {
		log.Fatalf("Failed to create session: %s", err)
	}
}
