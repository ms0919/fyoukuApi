package controllers

import (
	"fmt"
	"fyoukuApi/models"
	"regexp"
	"strconv"
	"strings"

	beego "github.com/astaxie/beego"
)

// Operations about Users

type UserController struct {
	beego.Controller
}

// SaveRegister 用户注册功能
// @router /register/save [get]
func (u *UserController) SaveRegister() {
	var (
		mobile   string
		password string
		err      error
	)

	mobile = u.GetString("mobile")
	password = u.GetString("password")

	//校验手机号和密码的合法性
	if mobile == "" {
		u.Data["json"] = ReturnError(4001, "手机号不能为空")
		u.ServeJSON()
	}
	fmt.Println("mobile:", mobile)
	isorno, _ := regexp.MatchString(`^1(3|4|5|7|8)[0-9]\d{8}$`, mobile)
	fmt.Println("结果：", isorno)
	if !isorno {
		u.Data["json"] = ReturnError(4002, "手机格式不正确")
		u.ServeJSON()
	}

	if password == "" {
		u.Data["json"] = ReturnError(4001, "密码不能为空")
		u.ServeJSON()
	}

	//查询账户是否存在
	isHasUser := models.GetUserByMobile(mobile)
	if isHasUser {
		u.Data["json"] = ReturnError(4005, "手机号已经注册")
		u.ServeJSON()
	} else {
		err = models.UserSave(mobile, MD5V(password))
		if err == nil {
			u.Data["json"] = ReturnSuccess(0, "注册账号成功", nil, 0)
			u.ServeJSON()
		} else {
			u.Data["json"] = ReturnError(5000, err)
			u.ServeJSON()
		}
	}
}

func (u *UserController) LoginDo() {
	mobile := u.GetString("mobile")
	password := u.GetString("password")

	if mobile == "" {
		u.Data["json"] = ReturnError(4001, "手机号不能为空")
		u.ServeJSON()
	}
	isorno, _ := regexp.MatchString(`^1(3|4|5|7|8)[0-9]\d{8}$`, mobile)
	if !isorno {
		u.Data["json"] = ReturnError(4002, "手机号格式不正确")
		u.ServeJSON()
	}
	if password == "" {
		u.Data["json"] = ReturnError(4003, "密码不能为空")
		u.ServeJSON()
	}
	uid, name := models.IsMobileLogin(mobile, MD5V(password))
	if uid != 0 {
		u.Data["json"] = ReturnSuccess(0, "登录成功", map[string]interface{}{"uid": uid, "username": name}, 1)
		u.ServeJSON()
	} else {
		u.Data["json"] = ReturnError(4004, "手机号或密码不正确")
		u.ServeJSON()
	}
}

// SendMessageDo 批量向用户发送通知消息
func (u *UserController) SendMessageDo() {
	uids := u.GetString("uids")
	content := u.GetString("content")

	if uids == "" {
		u.Data["json"] = ReturnError(4001, "请填写接收人~")
		u.ServeJSON()
	}
	if content == "" {
		u.Data["json"] = ReturnError(4002, "请填写发送内容")
		u.ServeJSON()
	}
	meaageId, err := models.SendMessageDo(content)
	if err != nil {
		u.Data["json"] = ReturnError(5000, "发生消息失败")
		u.ServeJSON()
	} else {
		uidConfig := strings.Split(uids, ",")
		for _, id := range uidConfig {
			//发送消息给用户，就是每一个用户写入一条记录
			uid, _ := strconv.Atoi(id)
			//这里直接将数据写入到mq异步来发送消息给用户
			//models.SendMessageUser(uid, meaageId)
			models.MQSendMessageUser(uid, meaageId)
		}
		u.Data["json"] = ReturnSuccess(0, "发送成功", "", 1)
		u.ServeJSON()
	}
}

// 批量发送通知消息
// @router /send/message [*]
type SendData struct {
	UserId    int
	MessageId int64
}

func (u *UserController) SendMessageDo1() {
	uids := u.GetString("uids")
	content := u.GetString("content")

	if uids == "" {
		u.Data["json"] = ReturnError(4001, "请填写接收人~")
		u.ServeJSON()
	}
	if content == "" {
		u.Data["json"] = ReturnError(4002, "请填写发送内容")
		u.ServeJSON()
	}

	messageId, err := models.SendMessageDo(content)
	if err == nil {
		uidConfig := strings.Split(uids, ",")
		count := len(uidConfig)

		sendChan := make(chan SendData, count)
		closeChan := make(chan bool, count)

		// 发送任务
		go func() {
			var data SendData
			for _, v := range uidConfig {
				userId, _ := strconv.Atoi(v)
				data.UserId = userId
				data.MessageId = messageId

				sendChan <- data
			}
			close(sendChan)
		}()

		for i := 0; i < 5; i++ {
			go sendMessageFunc(sendChan, closeChan)
		}

		for i := 0; i < 5; i++ {
			<-closeChan
		}

		close(closeChan)

		u.Data["json"] = ReturnSuccess(0, "发送成功", "", 1)
		u.ServeJSON()
	}
	u.Data["json"] = ReturnError(500, "发送失败，请联系客服")
	u.ServeJSON()
}

func sendMessageFunc(sendChan chan SendData, closeChan chan bool) {
	for t := range sendChan {
		fmt.Println(t)
		models.MQSendMessageUser(t.UserId, t.MessageId)
	}
	closeChan <- true
}
