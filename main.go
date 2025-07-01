package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"bufio"

	qrcodeTerminal "github.com/Baozisoftware/qrcode-terminal-go"
	"github.com/go-resty/resty/v2"
	"github.com/skip2/go-qrcode"
)

const (
	ua               = "Mozilla/5.0 (Linux; Android 13; Pixel 6 Build/HWA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.6099.199 Mobile Safari/537.36"
	appKey           = "1d8b6e7d45233436"
	appSecret        = "560c52ccd288fed045859ed18bffd973"
	SecondsPerMinute = 60
	SecondsPerHour   = SecondsPerMinute * 60
	SecondsPerDay    = SecondsPerHour * 24
)

var (
	config                                   = &Config{}
	client                                   = resty.New() // APP端客户端
	webClient                                = resty.New() // Web端客户端
	login                                    = resty.New()
	cookies                                  []*http.Cookie
	orderId                                  string
	itemName                                 string
	strStartTime                             string
	qrcodeKey                                string
	qrCodeUrl                                string
	fileName                                 string
	startTime, waitTime, errorTime, fastTime int64
	bp, price                                float64
	rankInfo                                 *Rank
)

// 登录相关结构体 - 新版Web端扫码登录API
type GetLoginUrl struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Ttl     int    `json:"ttl"`
	Data    struct {
		Url       string `json:"url"`
		QrcodeKey string `json:"qrcode_key"`
	} `json:"data"`
}

type GetLoginInfo struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Url          string `json:"url"`
		RefreshToken string `json:"refresh_token"`
		Timestamp    int64  `json:"timestamp"`
		Code         int    `json:"code"`
		Message      string `json:"message"`
	} `json:"data"`
}

// 配置文件结构体
type Config struct {
	BpEnough    bool   `json:"bp_enough"`
	BuyNum      string `json:"buy_num"`
	CouponToken string `json:"coupon_token"`
	Device      string `json:"device"`
	ItemId      string `json:"item_id"`
	TimeBefore  int    `json:"time_before"`
	Cookies     struct {
		SESSDATA        string `json:"SESSDATA"`
		BiliJct         string `json:"bili_jct"`
		DedeUserID      string `json:"DedeUserID"`
		DedeUserIDCkMd5 string `json:"DedeUserID__ckMd5"`
	} `json:"cookies"`
	// 扩展支持原项目的功能
	TargetMode struct {
		TargetID int64 `json:"target_id"`
		LimitV   int64 `json:"limit_v"`
		BuyNum   int64 `json:"buy_num"`
	} `json:"target_mode"`
	TimedMode struct {
		BuyNum int64 `json:"buy_num"`
	} `json:"timed_mode"`
}

// API响应结构体
type Details struct {
	Data struct {
		Name       string `json:"name"`
		Properties struct {
			SaleTimeBegin    string `json:"sale_time_begin"`
			SaleBpForeverRaw string `json:"sale_bp_forever_raw"`
		}
		CurrentActivity struct {
			PriceBpForever float64 `json:"price_bp_forever"`
		} `json:"current_activity"`
	} `json:"data"`
}

type Now struct {
	Data struct {
		Now int64 `json:"now"`
	} `json:"data"`
}

type Navs struct {
	Code int `json:"code"`
	Data struct {
		Wallet struct {
			BcoinBalance float64 `json:"bcoin_balance"`
		} `json:"wallet"`
		Uname string `json:"uname"`
	} `json:"data"`
}

type Asset struct {
	Data struct {
		Id   int `json:"id"`
		Item struct {
			ItemId int `json:"item_id"`
		} `json:"item"`
	} `json:"data"`
}

type Rank struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Ttl     int    `json:"ttl"`
	Data    struct {
		Rank []struct {
			Mid      int    `json:"mid"`
			Nickname string `json:"nickname"`
			Avatar   string `json:"avatar"`
			Number   int    `json:"number"`
		} `json:"rank"`
	} `json:"data"`
}

type Create struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Ttl     int    `json:"ttl"`
	Data    struct {
		OrderId  string `json:"order_id"`
		State    string `json:"state"`
		BpEnough int    `json:"bp_enough"`
	} `json:"data"`
}

type Query struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Ttl     int    `json:"ttl"`
	Data    struct {
		OrderId  string `json:"order_id"`
		Mid      int    `json:"mid"`
		Platform string `json:"platform"`
		ItemId   int    `json:"item_id"`
		PayId    string `json:"pay_id"`
		State    string `json:"state"`
	} `json:"data"`
}

type Wallet struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Ttl     int    `json:"ttl"`
	Data    struct {
		BcoinBalance  float64 `json:"bcoin_balance"`
		CouponBalance int     `json:"coupon_balance"`
	} `json:"data"`
}

type SuitAsset struct {
	Data struct {
		Fan struct {
			IsFan      bool   `json:"is_fan"`
			Token      string `json:"token"`
			Number     int    `json:"number"`
			Color      string `json:"color"`
			Name       string `json:"name"`
			LuckItemId int    `json:"luck_item_id"`
			Date       string `json:"date"`
		} `json:"fan"`
	} `json:"data"`
}

// 登录实现
func webLogin() {
	var mode int
	log.Println("暂未检测到 SESSDATA, 需要进行扫码登录，请选择登陆模式 (输入数字).")
	fmt.Println("\t1. 终端中生成二维码.")
	fmt.Println("\t2. 当前目录下生成二维码图片.")
	fmt.Println("\t3. APP 打开 URL 登陆.")

Loop:
	_, err := fmt.Scanf("%v", &mode)
	checkErr(err)

	switch mode {
	case 1:
		getLoginUrl()
		qrcodeTerminal.New().Get(qrCodeUrl).Print()
		getLoginInfo()
	case 2:
		getLoginUrl()
		err := qrcode.WriteFile(qrCodeUrl, qrcode.Medium, 256, "qr.png")
		checkErr(err)
		log.Println("已在当前目录生成 qr.png")
		getLoginInfo()
	case 3:
		getLoginUrl()
		log.Printf("请打开 APP 并访问: %v", qrCodeUrl)
		getLoginInfo()
	default:
		log.Println("输入错误，请重新选择")
		goto Loop
	}
}

func getLoginUrl() {
	g := &GetLoginUrl{}
	r, err := login.R().
		SetResult(g).
		SetHeader("User-Agent", ua).
		Get("/x/passport-login/web/qrcode/generate")

	checkErr(err)

	log.Printf("申请二维码API响应状态: %d", r.StatusCode())
	if g.Code != 0 {
		log.Printf("申请二维码失败: Code=%d, Message=%s", g.Code, g.Message)
		log.Fatalln("申请二维码失败，请检查网络连接")
	}

	qrCodeUrl = g.Data.Url
	qrcodeKey = g.Data.QrcodeKey
	log.Printf("申请二维码成功，密钥: %s", qrcodeKey[:8]+"...")
}

// 获取二维码状态
func getLoginInfo() {
	for {
		task := time.NewTimer(3 * time.Second)

		g := &GetLoginInfo{}
		r, err := login.R().
			SetResult(g).
			SetQueryParam("qrcode_key", qrcodeKey).
			SetHeader("User-Agent", ua).
			Get("/x/passport-login/web/qrcode/poll")

		checkErr(err)

		log.Printf("扫码状态检查: Code=%d, Message=%s", g.Data.Code, g.Data.Message)

		switch g.Data.Code {
		case 0: // 扫码登录成功
			log.Println("扫码登录成功！")
			cookies = r.Cookies()
			for _, cookie := range cookies {
				switch cookie.Name {
				case "SESSDATA":
					config.Cookies.SESSDATA = cookie.Value
				case "bili_jct":
					config.Cookies.BiliJct = cookie.Value
				case "DedeUserID":
					config.Cookies.DedeUserID = cookie.Value
				case "DedeUserID__ckMd5":
					config.Cookies.DedeUserIDCkMd5 = cookie.Value
				}
			}

			result, err := json.MarshalIndent(config, "", " ")
			checkErr(err)

			err = os.WriteFile(fileName, result, 0644)
			checkErr(err)

			log.Println("登录信息已保存到配置文件")
			return

		case 86101: // 未扫码
			log.Println("等待扫码...")

		case 86090: // 二维码已扫码未确认
			log.Println("扫码成功，请在手机上确认登录")

		case 86038: // 二维码已失效
			log.Fatalln("二维码已失效，请重新运行程序")

		default:
			log.Printf("未知状态码: %d, 消息: %s", g.Data.Code, g.Data.Message)
		}

		<-task.C
	}
}

func nav() {
	params := map[string]string{
		"csrf": config.Cookies.BiliJct,
	}

	navs := &Navs{}
	r, err := webClient.R().
		SetResult(navs).
		SetQueryParams(params).
		SetHeader("Referer", "https://www.bilibili.com").
		Get("/web-interface/nav")
	checkErr(err)

	log.Printf("导航API响应状态: %d", r.StatusCode())
	if navs.Code == -101 {
		log.Fatalln("帐号未登录，请检查cookies.")
	} else if navs.Code != 0 {
		log.Printf("导航API返回错误: Code=%d, Response=%s", navs.Code, r.String())
		log.Fatalln("获取用户信息失败，请检查网络连接和cookies.")
	}

	bp = navs.Data.Wallet.BcoinBalance
	uname := navs.Data.Uname
	log.Printf("登录成功, 当前帐号: %v, B币余额为: %v.", uname, bp)
}

func popup() {
	params := map[string]string{
		"csrf": config.Cookies.BiliJct,
	}

	r, err := webClient.R().
		SetQueryParams(params).
		Get("/garb/popup")
	checkErr(err)

	log.Printf("弹窗API响应状态: %d", r.StatusCode())
}

func detail() {
	params := map[string]string{
		"csrf":    config.Cookies.BiliJct,
		"item_id": config.ItemId,
		"part":    "suit",
	}

	details := &Details{}
	r, err := webClient.R().
		SetQueryParams(params).
		SetResult(details).
		Get("/garb/v2/mall/suit/detail")
	checkErr(err)

	log.Printf("装扮详情API响应状态: %d", r.StatusCode())

	itemName = details.Data.Name
	strStartTime = details.Data.Properties.SaleTimeBegin
	startTime, err = strconv.ParseInt(strStartTime, 10, 64)
	if err != nil {
		log.Printf("解析开售时间失败: %v, 原始时间: %s", err, strStartTime)
		checkErr(err)
	}

	if details.Data.CurrentActivity.PriceBpForever == 0 {
		p, _ := strconv.ParseFloat(details.Data.Properties.SaleBpForeverRaw, 64)
		price = p / 100
	} else {
		price = details.Data.CurrentActivity.PriceBpForever / 100
	}

	log.Printf("装扮名称: %v，开售时间: %v, 价格: %.2f B币.", details.Data.Name, startTime, price)
	if config.BpEnough == true {
		if price > bp {
			log.Fatalf("您没有足够的钱钱，购买此装扮需要 %.2f B币.", price)
		}
	} else if config.BpEnough == false {
		if price > bp {
			log.Printf("您没有足够的钱钱，购买此装扮需要 %.2f B币.\n", price)
		}
	}
}

func asset() {
	// 根据bilibili-API-collect标准，改为Web端调用
	params := map[string]string{
		"csrf":    config.Cookies.BiliJct,
		"item_id": config.ItemId,
		"part":    "suit",
	}

	assetData := &Asset{}
	r, err := webClient.R().
		SetQueryParams(params).
		SetResult(assetData).
		Get("/garb/user/asset")
	checkErr(err)

	log.Printf("用户资产API响应状态: %d", r.StatusCode())
	if r.StatusCode() != 200 {
		log.Printf("用户资产API调用失败，可能该API已更新或废弃")
	}
}

func state() {
	// 根据bilibili-API-collect标准，改为Web端调用
	params := map[string]string{
		"csrf":    config.Cookies.BiliJct,
		"item_id": config.ItemId,
		"part":    "suit",
	}

	r, err := webClient.R().
		SetQueryParams(params).
		Get("/garb/user/reserve/state")
	checkErr(err)

	log.Printf("预约状态API响应状态: %d", r.StatusCode())
	if r.StatusCode() != 200 {
		log.Printf("预约状态API调用失败，可能该API已更新或废弃")
	}
}

func rank() {
	// 根据bilibili-API-collect标准，改为Web端调用
	params := map[string]string{
		"csrf":    config.Cookies.BiliJct,
		"item_id": config.ItemId,
	}

	ranks := &Rank{}
	r, err := webClient.R().
		SetQueryParams(params).
		SetResult(ranks).
		Get("/garb/rank/fan/recent")
	checkErr(err)

	log.Printf("排行榜API响应状态: %d", r.StatusCode())
	if r.StatusCode() == 200 && ranks.Code != 0 {
		log.Printf("排行榜API返回错误: Code=%d, Message=%s", ranks.Code, ranks.Message)
	} else if r.StatusCode() != 200 {
		log.Printf("排行榜API调用失败，可能该API已更新或废弃")
	}

	rankInfo = ranks
}

func stat() {
	// 根据bilibili-API-collect标准，改为Web端调用
	params := map[string]string{
		"csrf":    config.Cookies.BiliJct,
		"item_id": config.ItemId,
	}

	r, err := webClient.R().
		SetQueryParams(params).
		Get("/garb/order/user/stat")
	checkErr(err)

	log.Printf("订单统计API响应状态: %d", r.StatusCode())
	if r.StatusCode() != 200 {
		log.Printf("订单统计API调用失败，可能该API已更新或废弃")
	}
}

func coupon() {
	// 根据bilibili-API-collect标准，改为Web端调用
	params := map[string]string{
		"csrf":    config.Cookies.BiliJct,
		"item_id": config.ItemId,
	}

	r, err := webClient.R().
		SetQueryParams(params).
		Get("/garb/coupon/usable")
	checkErr(err)

	log.Printf("优惠券API响应状态: %d", r.StatusCode())
	if r.StatusCode() != 200 {
		log.Printf("优惠券API调用失败，可能该API已更新或废弃")
	}
}

func create() {
Loop:
	for {
		// 1s 循环一次
		task := time.NewTimer(1 * time.Second)

		// 构建APP端请求参数（保持APP签名）
		params := map[string]string{
			"add_month":    "-1",
			"buy_num":      config.BuyNum,
			"coupon_token": "",
			"csrf":         config.Cookies.BiliJct,
			"currency":     "bp",
			"item_id":      config.ItemId,
			"platform":     config.Device,
			"mobi_app":     "android",
			"build":        "7490300",
		}

		// 对参数进行APP签名
		signedParams := signParams(params)

		creates := &Create{}
		r, err := client.R().
			SetFormData(signedParams).
			SetResult(creates).
			EnableTrace().
			Post("/garb/v2/trade/create")
		checkErr(err)
		log.Printf("本次请求用时: %v.", r.Request.TraceInfo().TotalTime)

		// 添加详细响应日志
		log.Printf("装扮购买API响应: Code=%d, Message=%s", creates.Code, creates.Message)
		if creates.Data.OrderId != "" {
			log.Printf("订单ID: %s, 状态: %s", creates.Data.OrderId, creates.Data.State)
		}

		switch creates.Code {
		case 0: // 成功
			if creates.Data.BpEnough == -1 {
				log.Println(r)
				log.Fatalln("余额不足.")
			}
			orderId = creates.Data.OrderId
			if orderId == "" {
				log.Printf("APP端API返回空订单ID，尝试使用Web端API")
				log.Println("APP端响应内容:", r.String())
				// 尝试使用Web端API
				webOrderId := createWebOrder()
				if webOrderId != "" {
					orderId = webOrderId
					log.Printf("Web端API成功创建订单: %s", orderId)
				} else {
					log.Printf("Web端API也失败，可能是余额不足或其他限制")
				}
			}
			if creates.Data.State != "paying" {
				log.Printf("订单创建成功，状态: %s, 订单ID: %s", creates.Data.State, orderId)
			}
			break Loop
		case -400:
			log.Printf("请求参数错误: %s", creates.Message)
			log.Fatalln(r)
		case -403: //号被封了
			log.Fatalln("您已被封禁.")
		case -404:
			log.Printf("接口不存在或商品不存在")
			log.Fatalln(r)
		case 26102: //商品不存在，可能是未到抢购时间，立即重新执行
			errorTime += 1
			if errorTime >= 5 {
				log.Fatalln("失败次数已达到五次，退出执行...")
			}
			log.Printf("商品不存在或未到抢购时间，重试中... (%d/5)", errorTime)
			log.Println(r)
			task.Reset(0)
			create()
		case 26103: //商品库存不足
			log.Printf("商品库存不足")
			log.Fatalln(r)
		case 26104: //商品已下架
			log.Printf("商品已下架")
			log.Fatalln(r)
		case 26105: //超出购买限制
			log.Printf("超出购买限制")
			log.Fatalln(r)
		case 26106: //购买数量达到上限
			log.Printf("购买数量达到上限")
			log.Fatalln(r)
		case 26107: //商品限时购买已结束
			log.Printf("商品限时购买已结束")
			log.Fatalln(r)
		case 26108: //账号未达到购买条件
			log.Printf("账号未达到购买条件")
			log.Fatalln(r)
		case 26120: //请求频率过快，等一下执行，需要测试是否延迟执行
			fastTime++
			if fastTime >= 5 {
				log.Println(r)
				log.Fatalln("请求频率过快！失败次数已达到五次，退出执行...")
			}
			log.Printf("请求频率过快，等待后重试... (%d/5)", fastTime)
			log.Println(r)
		case 26113: //号被封了
			log.Fatalln("当前设备/账号/环境存在风险，暂时无法下单.")
		case 26134: //当前抢购人数过多，风控等级大于26135，此时无法购买
			errorTime += 1
			if errorTime >= 5 {
				log.Println(r)
				log.Fatalln("失败次数已达到五次，退出执行...")
			}
			log.Printf("当前抢购人数过多，等待后重试... (%d/5)", errorTime)
			log.Println(r)
			task.Reset(500 * time.Millisecond)
			create()
		case 26135: //当前抢购人数过多，失败四次或者锁四秒后能够购买
			errorTime += 1
			if errorTime >= 5 {
				log.Println(r)
				log.Fatalln("失败次数已达到五次，退出执行...")
			}
			log.Printf("抢购人数过多，等待后重试... (%d/5)", errorTime)
			log.Println(r)
			task.Reset(500 * time.Millisecond)
			create()
		case 69949: //老风控代码，疑似封锁设备
			errorTime += 1
			log.Println(r)
			log.Printf("触发设备风控69949，尝试重试... (%d/5)", errorTime)
			go coupon()
			if errorTime >= 5 {
				log.Fatalln("失败次数已达到五次，退出执行...")
			}
		case 88000: //新增的风控码
			errorTime += 1
			log.Printf("触发新风控88000，等待后重试... (%d/5)", errorTime)
			log.Println(r)
			go coupon()
			if errorTime >= 5 {
				log.Fatalln("失败次数已达到五次，退出执行...")
			}
		case -412: //风控拦截
			errorTime += 1
			log.Printf("触发风控拦截-412，等待后重试... (%d/5)", errorTime)
			log.Println(r)
			if errorTime >= 5 {
				log.Fatalln("失败次数已达到五次，退出执行...")
			}
		default:
			errorTime += 1
			log.Printf("未知错误码: %d, 消息: %s, 重试中... (%d/5)", creates.Code, creates.Message, errorTime)
			log.Println(r)
			go coupon()
			if errorTime >= 5 {
				log.Fatalln("失败次数已达到五次，退出执行...")
			}
		}
		<-task.C
	}
}

// Web端购买API函数（根据bilibili-API-collect标准）
func createWebOrder() string {
	// 使用Web端API参数
	params := map[string]string{
		"add_month":    "-1",
		"buy_num":      config.BuyNum,
		"coupon_token": "",
		"csrf":         config.Cookies.BiliJct,
		"currency":     "bp",
		"item_id":      config.ItemId,
		"platform":     config.Device,
	}

	creates := &Create{}
	r, err := webClient.R().
		SetFormData(params).
		SetResult(creates).
		EnableTrace().
		Post("/garb/v2/trade/create")
	if err != nil {
		log.Printf("Web端购买API请求失败: %v", err)
		return ""
	}

	log.Printf("Web端购买API响应: Code=%d, Message=%s", creates.Code, creates.Message)
	log.Printf("Web端响应内容: %s", r.String())

	if creates.Code == 0 && creates.Data.OrderId != "" {
		log.Printf("Web端成功创建订单: %s, 状态: %s", creates.Data.OrderId, creates.Data.State)
		return creates.Data.OrderId
	} else if creates.Code == 0 {
		log.Printf("Web端返回成功但订单ID为空，可能是余额不足")
		if creates.Data.BpEnough == -1 {
			log.Printf("确认余额不足，需要充值B币")
		}
	} else {
		log.Printf("Web端购买失败: Code=%d, Message=%s", creates.Code, creates.Message)
	}

	return ""
}

func tradeQuery() {
Loop:
	for {
		task := time.NewTimer(500 * time.Millisecond)

		// 先尝试APP端查询
		params := map[string]string{
			"order_id": orderId,
			"mobi_app": "android",
			"build":    "7490300",
		}

		// 使用APP签名
		signedParams := signParams(params)

		query := &Query{}
		r, err := client.R().
			SetQueryParams(signedParams).
			SetResult(query).
			Get("/garb/trade/query")
		checkErr(err)

		log.Printf("APP端订单查询API响应状态: %d", r.StatusCode())
		log.Printf("APP端订单查询API响应: Code=%d, Message=%s", query.Code, query.Message)

		// 如果APP端失败，尝试Web端
		if query.Code != 0 || r.StatusCode() != 200 {
			log.Printf("APP端查询失败，尝试Web端查询")
			webQuery := queryWebOrder(orderId)
			if webQuery != nil {
				query = webQuery
				log.Printf("Web端订单查询成功: Code=%d", query.Code)
			}
		}

		if query.Code == 0 {
			log.Printf("订单状态: %s, 订单ID: %s", query.Data.State, query.Data.OrderId)
			switch query.Data.State {
			case "paid":
				log.Println("已成功支付.")
				break Loop
			case "paying":
				log.Println("支付中，请稍候...")
			case "created":
				log.Println("订单已创建，等待支付...")
			case "cancelled":
				log.Println("订单已取消")
				break Loop
			case "failed":
				log.Println("订单支付失败")
				break Loop
			default:
				errorTime += 1
				log.Printf("未知订单状态: %s", query.Data.State)
				log.Println(r)
				if errorTime >= 5 {
					log.Fatalln("失败次数已达到五次，退出执行...")
				}
			}
		} else {
			errorTime += 1
			log.Printf("订单查询API返回错误: Code=%d, Message=%s", query.Code, query.Message)
			log.Println("响应内容:", r.String())
			if errorTime >= 5 {
				log.Fatalln("失败次数已达到五次，退出执行...")
			}
		}
		<-task.C
	}
}

// Web端订单查询API函数（根据bilibili-API-collect标准）
func queryWebOrder(orderID string) *Query {
	params := map[string]string{
		"order_id": orderID,
		"csrf":     config.Cookies.BiliJct,
	}

	query := &Query{}
	r, err := webClient.R().
		SetQueryParams(params).
		SetResult(query).
		Get("/garb/trade/query")
	if err != nil {
		log.Printf("Web端订单查询API请求失败: %v", err)
		return nil
	}

	log.Printf("Web端订单查询API响应状态: %d", r.StatusCode())
	log.Printf("Web端订单查询响应: Code=%d, Message=%s", query.Code, query.Message)

	if query.Code == 0 {
		return query
	}

	return nil
}

func wallet() {
	// 使用导航API中的钱包信息，因为我们已经在nav()中获取了余额
	log.Printf("购买完成！当前B币余额: %.2f.", bp)
}

func suitAsset() {
	// 使用Web端API查询装扮资产
	params := map[string]string{
		"csrf":    config.Cookies.BiliJct,
		"item_id": config.ItemId,
		"part":    "suit",
		"trial":   "0",
	}

	response := &SuitAsset{}
	r, err := webClient.R().
		SetQueryParams(params).
		SetResult(response).
		Get("/garb/user/suit/asset")
	checkErr(err)

	log.Printf("装扮资产API响应状态: %d", r.StatusCode())
	if response.Data.Fan.Number == 0 {
		log.Printf("未获取到装扮编号信息")
		return
	}

	log.Printf("名称: %v 编号: %v.", itemName, response.Data.Fan.Number)
}

func now() {
	result := &Now{}
	clock := resty.New()
	for {
		r, err := clock.R().
			SetResult(result).
			EnableTrace().
			SetHeader("user-agent", ua).
			Get("http://api.bilibili.com/x/report/click/now")
		checkErr(err)
		if result.Data.Now >= startTime-28 {
			waitTime = r.Request.TraceInfo().ServerTime.Milliseconds()
			break
		}
	}
}

func sign(params map[string]string) string {
	var query string
	var buffer bytes.Buffer

	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		buffer.WriteString(k)
		buffer.WriteString("=")
		buffer.WriteString(params[k])
		buffer.WriteString("&")
	}
	query = strings.TrimRight(buffer.String(), "&")

	s := strMd5(fmt.Sprintf("%v%v", query, appSecret))
	return s
}

// APP签名函数
func signParams(params map[string]string) map[string]string {
	// 添加必要的APP参数
	params["appkey"] = appKey
	params["ts"] = strconv.FormatInt(time.Now().Unix(), 10)

	// 生成签名
	sign := sign(params)
	params["sign"] = sign

	return params
}

// 计算 MD5
func strMd5(str string) (retMd5 string) {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

func outPutRank() {
	if rankInfo == nil || rankInfo.Data.Rank == nil {
		log.Println("当前列表为空，可能有依号出现！！！")
		return
	}
	log.Println("当前装扮列表:")
	fmt.Println("")
	for _, x := range rankInfo.Data.Rank {
		fmt.Printf("\t编号: %v\t拥有者: %v\n", x.Number, x.Nickname)
	}
	fmt.Println("")
}

func waitToStart() {
	log.Println("正在等待开售...")
	for {
		task := time.NewTimer(1 * time.Millisecond)
		t := time.Now().Unix()
		fmt.Printf("倒计时: %v.\r", formatSecond(startTime-t))
		if t >= startTime-30 {
			log.Println("准备冻手！！！")
			task.Reset(0)
			break
		}
		<-task.C
	}
}

func formatSecond(seconds int64) string {
	var d, h, m, s int64
	var msg string

	if seconds > SecondsPerDay {
		d = seconds / SecondsPerDay
		h = seconds % SecondsPerDay / SecondsPerHour
		m = seconds % SecondsPerDay % SecondsPerHour / SecondsPerMinute
		s = seconds % 60
		msg = fmt.Sprintf("%v天%v小时%v分%v秒", d, h, m, s)
	} else if seconds > SecondsPerHour {
		h = seconds / SecondsPerHour
		m = seconds % SecondsPerHour / SecondsPerMinute
		s = seconds % 60
		msg = fmt.Sprintf("%v小时%v分%v秒", h, m, s)
	} else if seconds > SecondsPerMinute {
		m = seconds / SecondsPerMinute
		s = seconds % 60
		msg = fmt.Sprintf("%v分%v秒", m, s)
	} else {
		s = seconds
		msg = fmt.Sprintf("%v秒", s)
	}
	return msg
}

// 抢购特定序号装扮（监控模式）
func watchTargetId(itemId int, targetId int64, limitV int64, buyNum *int64) bool {
	// 根据bilibili-API-collect标准，使用Web端调用
	params := map[string]string{
		"csrf":    config.Cookies.BiliJct,
		"item_id": config.ItemId,
	}

	ranks := &Rank{}
	_, err := webClient.R().
		SetQueryParams(params).
		SetResult(ranks).
		Get("/garb/rank/fan/recent")
	if err != nil {
		log.Printf("获取排行榜失败: %v", err)
		return false
	}

	if ranks.Code != 0 {
		log.Printf("获取排行榜错误: %s", ranks.Message)
		return false
	}

	// 检查目标编号是否已被购买
	for _, rank := range ranks.Data.Rank {
		if int64(rank.Number) == targetId {
			log.Printf("目标编号 %d 已被 %s 购买", targetId, rank.Nickname)
			return false
		}
		if int64(rank.Number) > limitV {
			log.Printf("当前最大编号 %d 已超过限制 %d，开始抢购", rank.Number, limitV)
			return true
		}
	}

	return false
}

// 定时抢购模式
func timedPurchase(itemId int, targetTime time.Time, buyNum int64, config *Config) error {
	log.Printf("定时抢购模式，目标时间: %s", targetTime.Format("2006-01-02 15:04:05"))

	// 等待到目标时间前10秒
	for {
		now := time.Now()
		diff := targetTime.Sub(now)

		if diff <= 10*time.Second {
			log.Printf("进入最后10秒倒计时...")
			break
		}

		log.Printf("距离目标时间还有: %s", diff.String())
		time.Sleep(1 * time.Second)
	}

	// 精确倒计时
	for {
		now := time.Now()
		diff := targetTime.Sub(now)

		if diff <= 0 {
			log.Printf("开始定时抢购！")
			break
		}

		log.Printf("倒计时: %s", diff.String())

		if diff <= 100*time.Millisecond {
			time.Sleep(10 * time.Millisecond)
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}

	// 执行抢购
	create()
	return nil
}

// 获取用户选择
func getUserChoice() int {
	var choice int
	fmt.Println("=== B站装扮抢购脚本 ===")
	fmt.Println("请选择抢购模式：")
	fmt.Println("1. 抢购特定序号装扮（监控当前ID，抢购指定粉丝编号）")
	fmt.Println("2. 定时抢购装扮（在指定时间点抢购）")
	fmt.Print("请输入选择 (1 或 2): ")

	for {
		_, err := fmt.Scanf("%d", &choice)
		if err != nil || (choice != 1 && choice != 2) {
			fmt.Print("输入无效，请输入 1 或 2: ")
			continue
		}
		break
	}

	return choice
}

// 获取定时抢购时间
func getTimedPurchaseTime() time.Time {
	var timeStr string
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("\n=== 定时抢购模式 ===")
	fmt.Print("请输入抢购时间（格式：2006-01-02 15:04:05）:\n时间: ")

	for {
		if scanner.Scan() {
			timeStr = strings.TrimSpace(scanner.Text())
		} else {
			fmt.Print("输入错误，请重新输入: ")
			continue
		}

		// 尝试解析时间
		targetTime, err := time.ParseInLocation("2006-01-02 15:04:05", timeStr, time.Local)
		if err != nil {
			fmt.Printf("时间格式错误，请使用格式 2006-01-02 15:04:05: ")
			continue
		}

		// 检查时间是否在未来
		if targetTime.Before(time.Now()) {
			fmt.Print("时间不能是过去的时间，请重新输入: ")
			continue
		}

		fmt.Printf("目标时间: %s (%s)\n", targetTime.Format("2006-01-02 15:04:05"), targetTime.Location())
		fmt.Printf("当前时间: %s (%s)\n", time.Now().Format("2006-01-02 15:04:05"), time.Now().Location())

		return targetTime
	}
}

func init() {
	flag.StringVar(&fileName, "c", "./config.json", "Path to config file.")
	flag.Parse()

	// 读取配置文件
	jsonFile, err := os.ReadFile(fileName)
	checkErr(err)
	err = json.Unmarshal(jsonFile, config)
	checkErr(err)

	// 登录
	if config.Cookies.SESSDATA == "" {
		login.SetHeader("user-agent", ua)
		login.SetBaseURL("https://passport.bilibili.com")
		log.Println("检测到空的SESSDATA，开始扫码登录流程...")
		webLogin()
	}

	cookies = []*http.Cookie{
		{Name: "SESSDATA", Value: config.Cookies.SESSDATA},
		{Name: "bili_jct", Value: config.Cookies.BiliJct},
		{Name: "DedeUserID", Value: config.Cookies.DedeUserID},
		{Name: "DedeUserID__ckMd5", Value: config.Cookies.DedeUserIDCkMd5},
	}

	headers := map[string]string{
		"User-Agent":      ua,
		"Content-Type":    "application/x-www-form-urlencoded; charset=utf-8",
		"Accept":          "application/json",
		"Accept-Language": "zh-CN,zh;q=0.9",
		"Accept-Encoding": "gzip, deflate",
		"Connection":      "keep-alive",
		"Cache-Control":   "no-cache",
		"Pragma":          "no-cache",
	}

	client.SetHeaders(headers)
	client.SetBaseURL("https://app.bilibili.com")
	client.SetCookies(cookies)
	client.SetTimeout(10 * time.Second)
	client.SetRetryCount(2)

	// 配置Web端客户端（用于导航等Web端API）
	webHeaders := map[string]string{
		"User-Agent":      ua,
		"Accept":          "application/json, text/plain, */*",
		"Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
		"Cache-Control":   "no-cache",
		"Pragma":          "no-cache",
		"Referer":         "https://www.bilibili.com",
		"Origin":          "https://www.bilibili.com",
	}

	webClient.SetHeaders(webHeaders)
	webClient.SetBaseURL("https://api.bilibili.com/x")
	webClient.SetCookies(cookies)
	webClient.SetTimeout(10 * time.Second)
	webClient.SetRetryCount(2)
}

func main() {
	// 初始化log
	f, err := os.OpenFile("log.txt", os.O_CREATE|os.O_APPEND|os.O_RDWR, os.ModePerm)
	checkErr(err)
	defer func(f *os.File) {
		err = f.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(f)

	multiWriter := io.MultiWriter(os.Stdout, f)
	log.SetOutput(multiWriter)
	log.SetFlags(log.Ldate | log.Lmicroseconds)

	log.Println("配置文件加载成功")

	// 获取用户选择的模式
	choice := getUserChoice()

	// 登陆验证
	nav()

	// 获取装扮信息
	detail()

	// 预热请求（可选，失败不影响核心功能）
	log.Printf("开始预热API...")

	// 安全调用预热API
	func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("预热API调用异常（不影响核心功能）: %v", r)
			}
		}()
		asset()
		state()
		rank()
		stat()
		coupon()
	}()

	log.Printf("预热API完成")

	// 输出编号列表
	outPutRank()

	switch choice {
	case 1:
		// 监控特定序号模式
		fmt.Println("\n=== 监控特定序号模式 ===")
		if config.TargetMode.TargetID == 0 {
			log.Fatalln("配置文件中缺少 target_mode.target_id 参数")
		}

		log.Printf("开始监控装扮 %s (ID: %s)", itemName, config.ItemId)
		log.Printf("目标编号: %d, 限制编号: %d, 购买数量: %d",
			config.TargetMode.TargetID, config.TargetMode.LimitV, config.TargetMode.BuyNum)

		for {
			shouldBuy := watchTargetId(
				int(config.TargetMode.TargetID),
				config.TargetMode.TargetID,
				config.TargetMode.LimitV,
				&config.TargetMode.BuyNum,
			)

			if shouldBuy {
				log.Printf("触发抢购条件，开始抢购...")
				create()
				break
			}

			time.Sleep(1 * time.Second)
		}

	case 2:
		// 定时抢购模式
		targetTime := getTimedPurchaseTime()

		if config.TimedMode.BuyNum == 0 {
			config.TimedMode.BuyNum = 1
		}

		err := timedPurchase(
			int(config.TargetMode.TargetID),
			targetTime,
			config.TimedMode.BuyNum,
			config,
		)
		if err != nil {
			log.Fatalf("定时抢购失败:%v", err)
		}
	}

	// 如果订单ID不为空，则追踪订单
	if orderId != "" {
		log.Printf("开始追踪订单: %s", orderId)
		tradeQuery()
	} else {
		log.Printf("订单ID为空，跳过订单查询（可能是余额不足或其他限制）")
	}

	// 查询余额和装扮信息
	nav()
	wallet()
	suitAsset()

	log.Println("抢购流程完成！")
}

func checkErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
