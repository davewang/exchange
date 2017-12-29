// This is the name of our package
// Everything with this package name can see everything
// else inside the same package, regardless of the file they are in
package main

// These are the libraries we are going to use
// Both "fmt" and "net" are part of the Go standard library
import (
	// "fmt" has methods for formatted I/O operations (like printing to the console)
	"fmt"
	// The "net/http" library has methods to implement HTTP clients and servers
	"net/http"
	"github.com/gorilla/mux"
	"strconv"
	"encoding/json"
    "time"
    "io"
)

var mc *MessageCenter

type Message struct {
    Uid int
    Message string
}

type MessageCenter struct {
    // 测试 没有加读写锁
    messageList []*Message
    userList map[int]chan string
}

func NewMessageCenter() *MessageCenter {
    mc := new(MessageCenter)
    mc.messageList = make([]*Message, 0, 100)
    mc.userList = make(map[int]chan string)
    return mc
}

func (mc *MessageCenter) GetMessageChan(uid int) <- chan string {
    messageChan := make(chan string)
    mc.userList[uid] = messageChan
    return messageChan
}

func (mc *MessageCenter) GetMessage(uid int) []string {
    messages := make([]string, 0, 10)
    for i, msg := range mc.messageList {
        if msg == nil {
            continue
        }
        if msg.Uid == uid {
            messages = append(messages, msg.Message)
            // 临时方案 只是测试用 应更换为list
            mc.messageList[i] = nil
        }
    }
    return messages
}

func (mc *MessageCenter) SendMessage(uid int, message string) {
    messageChan, exist := mc.userList[uid]
    if exist {
        messageChan <- message
        return
    }
    // 未考虑同一账号多登陆情况
    mc.messageList = append(mc.messageList, &Message{uid, message})
}

func (mc *MessageCenter) RemoveUser(uid int) {
    _, exist := mc.userList[uid]
    if exist {
        delete(mc.userList, uid)
    }
}


func SendMessageServer(w http.ResponseWriter, req *http.Request) {
    uid, _ := strconv.Atoi(req.FormValue("uid"))
    message := req.FormValue("message")

    mc.SendMessage(uid, message)

    io.WriteString(w, `{}`)
}

func PollMessageServer(w http.ResponseWriter, req *http.Request) {
    uid, _ := strconv.Atoi(req.FormValue("uid"))

    messages := mc.GetMessage(uid)

    if len(messages) > 0 {
        jsonData, _ := json.Marshal(map[string]interface{}{"status":0, "messages":messages})
        w.Write(jsonData)
        return
    }

    messageChan := mc.GetMessageChan(uid)

    select {
    case message := <- messageChan:
        jsonData, _ := json.Marshal(map[string]interface{}{"status":0, "messages":[]string{message}})
        w.Write(jsonData)
    case <- time.After(10 * time.Second):
        mc.RemoveUser(uid)
        jsonData, _ := json.Marshal(map[string]interface{}{"status":1, "messages":nil})
        n, err := w.Write(jsonData)
        fmt.Println(n, err)
    }
}

func main() {

	mc = NewMessageCenter()
	// Declare a new router
	r := mux.NewRouter()

	// Declare the static file directory and point it to the 
	// directory we just made
	staticFileDirectory := http.Dir("./assets/")
	staticFileHandler := http.StripPrefix("/assets/", http.FileServer(staticFileDirectory))
	// The "PathPrefix" method acts as a matcher, and matches all routes starting
	// with "/assets/", instead of the absolute route itself
	r.PathPrefix("/assets/").Handler(staticFileHandler).Methods("GET")
	// This is where the router is useful, it allows us to declare methods that
	// this path will be valid for
	r.HandleFunc("/sendmessage", SendMessageServer)
    r.HandleFunc("/pollmessage", PollMessageServer)

	// We can then pass our router (after declaring all our routes) to this method
	// (where previously, we were leaving the secodn argument as nil)
	http.ListenAndServe(":8080", r)
}

