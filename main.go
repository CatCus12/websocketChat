package main

import (
	"context"
	"fmt"

	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var jwtSecretKey = []byte("your-secret-key")
var connectedUsers = make(map[string]*Client)

var pool *pgxpool.Pool

func initDB() {
	var err error
	pool, err = pgxpool.New(context.Background(), "postgres://postgres:1234@[::1]:5432/USERS?sslmode=disable")
	if err != nil {
		log.Fatalf("Не удалось подключиться к базе данных: %v", err)
	}
}

type Subject interface {
	RegisterClient(o Observer)
	RemoveClient(o Observer)
	NotifyObservers()
}
type Observer interface {
	Update(newMessage interface{})
}

type chat struct {
	mu         sync.Mutex
	clients    []Observer
	newMessage clientMessage
}

func (ch *chat) RegisterObserver(o Observer) {
	fmt.Println("new client!")

	ch.mu.Lock()
	defer ch.mu.Unlock()
	ch.clients = append(ch.clients, o)
}

func (ch *chat) RemoveObserver(o Observer) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	for i, client := range ch.clients {
		if client == o {
			ch.clients = append(ch.clients[:i], ch.clients[i+1:]...)
			break
		}
	}
}

func (ch *chat) NotifyObservers() {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	for _, client := range ch.clients {
		client.Update(ch.newMessage)
	}
}

type Client struct {
	conn *websocket.Conn
}

type clientMessage struct {
	Name        string `json:"name"`
	Password    string `json:"password"`
	Is_register bool   `json:"is_register"`
	Message     string `json:"message"`
	Token       string `json:"token"`
	Type        string `json:"type"`
}

type serverMessage struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

func (ch *chat) NotifyServerMessage(msg serverMessage) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	for _, client := range ch.clients {
		client.Update(msg)
	}
}

func (c *chat) RemoveClientFromMap(clientName string) {
	delete(connectedUsers, clientName)
}
func (c *chat) AutorizationCheck(newMessage clientMessage, newClient *Client) bool {

	if _, exists := connectedUsers[newMessage.Name]; exists {
		log.Println("Пользователь уже подключен:", newMessage.Name)
		return false
	}

	var name string
	var passwordHash string
	conn, err := pool.Acquire(context.Background())
	if err != nil {
		log.Println("Ошибка получения соединения из пула:", err)
		return false
	}
	defer conn.Release()

	row := conn.QueryRow(context.Background(), "SELECT name, password FROM users WHERE name=$1", newMessage.Name)

	err = row.Scan(&name, &passwordHash)

	fmt.Println("Имя и пароль", name, passwordHash)
	if err != nil {

		if err == pgx.ErrNoRows {
			log.Println("Пользователь не найден")
		} else {
			log.Println("Ошибка при запросе данных: ", err)
		}
		return false
	}
	passwordHash = strings.TrimSpace(passwordHash)
	newMessage.Password = strings.TrimSpace(newMessage.Password)

	if passwordHash == newMessage.Password {
		log.Println("Авторизация успешна")

		token, err := GenerateJWT(newMessage.Name, newMessage.Password, []byte("your-secret-key"))
		if err != nil {
			log.Printf("Ошибка при генерации токена: %v", err)
			return false
		}

		newMessage.Token = token
		log.Printf("Пользователь %s успешно авторизован! Токен: %s", newMessage.Name, token)

		chatRoom.NotifyServerMessage(serverMessage{Message: "Авторизация успешна", Type: "server"})

		connectedUsers[newMessage.Name] = newClient
		return true
	} else {
		log.Println("Неверный пароль")
		return false
	}
}
func getChatHistory(limit int) ([]clientMessage, error) {
	conn, err := pool.Acquire(context.Background())
	if err != nil {
		log.Println("Ошибка получения соединения из пула:", err)
		return nil, err
	}
	defer conn.Release()

	rows, err := conn.Query(context.Background(),
		"SELECT sender_name, content,type FROM messages ORDER BY created_at ASC LIMIT $1", limit)
	if err != nil {
		log.Printf("Ошибка получения истории сообщений: %v", err)
		return nil, err
	}
	defer rows.Close()

	var messages []clientMessage
	for rows.Next() {
		var message clientMessage
		err := rows.Scan(&message.Name, &message.Message, &message.Type)
		if err != nil {
			log.Printf("Ошибка чтения строки истории сообщений: %v", err)
			continue
		}
		messages = append(messages, message)
	}
	return messages, nil
}

func saveMessage(senderName, content string, types string) error {
	conn, err := pool.Acquire(context.Background())
	if err != nil {
		log.Println("Ошибка получения соединения из пула:", err)
		return err
	}
	defer conn.Release()

	_, err = conn.Exec(context.Background(),
		"INSERT INTO messages (sender_name, content,type) VALUES ($1, $2,$3)",
		senderName, content, types)
	if err != nil {
		log.Printf("Ошибка сохранения сообщения: %v", err)
		return err
	}
	return nil
}

func (c *chat) Registration(newMessage clientMessage) error {
	var existingName string

	log.Printf("Запрос на проверку существования пользователя с именем: %s", newMessage.Name)
	conn, err := pool.Acquire(context.Background())
	if err != nil {
		log.Println("Ошибка получения соединения из пула:", err)
		return fmt.Errorf("Ошибка получения соединения из пула")
	}
	defer conn.Release()

	row := conn.QueryRow(context.Background(), "SELECT name FROM users WHERE name=$1", newMessage.Name)

	err = row.Scan(&existingName)

	if err != nil {

		if err == pgx.ErrNoRows {
			log.Printf("Пользователь с таким именем не найден")

			_, err := conn.Exec(context.Background(), "INSERT INTO users (name, password) VALUES ($1, $2)", newMessage.Name, newMessage.Password)
			if err != nil {
				log.Printf("Ошибка при добавлении нового пользователя: %v", err)
				return fmt.Errorf("ошибка при добавлении нового пользователя: %v", err)
			}

			token, err := GenerateJWT(newMessage.Name, newMessage.Password, []byte("your-secret-key"))
			if err != nil {
				log.Printf("Ошибка при генерации токена: %v", err)
				return fmt.Errorf("ошибка при генерации токена: %v", err)
			}

			newMessage.Token = token
			log.Printf("Пользователь %s успешно зарегистрирован! Токен: %s", newMessage.Name, token)
			chatRoom.NotifyServerMessage(serverMessage{Message: "Регистрация успешна!", Type: "server"})

			return nil
		}

		return fmt.Errorf("ошибка при проверке пользователя: %v", err)
	}

	log.Printf("Пользователь с таким именем %s уже существует", newMessage.Name)
	return fmt.Errorf("Пользователь с таким именем уже существует")
}

func (c *Client) Update(newMessage interface{}) {

	switch msg := newMessage.(type) {
	case clientMessage:
		err := c.conn.WriteJSON(msg)
		if err != nil {
			log.Printf("Error sending client message: %v", err)
			c.conn.Close()
		}
	case serverMessage:
		err := c.conn.WriteJSON(msg)
		if err != nil {
			log.Printf("Error sending server message: %v", err)
			c.conn.Close()
		}
	default:
		log.Println("Unknown message type")
	}
}

func (c *Client) Close() {
	c.conn.Close()
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var chatRoom = &chat{}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading to WebSocket: %v", err)
		return
	}
	defer conn.Close()

	newClient := &Client{conn}

	var msg clientMessage
	err = conn.ReadJSON(&msg)
	if err != nil {
		log.Printf("Error reading initial authorization message: %v", err)
		return
	}

	if _, exists := connectedUsers[msg.Name]; exists {
		log.Printf("Пользователь %s уже подключен", msg.Name)
		conn.WriteJSON(serverMessage{Message: "Пользователь уже подключен", Type: "server"})
		return
	}

	if msg.Is_register {
		err := chatRoom.Registration(msg)
		if err != nil {

			log.Printf("Ошибка при регистрации: %v", err)

			conn.WriteJSON(serverMessage{Message: err.Error(), Type: "server"})
		} else {
			conn.WriteJSON(serverMessage{Message: "Регистрация успешна", Type: "server"})
		}
	} else {

		chatRoom.NotifyServerMessage(serverMessage{Message: "Авторизация", Type: "server"})
		if !chatRoom.AutorizationCheck(msg, newClient) {

			conn.WriteJSON(serverMessage{Message: "Неверный логин или пароль"})
			fmt.Println("Неверный логин или пароль")
		} else {
			fmt.Println("новый клиент ", msg.Name)
			conn.WriteJSON(serverMessage{Message: "Авторизация успешна", Type: "server"})
		}
	}

	s := "Присоединился к чату " + msg.Name
	chatRoom.NotifyServerMessage(serverMessage{Message: s})
	chatRoom.RegisterObserver(newClient)
	if err := saveMessage("", s, "Server"); err != nil {
		log.Printf("Ошибка сохранения сообщения в базу данных: %v", err)
	}
	history, err := getChatHistory(50)
	if err != nil {
		log.Printf("Ошибка получения истории сообщений: %v", err)
	} else {
		for _, msg := range history {
			if err := newClient.conn.WriteJSON(msg); err != nil {
				log.Printf("Ошибка отправки истории сообщения: %v", err)
			}
		}
	}

	defer func() {
		chatRoom.RemoveObserver(newClient)
		chatRoom.RemoveClientFromMap(msg.Name)
		s1 := "Вышел из чата  " + msg.Name
		chatRoom.NotifyServerMessage(serverMessage{Message: s1})

	}()

	for {
		var msg clientMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("---Error reading JSON: %v", err)
			break
		}
		log.Printf("Received: Name=%s, Message=%s", msg.Name, msg.Message)

		chatRoom.newMessage = msg
		chatRoom.NotifyObservers()
		if err := saveMessage(msg.Name, msg.Message, "client"); err != nil {
			log.Printf("Ошибка сохранения сообщения в базу данных: %v", err)
		}
	}
}

func main() {

	initDB()
	defer pool.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fs := http.FileServer(http.Dir("./dist"))
	http.Handle("/", fs)
	http.HandleFunc("/ws", handleConnections)

	server := &http.Server{Addr: ":8080"}

	shutdownComplete := make(chan struct{})

	go func() {
		log.Println("Server started on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutdown signal received")
	chatRoom.NotifyServerMessage(serverMessage{Message: "Сервер прекратил соединение"})

	stop()
	log.Println("Closing WebSocket connections...")
	chatRoom.mu.Lock()
	for _, client := range chatRoom.clients {
		client.(*Client).Close()
	}
	chatRoom.mu.Unlock()

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}

	close(shutdownComplete)
	log.Println("Server stopped gracefully")
}
