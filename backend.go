package main

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"
	"github.com/toorop/gin-logrus"
	"io"
	"os"
)
import "github.com/go-resty/resty/v2"

// Fill In Blanks
const VerifyToken = ""
const AppID = ""
const AppSecret = ""

// 飞书用户:"open.feishu.cn/"
const DeveloperHost = "open.larksuite.com"

type EventBase struct { //v1.0 Scheme
	Ts        string                 `json:"ts"`
	UUID      string                 `json:"uuid"`
	Token     string                 `json:"token" binding:"required"`
	Type      string                 `json:"type" binding:"required"`
	Event     map[string]interface{} `json:"event"`
	Challenge string                 `json:"challenge"`
}
type APICallback struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}
type MessageContent struct {
	Text string `json:"text"`
}
type SendMessageBase struct {
	OpenId  string         `json:"open_id,omitempty"`
	RootId  string         `json:"root_id,omitempty"`
	ChatId  string         `json:"chat_id,omitempty"`
	UserId  string         `json:"user_id,omitempty"`
	Email   string         `json:"email,omitempty"`
	MsgType string         `json:"msg_type"`
	Content MessageContent `json:"content"`
}

const (
	Challenge = "url_verification"
	// Callback JSON message:{
	//    "uuid": "41b5f371157e3d5341b38b20396e77e3",
	//    "token": "2g7als3DgPW6Xp1xEpmcvgVhQG621bFY",//校验Token
	//    "ts": "1550038209.428520",  //时间戳
	//    "type": "event_callback",//事件回调此处固定为event_callback
	//    "event": {
	//        "type": "message", // 事件类型
	//        "app_id": "cli_xxx",
	//        "tenant_key": "xxx", //企业标识
	//        "root_id": "",
	//        "parent_id": "",
	//        "open_chat_id": "oc_5ce6d572455d361153b7cb51da133945",
	//        "chat_type": "private", //私聊private，群聊group
	//        "msg_type": "text",    //消息类型
	//        "open_id": "ou_18eac85d35a26f989317ad4f02e8bbbb",
	//        "employee_id": "xxx", // 即“用户ID”，仅企业自建应用会返回
	//        "union_id": "xxx",
	//        "open_message_id": "om_36686ee62209da697d8775375d0c8e88",
	//        "is_mention": false,
	//        "text": "<at open_id="xxx">@小助手</at> 消息内容 <at open_id="yyy">@张三</at>",      // 消息文本，可能包含被@的人/机器人。
	//        "text_without_at_bot":"消息内容 <at open_id="yyy">@张三</at>" //消息内容，会过滤掉at你的机器人的内容
	//   }
	//}
	Callback = "event_callback"
)
const (
	Start   = "p2p_chat_create"
	Message = "message"
)

var log = logrus.New()
var tokenService TokenService
var client = resty.New()

func init() {
	r, _ := rotatelogs.New("log." + "%Y%m%d")
	mw := io.MultiWriter(os.Stdout, r)
	log.SetOutput(mw)
}
func callbackHandler(message EventBase) error {
	token := tokenService.Token()
	switch message.Event["type"] {
	case Start:
		sender := message.Event["chat_id"].(string)
		resp := &APICallback{}
		_, err := client.R().SetAuthToken(token).SetResult(resp).SetBody(SendMessageBase{ChatId: sender, MsgType: "text", Content: MessageContent{Text: "Hello!"}}).Post("/open-apis/message/v4/send/")
		if err != nil {
			log.Error(err)
			return err
		}
		if resp.Code != 0 {
			err = errors.New("SendMessage API Request Error" + resp.Msg)
			log.Error(err)
			return err
		}

	case Message:
		resp := &APICallback{}
		open_chat_id, ok := message.Event["chat_id"].(string)
		if !ok {
			open_chat_id = ""

		}
		open_id, ok := message.Event["open_id"].(string)
		if !ok {
			open_id = ""
		}
		open_message_id := message.Event["open_message_id"].(string)
		text := message.Event["text_without_at_bot"].(string)
		_, err := client.R().SetAuthToken(token).SetResult(resp).SetBody(SendMessageBase{ChatId: open_chat_id, OpenId: open_id, RootId: open_message_id, MsgType: "text", Content: MessageContent{Text: text}}).Post("/open-apis/message/v4/send/")
		if err != nil {
			log.Error(err)
			return err
		}
		if resp.Code != 0 {
			err = errors.New("SendMessage API Request Error: " + resp.Msg)
			log.Error(err)
			return err
		}

	}
	return nil

}
func eventHandler(c *gin.Context) {
	var base EventBase
	// 如果要多次解析(比如兼容V2 Scheme),使用ShouldBindBodyWith
	err := c.ShouldBindJSON(&base)
	if err != nil {
		//
		c.JSON(400, gin.H{"code": -1})
		return
	}
	if base.Token != VerifyToken {
		c.JSON(418, gin.H{
			"code": 1,
		})

	}
	switch base.Type {
	case Challenge:
		c.JSON(200, gin.H{
			"challenge": base.Challenge,
		})
	case Callback:
		err = callbackHandler(base)
		if err != nil {
			c.JSON(500, gin.H{"code": -1})
		}

		c.JSON(200, gin.H{"code": 0})

	}

}

func main() {
	client.SetHostURL(fmt.Sprintf("https://%s", DeveloperHost))
	//client.SetAuthToken("")
	tokenService = TokenService{AppId: AppID, AppSecret: AppSecret}
	err := tokenService.Start()
	if err != nil {
		log.Fatal(err)
		return
	}
	r := gin.New()
	r.Use(ginlogrus.Logger(log), gin.Recovery())

	r.POST("/events", eventHandler)
	r.Run("0.0.0.0:8123")

}
