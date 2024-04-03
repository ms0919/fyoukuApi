package main

import (
	"encoding/json"
	"fmt"
	"fyoukuApi/services/mq"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	beego.LoadAppConfig("ini", "../../../conf/app.conf")
	defaultdb := beego.AppConfig.String("defaultdb")

	orm.RegisterDriver("mysql", orm.DRMySQL)
	err := orm.RegisterDataBase("default", "mysql", defaultdb)

	if err != nil {
		panic(err)
	}
	mq.ConsumerDlx(
		"fyouku.comment.count",
		"fyouku_comment_queue",
		"fyouku.comment.count.dlx",
		"fyouku_comment_queue_dlx",
		10000, //10s
		callback,
	)
}

func callback(s string) {
	o := orm.NewOrm()
	type Data struct {
		VideoId    int
		EpisodesId int
	}
	var data Data
	err := json.Unmarshal([]byte(s), &data)
	if err == nil {
		// 修改视频总评论数
		o.Raw("UPDATE video SET comment = comment + 1 WHERE id = ?", data.VideoId).Exec()
		// 修改视频剧集评论数
		o.Raw("UPDATE video_episodes SET comment = comment + 1 WHERE id = ?", data.EpisodesId).Exec()
		// 更新Redis排行榜-通过MQ
		// 发布订阅简单模式
		videoObj := map[string]int{
			"VideoId": data.VideoId,
		}
		videoJson, _ := json.Marshal(videoObj)
		_ = mq.PublishEx("ulivideo", "direct", "ulivideo.top", string(videoJson))
	}
	fmt.Println("msg is :%s\n", s)
}
