package main

import (
	"bytes"
	"container/heap"
	"crypto/sha1"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// global constants
const (
	WxAppID     = "wxfcb4cf7361481dba"
	WxAppSecret = "70ce64f8b29014fc669b8e5d1e0f6544"
	WxToken     = "cyclone"
)

// global variables
var (
	gUserRobotMap UserRobotRel
	gMQGuard      sync.Mutex
	gRobotMQ      = make(map[string]RobotMessageHeap)
)

func main() {
	log.Println("[SERVER] run")
	mux := http.NewServeMux()
	mux.Handle("/wx", http.HandlerFunc(handleWechat))
	mux.Handle("/cyclone/wxmsg", http.HandlerFunc(handleCyclone))
	server := &http.Server{
		Addr:    ":80",
		Handler: mux,
	}
	log.Printf("[SERVER] listening on port%s\n", server.Addr)
	log.Printf("[ERROR] server: stop unexpectedly: %s\n", server.ListenAndServe().Error())
}

func handleCyclone(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		log.Printf("[SERVER] %s %s\n", r.Method, r.URL.RequestURI())
		handlerReadMsg(w, r)
	}
	if r.Method == "POST" {
		data, _ := ioutil.ReadAll(r.Body)
		log.Printf("[SERVER] %s %s\n%s", r.Method, r.URL.RequestURI(), data)
		var msgReq struct {
			ToWechatID string `json:"wxid"`
			Content    string `json:"text"`
		}
		if err := json.Unmarshal(data, &msgReq); err != nil {
			http.Error(w, fmt.Sprintf("can't parse post data as json: %v", err), http.StatusBadRequest)
			return
		}
		if err := sendCustomMessage(msgReq.ToWechatID, msgReq.Content); err != nil {
			http.Error(w, fmt.Sprintf("failed to send custom message: %v", err), http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte("custom message send successfully!"))
	}
}

func handlerReadMsg(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	robotID := r.Form.Get("robot")
	if robotID == "" {
		http.Error(w, "missing request string field 'robot'", http.StatusBadRequest)
		return
	}
	numStr := r.Form.Get("n")
	num, err := strconv.ParseInt(numStr, 0, 0)
	if err != nil {
		http.Error(w, "wrong request number field 'n'", http.StatusBadRequest)
		return
	}
	var ret struct {
		Size    int             `json:"size"`
		MsgList []*RobotMessage `json:"messages"`
	}
	gMQGuard.Lock()
	defer gMQGuard.Unlock()
	mq, ok := gRobotMQ[robotID]
	log.Printf("[CYLONE] ACQ package from <%s>, capacity=%d", robotID, len(mq))
	if !ok || len(mq) == 0 {
		ret.Size = 0
	} else {
		calc := func(num, size int) int {
			if num > 0 && num < size {
				return num
			}
			return size
		}
		n := calc(int(num), len(mq))
		ret.Size = n
		for i := 0; i < n; i++ {
			ret.MsgList = append(ret.MsgList, heap.Pop(&mq).(*RobotMessage))
		}
		gRobotMQ[robotID] = mq
		log.Printf("[CYLONE] POP %d messages", n)
	}
	data, err := json.Marshal(&ret)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to encode reply msg as json: %v", err),
			http.StatusInternalServerError)
		return
	}
	w.Write(data)
}

func handleWechat(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		log.Printf("[SERVER] %s %s\n", r.Method, r.URL.RequestURI())
		r.ParseForm()
		resp := ""
		if verifySignature(r) {
			log.Println("[WECHAT] signature verified!")
			resp = r.Form.Get("echostr")
		} else {
			log.Println("[ERROR] failed to verify signature!")
		}
		log.Printf("[SERVER] response =>\n%q", resp)
		w.Write([]byte(resp))
	}
	if r.Method == "POST" {
		data, _ := ioutil.ReadAll(r.Body)
		log.Printf("[SERVER] %s %s\n%s", r.Method, r.URL.RequestURI(), data)
		r.ParseForm()
		if !verifySignature(r) {
			log.Println("[ERROR] failed to verify signature!")
			http.Error(w, "", http.StatusBadRequest)
			return
		}
		log.Println("[WECHAT] signature verified!")
		msg := WechatMessage{}
		if err := xml.Unmarshal(data, &msg); err != nil {
			log.Printf("[ERROR] can't parse post data as xml: %v", err)
			http.Error(w, "", http.StatusBadRequest)
			return
		}
		log.Printf("[WECHAT] FROM<%s>: %s", msg.FromUserName, msg.Content)
		resp := ""
		if msg.MsgType == "text" {
			resp = makeTextMsg(msg.FromUserName, msg.ToUserName, respTextMsg(&msg))
		}
		log.Printf("[SERVER] response =>\n%q", resp)
		w.Write([]byte(resp))
	}
}

var robotIDMatcher = regexp.MustCompile(`^@(\S+)$`)

func respTextMsg(msg *WechatMessage) string {
	if robotIDMatcher.MatchString(msg.Content) {
		matches := robotIDMatcher.FindStringSubmatch(msg.Content)
		if len(matches) == 2 {
			userID := msg.FromUserName
			robotID := matches[1]
			log.Printf("[WECHAT] LINK<%s>: <%s>", userID, robotID)
			gUserRobotMap.Link(userID, robotID)
			return fmt.Sprintf("关联成功!接下来的消息将转发至RPA机器人%q", robotID)
		}
	}
	robotID, ok := gUserRobotMap.Get(msg.FromUserName)
	if !ok {
		return fmt.Sprint("尚未关联RPA机器人, 请回复\"@+机器人编号\"进行关联")
	}
	robotMsg := new(RobotMessage)
	robotMsg.FromWechatID = msg.FromUserName
	robotMsg.Content = msg.Content
	robotMsg.CreateTime = msg.CreateTime
	gMQGuard.Lock()
	defer gMQGuard.Unlock()
	mq, ok := gRobotMQ[robotID]
	if !ok {
		mq = make(RobotMessageHeap, 0)
	}
	heap.Push(&mq, robotMsg)
	gRobotMQ[robotID] = mq
	log.Printf("[CYLONE] NEW package into <%s>, capacity=%d", robotID, len(mq))
	return ""
}

func makeTextMsg(toUser, fromUser, content string) string {
	if content == "" {
		return ""
	}
	reply := WechatMessage{}
	reply.XMLName.Local = "xml"
	reply.ToUserName = toUser
	reply.FromUserName = fromUser
	reply.CreateTime = time.Now().Unix()
	reply.MsgType = "text"
	reply.Content = content
	data, err := xml.Marshal(&reply)
	log.Printf("[WECHAT] - TO<%s>: %s", reply.ToUserName, reply.Content)
	if err != nil {
		log.Printf("[WECHAT] failed to encode reply msg as xml: %v", err)
		return ""
	}
	return string(data)
}

func verifySignature(r *http.Request) bool {
	vlist := make([]string, 0)
	vlist = append(vlist, WxToken)
	vlist = append(vlist, r.Form.Get("timestamp"))
	vlist = append(vlist, r.Form.Get("nonce"))
	sort.Strings(vlist)
	hash := sha1.New()
	hash.Write([]byte(strings.Join(vlist, "")))
	hashcode := fmt.Sprintf("%x", hash.Sum(nil))
	signature := r.Form.Get("signature")
	return hashcode == signature
}

var gAccessToken struct {
	sync.Mutex
	Token      string
	ExpireTime time.Time
}

func getAccessToken() string {
	gAccessToken.Lock()
	defer gAccessToken.Unlock()
	if time.Now().Before(gAccessToken.ExpireTime) {
		return gAccessToken.Token
	}
	log.Println("[WECHAT] access token expired, get new one!")
	const url = "https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s"
	gAccessToken.Token = ""
	retry := 0
	for retry <= 3 {
		if retry > 0 {
			log.Printf("[WECHAT] try to get access token again, count=%d", retry)
			time.Sleep(2 * time.Second)
		}
		retry++
		resp, err := http.Get(fmt.Sprintf(url, WxAppID, WxAppSecret))
		if err != nil {
			log.Printf("[ERROR] failed to connect access token server: %v", err)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			log.Printf("[ERROR] failed to connect access token server: %s", resp.Status)
			continue
		}
		data, _ := ioutil.ReadAll(resp.Body)
		var access struct {
			Token       string `json:"access_token"`
			ExpireInSec int    `json:"expires_in"`
		}
		if err := json.Unmarshal(data, &access); err != nil || access.Token == "" {
			log.Printf("[ERROR] can't parse access token as json: %v, body =>\n%s", err, data)
			continue
		}
		gAccessToken.Token = access.Token
		gAccessToken.ExpireTime = time.Now().Add(
			time.Duration(float64(access.ExpireInSec)*0.8) * time.Second)
		log.Printf("[WECHAT] token=%q, expireInSec=%d, expireTime=%q", gAccessToken.Token,
			access.ExpireInSec, gAccessToken.ExpireTime.Format("06/01/02 15:04:05"))
		break
	}
	if gAccessToken.Token == "" {
		log.Printf("[ERROR] failed to get access token after retry %d times", retry-1)
	}
	return gAccessToken.Token
}

func sendCustomMessage(uid string, content string) error {
	log.Printf("[WECHAT] * TO<%s>: %s", uid, content)
	var post struct {
		ToUser  string `json:"touser"`
		MsgType string `json:"msgtype"`
		Text    struct {
			Content string `json:"content"`
		} `json:"text"`
	}
	post.ToUser = uid
	post.MsgType = "text"
	post.Text.Content = content
	data, err := json.Marshal(&post)
	if err != nil {
		return fmt.Errorf("failed to encode reply msg as json: %v", err)
	}
	acToken := getAccessToken()
	const url = "https://api.weixin.qq.com/cgi-bin/message/custom/send?access_token=%s"
	resp, err := http.Post(fmt.Sprintf(url, acToken), "application/json;charset=utf-8", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to connect custom message server: %v", err)
	}
	body, _ := ioutil.ReadAll(resp.Body)
	var status struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	perr := json.Unmarshal(body, &status)
	if perr != nil {
		return fmt.Errorf("can't parse custom message send response as json: %v, body =>\n%s", err, body)
	}
	if status.ErrCode != 0 {
		return fmt.Errorf("can't send custom message: errno=%d, msg=%q", status.ErrCode, status.ErrMsg)
	}
	return nil
}

// RobotMessageHeap represents message queue
type RobotMessageHeap []*RobotMessage

func (h RobotMessageHeap) Len() int { return len(h) }

func (h RobotMessageHeap) Less(i, j int) bool { return h[i].CreateTime < h[j].CreateTime }

func (h RobotMessageHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

// Push ...
func (h *RobotMessageHeap) Push(x interface{}) {
	*h = append(*h, x.(*RobotMessage))
}

// Pop ...
func (h *RobotMessageHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// UserRobotRel represents user-robot id link map
type UserRobotRel struct {
	sync.Mutex
	Data map[string]string
}

// Link user with robot
func (l *UserRobotRel) Link(wechatID, robotID string) {
	l.Lock()
	defer l.Unlock()
	if l.Data == nil {
		l.Data = make(map[string]string)
	}
	l.Data[wechatID] = robotID
}

// Get robot id by user id
func (l *UserRobotRel) Get(wechatID string) (robotID string, isExist bool) {
	l.Lock()
	defer l.Unlock()
	if l.Data == nil {
		isExist = false
		return
	}
	robotID, isExist = l.Data[wechatID]
	return
}

// WechatMessage represents wechat XML text message
type WechatMessage struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   `xml:"ToUserName,CDATA"`
	FromUserName string   `xml:"FromUserName,CDATA"`
	CreateTime   int64    `xml:"CreateTime"`
	MsgType      string   `xml:"MsgType,CDATA"`
	Content      string   `xml:"Content,CDATA"`
	MsgID        string   `xml:"MsgId"`
}

// RobotMessage represents message send to RPA robot
type RobotMessage struct {
	FromWechatID string `json:"wxid"`
	CreateTime   int64  `json:"create"`
	Content      string `json:"text"`
}

// curl -s -i -H "Content-Type:application/json" -X POST -d '{"wxid":"ogYaks-6NHyXk5vWpOiYEcmBLswM", "text":"测试"}' http://118.24.78.86/cyclone/wxmsg
