package ws

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// testWsClient 创建一个真实 WebSocket 连接并注册到 Hub
func testWsClient(t *testing.T, hub *Hub, username, role, ip string) (*websocket.Conn, *ClientConn, func()) {
	t.Helper()

	registered := make(chan *ClientConn)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
		wsConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		cc := &ClientConn{
			Conn:     wsConn,
			Username: username,
			Role:     role,
			IP:       ip,
			LoginAt:  time.Now(),
		}
		hub.Register(cc)
		registered <- cc
		select {}
	}))

	u, _ := url.Parse(srv.URL)
	u.Scheme = "ws"
	clientConn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		srv.Close()
		t.Fatalf("WebSocket 连接失败: %v", err)
	}

	cc := <-registered
	return clientConn, cc, func() { clientConn.Close(); srv.Close() }
}

// readWSMessage 读取下一条消息（超时 3s）
func readWSMessage(t *testing.T, conn *websocket.Conn) []byte {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("读取消息失败: %v", err)
	}
	return msg
}

// readWSMessageTimeout 尝试读取消息，超时返回 nil（不失败）
func readWSMessageTimeout(t *testing.T, conn *websocket.Conn, timeout time.Duration) []byte {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		return nil
	}
	return msg
}

func TestNewHub(t *testing.T) {
	h := NewHub(100)
	if h == nil {
		t.Fatal("NewHub returned nil")
	}
	if cap(h.chatHistory) != 100 {
		t.Errorf("chatHistory capacity = %d, want 100", cap(h.chatHistory))
	}
	if h.maxHistory != 100 {
		t.Errorf("maxHistory = %d, want 100", h.maxHistory)
	}
}

func TestHub_Register_GetOnlineUsers(t *testing.T) {
	hub := NewHub(100)
	go hub.Run()

	conn1, _, close1 := testWsClient(t, hub, "user1", "admin", "192.168.1.1")
	defer close1()
	conn2, _, close2 := testWsClient(t, hub, "user2", "user", "192.168.1.2")
	defer close2()

	// 跳过两个连接的 online 广播（数量不定，不用来验证用户数）
	readWSMessageTimeout(t, conn1, 500*time.Millisecond)
	readWSMessageTimeout(t, conn2, 500*time.Millisecond)

	// 通过 GetOnlineUsers 验证用户列表（同步 API，无时序问题）
	users := hub.GetOnlineUsers()
	if len(users) != 2 {
		t.Errorf("GetOnlineUsers 返回 %d 个用户, want 2", len(users))
	}

	// 验证用户列表（无聚合，直接数目）
	_, _, close3 := testWsClient(t, hub, "user1", "admin", "192.168.1.3")
	defer close3()
	time.Sleep(100 * time.Millisecond)

	users = hub.GetOnlineUsers()
	if len(users) != 3 {
		t.Errorf("连接数应为 3, 得到 %d", len(users))
	}
}

func TestHub_Unregister(t *testing.T) {
	hub := NewHub(100)
	go hub.Run()

	conn, cc, close := testWsClient(t, hub, "u1", "user", "10.0.0.1")
	defer close()

	// 跳过 online 广播
	readWSMessageTimeout(t, conn, 200*time.Millisecond)

	users := hub.GetOnlineUsers()
	if len(users) != 1 {
		t.Fatalf("注册后应有 1 用户, 得到 %d", len(users))
	}

	hub.Unregister(cc)
	time.Sleep(100 * time.Millisecond)

	users = hub.GetOnlineUsers()
	if len(users) != 0 {
		t.Errorf("注销后应有 0 用户, 得到 %d: %+v", len(users), users)
	}
}

func TestHub_Broadcast_Basic(t *testing.T) {
	hub := NewHub(100)
	go hub.Run()

	conn, _, close := testWsClient(t, hub, "test", "user", "")
	defer close()

	time.Sleep(200 * time.Millisecond)

	// 发送简单消息
	hub.Broadcast([]byte("ping"))
	time.Sleep(200 * time.Millisecond)

	// 持续读取直到遇到 "ping"（跳过 online 广播）
	for i := 0; i < 10; i++ {
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, msg, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("读取消息失败: %v", err)
		}
		if string(msg) == "ping" {
			return // 成功
		}
	}
	t.Error("未收到 'ping' 消息")
}

func TestHub_Broadcast_SendsToAllClients(t *testing.T) {
	hub := NewHub(100)
	go hub.Run()

	conn1, _, close1 := testWsClient(t, hub, "alice", "user", "")
	defer close1()
	conn2, _, close2 := testWsClient(t, hub, "bob", "user", "")
	defer close2()

	// 发送聊天消息
	chatMsg := ChatMessage{Type: "chat", Username: "alice", Content: "hello", Time: time.Now()}
	data, _ := json.Marshal(chatMsg)
	hub.Broadcast(data)

	// 读取两个连接的消息，跳过 online 广播，找到聊天消息
	findChatMsg := func(conn *websocket.Conn) ChatMessage {
		for i := 0; i < 10; i++ {
			msg := readWSMessage(t, conn)
			var cm ChatMessage
			if json.Unmarshal(msg, &cm) == nil && cm.Type == "chat" {
				return cm
			}
		}
		t.Fatal("未找到聊天消息")
		return ChatMessage{}
	}

	cm1 := findChatMsg(conn1)
	cm2 := findChatMsg(conn2)

	if cm1.Content != "hello" {
		t.Errorf("conn1 收到内容 = %q, want 'hello'", cm1.Content)
	}
	if cm2.Content != "hello" {
		t.Errorf("conn2 收到内容 = %q, want 'hello'", cm2.Content)
	}
}

func TestHub_Broadcast_SavesChatHistory(t *testing.T) {
	hub := NewHub(10)
	go hub.Run()

	chatMsg := ChatMessage{Type: "chat", Username: "testuser", Content: "history test", Time: time.Now()}
	data, _ := json.Marshal(chatMsg)
	hub.Broadcast(data)
	time.Sleep(30 * time.Millisecond)

	history := hub.GetChatHistory()
	if len(history) != 1 {
		t.Fatalf("历史记录数 = %d, want 1", len(history))
	}
	if history[0].Content != "history test" {
		t.Errorf("历史内容 = %q, want 'history test'", history[0].Content)
	}

	// 非 chat 类型不应写入历史
	onlineMsg := ChatMessage{Type: "online"}
	data, _ = json.Marshal(onlineMsg)
	hub.Broadcast(data)
	time.Sleep(20 * time.Millisecond)

	history = hub.GetChatHistory()
	if len(history) != 1 {
		t.Errorf("非 chat 消息不应写入历史, 历史记录数 = %d, want 1", len(history))
	}
}

func TestHub_History_MaxLimit(t *testing.T) {
	hub := NewHub(3)
	go hub.Run()

	for i := 0; i < 5; i++ {
		msg := ChatMessage{Type: "chat", Content: "msg", Time: time.Now()}
		data, _ := json.Marshal(msg)
		hub.Broadcast(data)
	}
	time.Sleep(50 * time.Millisecond)

	history := hub.GetChatHistory()
	if len(history) > 3 {
		t.Errorf("历史超过 maxHistory, 得到 %d 条, want <=3", len(history))
	}
}

func TestHub_SendToUser(t *testing.T) {
	hub := NewHub(100)
	go hub.Run()

	conn1, _, close1 := testWsClient(t, hub, "target", "user", "")
	defer close1()
	conn2, _, close2 := testWsClient(t, hub, "other", "user", "")
	defer close2()

	msg, _ := json.Marshal(ChatMessage{Type: "chat", Content: "private message", Time: time.Now()})
	count := hub.SendToUser("target", msg)
	if count != 1 {
		t.Errorf("SendToUser 返回 %d, want 1", count)
	}

	// target 应收到消息（跳过 online 广播）
	findChatMsg := func(conn *websocket.Conn) ChatMessage {
		for i := 0; i < 10; i++ {
			msg := readWSMessage(t, conn)
			var cm ChatMessage
			if json.Unmarshal(msg, &cm) == nil && cm.Type == "chat" {
				return cm
			}
		}
		t.Fatal("未找到聊天消息")
		return ChatMessage{}
	}

	cm := findChatMsg(conn1)
	if cm.Content != "private message" {
		t.Errorf("目标用户收到: %q, want 'private message'", cm.Content)
	}

	// other 不应收到任何聊天消息
	for i := 0; i < 5; i++ {
		msgOther := readWSMessageTimeout(t, conn2, 100*time.Millisecond)
		if msgOther == nil {
			return // 超时 = 没有消息，正确
		}
		var cmOther ChatMessage
		if json.Unmarshal(msgOther, &cmOther) == nil && cmOther.Type == "chat" {
			t.Error("other 用户不应收到定向消息")
			return
		}
	}
}

func TestHub_SendToUser_Nonexistent(t *testing.T) {
	hub := NewHub(100)
	go hub.Run()

	count := hub.SendToUser("nobody", []byte("test"))
	if count != 0 {
		t.Errorf("不存在的用户应返回 0, 得到 %d", count)
	}
}

func TestHub_GetChatHistory_Empty(t *testing.T) {
	hub := NewHub(100)
	history := hub.GetChatHistory()
	if len(history) != 0 {
		t.Errorf("新 hub 应有 0 条历史, 得到 %d", len(history))
	}
}

func TestHub_ChatHistory_CopyIsolation(t *testing.T) {
	hub := NewHub(100)
	go hub.Run()

	msg := ChatMessage{Type: "chat", Content: "test", Time: time.Now()}
	data, _ := json.Marshal(msg)
	hub.Broadcast(data)
	time.Sleep(30 * time.Millisecond)

	history1 := hub.GetChatHistory()
	history2 := hub.GetChatHistory()

	if len(history1) != 1 || len(history2) != 1 {
		t.Fatalf("应有 1 条历史, 得到 %d / %d", len(history1), len(history2))
	}

	history1[0].Content = "modified"
	if hub.GetChatHistory()[0].Content != "test" {
		t.Error("GetChatHistory 应返回副本，修改不应影响原始数据")
	}
}

// findChatMsg 从 WebSocket 连接中读取消息直到找到聊天消息
func findChatMsg(t *testing.T, conn *websocket.Conn) ChatMessage {
	t.Helper()
	for i := 0; i < 10; i++ {
		msg := readWSMessage(t, conn)
		var cm ChatMessage
		if json.Unmarshal(msg, &cm) == nil && cm.Type == "chat" {
			return cm
		}
	}
	t.Fatal("未找到聊天消息")
	return ChatMessage{}
}

func TestClientConn_SendJSON(t *testing.T) {
	hub := NewHub(100)
	go hub.Run()

	conn, cc, close := testWsClient(t, hub, "testuser", "admin", "10.0.0.1")
	defer close()

	// SendJSON 发送自定义消息
	type customMsg struct {
		Type    string `json:"type"`
		Payload string `json:"payload"`
	}
	err := cc.SendJSON(customMsg{Type: "custom", Payload: "hello from SendJSON"})
	if err != nil {
		t.Fatalf("SendJSON 失败: %v", err)
	}

	// 读取消息（跳过 online 广播）
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	for i := 0; i < 10; i++ {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("读取消息失败: %v", err)
		}
		var decoded customMsg
		if json.Unmarshal(msg, &decoded) == nil && decoded.Type == "custom" {
			if decoded.Payload != "hello from SendJSON" {
				t.Errorf("payload = %q, want 'hello from SendJSON'", decoded.Payload)
			}
			return
		}
	}
	t.Error("未找到 SendJSON 发送的消息")
}

func TestClientConn_SendBatchHistory(t *testing.T) {
	hub := NewHub(100)
	go hub.Run()

	conn, cc, close := testWsClient(t, hub, "testuser", "admin", "10.0.0.1")
	defer close()

	// SendBatchHistory 批量发送多条历史
	history := []ChatMessage{
		{Type: "chat", Username: "alice", Content: "msg1", Time: time.Now()},
		{Type: "chat", Username: "bob", Content: "msg2", Time: time.Now()},
		{Type: "chat", Username: "alice", Content: "msg3", Time: time.Now()},
	}
	cc.SendBatchHistory(history)

	// 读取所有消息（跳过 online 广播）
	var received []ChatMessage
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	for i := 0; i < 10; i++ {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		var cm ChatMessage
		if json.Unmarshal(msg, &cm) == nil && cm.Type == "chat" {
			received = append(received, cm)
			if len(received) == 3 {
				break
			}
		}
	}

	if len(received) != 3 {
		t.Fatalf("应收到 3 条历史消息, 得到 %d", len(received))
	}
	if received[0].Content != "msg1" {
		t.Errorf("第 1 条内容 = %q, want 'msg1'", received[0].Content)
	}
	if received[1].Content != "msg2" {
		t.Errorf("第 2 条内容 = %q, want 'msg2'", received[1].Content)
	}
	if received[2].Content != "msg3" {
		t.Errorf("第 3 条内容 = %q, want 'msg3'", received[2].Content)
	}
}
