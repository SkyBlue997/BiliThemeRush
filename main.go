package main

import (
	"bufio"
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/buger/jsonparser"
)

// 常量定义
const (
	// API相关常量
	AppKey     = "1d8b6e7d45233436"
	AppSecret  = "560c52ccd288fed045859ed18bffd973"
	AppVersion = "7490300"
	SDKVersion = "1.4.9"

	// 请求超时时间
	RequestTimeout = 10 * time.Second

	// 重试相关
	MaxRetries = 3
	RetryDelay = 1 * time.Second
)

// 全局HTTP客户端
var httpClient = &http.Client{
	Timeout: RequestTimeout,
}

// 配置文件结构体
type Config struct {
	API struct {
		CSRF      string `json:"csrf"`
		Cookie    string `json:"cookie"`
		AccessKey string `json:"access_key"`
		UserAgent string `json:"user_agent"`
	} `json:"api"`
	Pay struct {
		UserAgent         string `json:"user_agent"`
		Buvid             string `json:"buvid"`
		DeviceID          string `json:"device_id"`
		FpLocal           string `json:"fp_local"`
		FpRemote          string `json:"fp_remote"`
		SessionID         string `json:"session_id"`
		DeviceFingerprint string `json:"device_fingerprint"`
	} `json:"pay"`
	Item struct {
		ID       int `json:"id"`
		AddMonth int `json:"add_month"`
	} `json:"item"`
	TargetMode struct {
		TargetID int64 `json:"target_id"`
		LimitV   int64 `json:"limit_v"`
		BuyNum   int64 `json:"buy_num"`
	} `json:"target_mode"`
	TimedMode struct {
		BuyNum int64 `json:"buy_num"`
	} `json:"timed_mode"`
}

// 加载配置文件
func loadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename) // 使用新的API替代ioutil.ReadFile
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	return &config, nil
}

// 验证必要参数
func validateConfig(config *Config) error {
	if config.API.CSRF == "" {
		return fmt.Errorf("配置文件中缺少 api.csrf 参数")
	}
	if config.API.Cookie == "" {
		return fmt.Errorf("配置文件中缺少 api.cookie 参数")
	}
	if config.API.AccessKey == "" {
		return fmt.Errorf("配置文件中缺少 api.access_key 参数")
	}
	if config.Item.ID == 0 {
		return fmt.Errorf("配置文件中缺少 item.id 参数")
	}
	return nil
}

func init() {
	//proxyUrl, err := url.Parse("http://127.0.0.1:10801")
	//if err != nil {
	//	log.Fatal(err)
	//	return
	//}
	//http.DefaultTransport = &http.Transport{Proxy: http.ProxyURL(proxyUrl)}
}

// APP签名函数
func signParams(params map[string]string, appkey, appsec string) string {
	// 添加appkey
	params["appkey"] = appkey

	// 按key排序
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 构建查询字符串
	queryPairs := make([]string, 0, len(keys))
	for _, k := range keys {
		v := params[k]
		queryPairs = append(queryPairs, fmt.Sprintf("%s=%s", k, url.QueryEscape(v)))
	}
	query := strings.Join(queryPairs, "&")

	// 计算MD5签名
	signStr := query + appsec
	hash := md5.Sum([]byte(signStr))
	return hex.EncodeToString(hash[:])
}

// 设置通用请求头
func setCommonHeaders(req *http.Request, userAgent string) {
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
}

// 设置支付请求头
func setPaymentHeaders(req *http.Request, config *Config) {
	setCommonHeaders(req, config.Pay.UserAgent)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("cLocale", "zh_CN")
	req.Header.Set("sLocale", "zh_CN")
	req.Header.Set("Buvid", config.Pay.Buvid)
	req.Header.Set("Device-ID", config.Pay.DeviceID)
	req.Header.Set("fp_local", config.Pay.FpLocal)
	req.Header.Set("fp_remote", config.Pay.FpRemote)
	req.Header.Set("session_id", config.Pay.SessionID)
	req.Header.Set("deviceFingerprint", config.Pay.DeviceFingerprint)
	req.Header.Set("buildId", AppVersion)
	req.Header.Set("env", "prod")
	req.Header.Set("APP-KEY", "android64")
	req.Header.Set("bili-bridge-engine", "cronet")
}

// 执行HTTP请求并返回响应体
func doRequest(req *http.Request) ([]byte, error) {
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	var reader io.Reader = resp.Body

	// 检查是否是gzip压缩的响应
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("创建gzip读取器失败: %w", err)
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	return body, nil
}

// 更新支付数据
func updatePayData(payData, accessKey string) (string, error) {
	// 使用map来批量更新，减少重复的jsonparser.Set调用
	updates := map[string]string{
		"accessKey":    accessKey,
		"appName":      "tv.danmaku.bili",
		"appVersion":   AppVersion,
		"device":       "ANDROID",
		"deviceType":   "3",
		"network":      "WiFi",
		"payChannel":   "bp",
		"payChannelId": "99",
		"realChannel":  "bp",
		"sdkVersion":   SDKVersion,
	}

	result := []byte(payData)
	for key, value := range updates {
		var err error
		if key == "payChannelId" {
			result, err = jsonparser.Set(result, []byte(value), key)
		} else {
			result, err = jsonparser.Set(result, []byte(`"`+value+`"`), key)
		}
		if err != nil {
			return "", fmt.Errorf("更新支付数据字段 %s 失败: %w", key, err)
		}
	}

	return string(result), nil
}

func orderCreate(itemId int, addMonth int, buyNum int64, csrf string, apiCookie string, userAgent string, accessKey string) (string, error) {
	// 使用最新的API端点和参数
	params := map[string]string{
		"item_id":      strconv.Itoa(itemId),
		"platform":     "android",
		"currency":     "bp",
		"add_month":    strconv.Itoa(addMonth),
		"buy_num":      strconv.FormatInt(buyNum, 10),
		"coupon_token": "",
		"hasBiliapp":   "true",
		"csrf":         csrf,
		"ts":           strconv.FormatInt(time.Now().Unix(), 10),
	}

	sign := signParams(params, AppKey, AppSecret)
	params["sign"] = sign

	// 构建请求体
	formData := make([]string, 0, len(params))
	for k, v := range params {
		formData = append(formData, fmt.Sprintf("%s=%s", k, url.QueryEscape(v)))
	}

	req, err := http.NewRequest("POST", "https://api.bilibili.com/x/garb/trade/create", strings.NewReader(strings.Join(formData, "&")))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置必要的headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	setCommonHeaders(req, userAgent)
	req.Header.Set("Cookie", apiCookie)
	req.Header.Set("Referer", fmt.Sprintf("https://www.bilibili.com/h5/mall/suit/detail?id=%d&navhide=1", itemId))
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	respb, err := doRequest(req)
	if err != nil {
		return "", fmt.Errorf("下单请求失败: %w", err)
	}

	log.Println("下单响应:", string(respb))

	// 检查响应状态
	code, err := jsonparser.GetInt(respb, "code")
	if err != nil || code != 0 {
		msg, _ := jsonparser.GetString(respb, "message")
		return "", fmt.Errorf("下单失败，错误码: %d, 消息: %s", code, msg)
	}

	payData0, err := jsonparser.GetString(respb, "data", "pay_data")
	if err != nil {
		return "", fmt.Errorf("获取支付数据失败: %w", err)
	}

	// 更新支付数据中的参数
	payData, err := updatePayData(payData0, accessKey)
	if err != nil {
		return "", fmt.Errorf("更新支付数据失败: %w", err)
	}

	return payData, nil
}

func pay(payData string, config *Config) (string, error) {
	req, err := http.NewRequest("POST", "https://pay.bilibili.com/payplatform/pay/pay", strings.NewReader(payData))
	if err != nil {
		return "", fmt.Errorf("创建支付请求失败: %w", err)
	}

	setPaymentHeaders(req, config)

	respb, err := doRequest(req)
	if err != nil {
		return "", fmt.Errorf("支付请求失败: %w", err)
	}

	log.Println("支付平台响应:", string(respb))

	// 检查响应状态
	code, err := jsonparser.GetInt(respb, "code")
	if err != nil || code != 0 {
		msg, _ := jsonparser.GetString(respb, "message")
		return "", fmt.Errorf("支付平台调用失败，错误码: %d, 消息: %s", code, msg)
	}

	payChannelParam, err := jsonparser.GetString(respb, "data", "payChannelParam")
	if err != nil {
		return "", fmt.Errorf("获取支付通道参数失败: %w", err)
	}
	return payChannelParam, nil
}

func payBp(payChannelParam string, config *Config) (string, error) {
	req, err := http.NewRequest("POST", "https://pay.bilibili.com/paywallet/pay/payBp", strings.NewReader(payChannelParam))
	if err != nil {
		return "", fmt.Errorf("创建BP支付请求失败: %w", err)
	}

	setPaymentHeaders(req, config)

	respb, err := doRequest(req)
	if err != nil {
		return "", fmt.Errorf("BP支付请求失败: %w", err)
	}

	log.Println("BP支付响应:", string(respb))

	// 检查响应状态
	code, err := jsonparser.GetInt(respb, "code")
	if err != nil || code != 0 {
		msg, _ := jsonparser.GetString(respb, "message")
		return "", fmt.Errorf("BP支付失败，错误码: %d, 消息: %s", code, msg)
	}

	return string(respb), nil
}

func buy(payData string, config *Config) (string, error) {
	payChannelParam, err := pay(payData, config)
	if err != nil {
		return "", err
	}

	payResult, err := payBp(payChannelParam, config)
	if err != nil {
		return "", err
	}
	return payResult, nil
}

// 执行带重试的HTTP请求
func doRequestWithRetry(req *http.Request) ([]byte, error) {
	var lastErr error

	for i := 0; i < MaxRetries; i++ {
		// 克隆请求以避免重复使用问题
		reqClone := req.Clone(req.Context())

		respb, err := doRequest(reqClone)
		if err == nil {
			return respb, nil
		}

		lastErr = err
		if i < MaxRetries-1 {
			log.Printf("请求失败 (重试 %d/%d): %v", i+1, MaxRetries, err)
			time.Sleep(RetryDelay * time.Duration(i+1))
		}
	}

	return nil, fmt.Errorf("多次重试后仍然失败: %w", lastErr)
}

// 抢购特定序号的装扮（原根目录版本的功能）
func watchTargetId(itemId int, targetId int64, limitV int64, buyNum *int64) bool {
	log.Println("开始观望特定序号...")

	// 使用更加稳定的用户代理
	userAgent := "Mozilla/5.0 (Linux; Android 12; SM-G975F) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36 BiliApp/7.49.0"

	for {
		// 构建API请求参数
		params := map[string]string{
			"item_id": strconv.Itoa(itemId),
			"part":    "suit",
			"ts":      strconv.FormatInt(time.Now().Unix(), 10),
		}

		// 添加签名
		sign := signParams(params, AppKey, AppSecret)

		// 构建URL
		apiURL := fmt.Sprintf("https://api.bilibili.com/x/garb/mall/item/suit/v2?item_id=%d&part=suit&ts=%s&appkey=%s&sign=%s",
			itemId, params["ts"], AppKey, sign)

		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			log.Printf("创建请求失败: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}

		setCommonHeaders(req, userAgent)
		req.Header.Set("Referer", "https://www.bilibili.com/")
		req.Header.Set("Origin", "https://www.bilibili.com")

		respb, err := doRequestWithRetry(req)
		if err != nil {
			log.Printf("请求失败: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}

		// 打印响应内容用于调试（仅前200字符）
		responseStr := string(respb)
		if len(responseStr) > 200 {
			log.Printf("API响应前200字符: %s...", responseStr[:200])
		} else {
			log.Printf("API响应: %s", responseStr)
		}

		SuitRecentResultCode, err := jsonparser.GetInt(respb, "code")
		if err != nil {
			log.Printf("解析响应码失败: %v", err)
			log.Printf("尝试使用备用API...")

			// 尝试使用备用API
			backupURL := fmt.Sprintf("https://api.bilibili.com/x/garb/mall/item/detail?item_id=%d", itemId)
			backupReq, err := http.NewRequest("GET", backupURL, nil)
			if err != nil {
				log.Printf("创建备用请求失败: %v", err)
				time.Sleep(2 * time.Second)
				continue
			}

			setCommonHeaders(backupReq, userAgent)
			backupReq.Header.Set("Referer", "https://www.bilibili.com/")

			respb, err = doRequestWithRetry(backupReq)
			if err != nil {
				log.Printf("备用请求失败: %v", err)
				time.Sleep(2 * time.Second)
				continue
			}

			log.Printf("备用API响应: %s", string(respb))

			SuitRecentResultCode, err = jsonparser.GetInt(respb, "code")
			if err != nil {
				log.Printf("备用API也解析失败: %v", err)
				time.Sleep(2 * time.Second)
				continue
			}
		}

		if SuitRecentResultCode != 0 {
			msg, _ := jsonparser.GetString(respb, "message")
			log.Printf("API返回错误，错误码: %d, 消息: %s", SuitRecentResultCode, msg)
			time.Sleep(2 * time.Second)
			continue
		}

		// 尝试多种路径获取销售数量
		var saleQuantity string
		var saleQuantityI int64
		var saleSurplus int64

		// 路径1: data.item.properties.sale_quantity
		saleQuantity, err = jsonparser.GetString(respb, "data", "item", "properties", "sale_quantity")
		if err != nil {
			// 路径2: data.properties.sale_quantity
			saleQuantity, err = jsonparser.GetString(respb, "data", "properties", "sale_quantity")
			if err != nil {
				// 路径3: data.sale_quantity
				saleQuantity, err = jsonparser.GetString(respb, "data", "sale_quantity")
				if err != nil {
					log.Printf("获取销售数量失败，尝试所有路径都失败: %v", err)
					time.Sleep(2 * time.Second)
					continue
				}
			}
		}

		saleQuantityI, err = strconv.ParseInt(saleQuantity, 10, 64)
		if err != nil {
			log.Printf("解析销售数量失败: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}

		// 尝试多种路径获取剩余数量
		saleSurplus, err = jsonparser.GetInt(respb, "data", "sale_surplus")
		if err != nil {
			// 备用路径
			saleSurplus, err = jsonparser.GetInt(respb, "data", "item", "sale_surplus")
			if err != nil {
				log.Printf("获取剩余数量失败: %v", err)
				time.Sleep(2 * time.Second)
				continue
			}
		}

		nowId := saleQuantityI - saleSurplus
		log.Printf("当前ID: %d, 目标ID: %d, 剩余: %d", nowId, targetId, saleSurplus)

		if nowId < targetId && nowId+*buyNum >= targetId {
			log.Println("开始抢购特定序号")
			*buyNum = targetId - nowId
			return true
		} else if nowId >= targetId {
			log.Println("已经错过")
			return false
		}

		if nowId < targetId-limitV {
			time.Sleep(2 * time.Second)
		} else {
			// 接近目标时，缩短等待时间
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// 定时抢购装扮（time版本的功能）
func timedPurchase(itemId int, targetTime time.Time, buyNum int64, config *Config) error {
	log.Printf("定时抢购模式，目标时间: %s", targetTime.Format("2006-01-02 15:04:05"))

	// 等待到目标时间前10秒
	waitTime := targetTime.Add(-10 * time.Second)
	now := time.Now()

	if now.Before(waitTime) {
		sleepDuration := waitTime.Sub(now)
		log.Printf("等待到目标时间前10秒，等待时长: %v", sleepDuration)
		time.Sleep(sleepDuration)
	}

	log.Println("进入最后10秒倒计时...")

	// 最后10秒，每秒检查一次时间
	for {
		now := time.Now()
		if now.After(targetTime) {
			log.Println("开始定时抢购！")
			break
		}

		remaining := targetTime.Sub(now)
		log.Printf("倒计时: %v", remaining)
		time.Sleep(1 * time.Second)
	}

	// 开始抢购
	addMonth := config.Item.AddMonth
	payData, err := orderCreate(itemId, addMonth, buyNum, config.API.CSRF, config.API.Cookie, config.API.UserAgent, config.API.AccessKey)
	if err != nil {
		return fmt.Errorf("定时抢购下单失败: %w", err)
	}

	log.Println("定时抢购下单成功，开始支付...")
	payResult, err := buy(payData, config)
	if err != nil {
		return fmt.Errorf("定时抢购支付失败: %w", err)
	}
	log.Println("定时抢购支付完成:", payResult)
	return nil
}

func getUserChoice() int {
	fmt.Println("=== B站装扮抢购脚本 ===")
	fmt.Println("请选择抢购模式：")
	fmt.Println("1. 抢购特定序号装扮（监控当前ID，抢购指定粉丝编号）")
	fmt.Println("2. 定时抢购装扮（在指定时间点抢购）")
	fmt.Print("请输入选择 (1 或 2): ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal("读取输入失败:", err)
	}

	input = strings.TrimSpace(input)
	choice, err := strconv.Atoi(input)
	if err != nil || (choice != 1 && choice != 2) {
		fmt.Println("无效选择，请输入 1 或 2")
		return getUserChoice()
	}

	return choice
}

func getTimedPurchaseTime() time.Time {
	fmt.Println("请输入抢购时间（格式：2006-01-02 15:04:05）:")
	fmt.Print("时间: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal("读取输入失败:", err)
	}

	input = strings.TrimSpace(input)
	targetTime, err := time.Parse("2006-01-02 15:04:05", input)
	if err != nil {
		fmt.Printf("时间格式错误: %v\n", err)
		fmt.Println("请使用格式：2006-01-02 15:04:05 （例如：2024-01-01 12:00:00）")
		return getTimedPurchaseTime()
	}

	if targetTime.Before(time.Now()) {
		fmt.Println("目标时间不能早于当前时间")
		return getTimedPurchaseTime()
	}

	return targetTime
}

func main() {
	// 加载配置文件
	config, err := loadConfig("config.json")
	if err != nil {
		log.Fatalf("加载配置文件失败: %v\n请确保 config.json 文件存在并格式正确", err)
	}

	// 验证必要参数
	if err := validateConfig(config); err != nil {
		log.Fatalf("配置验证失败: %v\n请检查 config.json 文件中的参数配置", err)
	}

	log.Println("配置文件加载成功")

	// 获取用户选择
	choice := getUserChoice()

	switch choice {
	case 1:
		// 抢购特定序号模式
		fmt.Println("\n=== 抢购特定序号模式 ===")

		limitV := config.TargetMode.LimitV
		targetId := config.TargetMode.TargetID
		buyNum := config.TargetMode.BuyNum
		addMonth := config.Item.AddMonth

		log.Printf("开始监控装扮 ID: %d, 目标粉丝编号: %d", config.Item.ID, targetId)

		if watchTargetId(config.Item.ID, targetId, limitV, &buyNum) {
			log.Printf("触发抢购条件，准备购买 %d 个", buyNum)

			payData, err := orderCreate(config.Item.ID, addMonth, buyNum, config.API.CSRF, config.API.Cookie, config.API.UserAgent, config.API.AccessKey)
			if err != nil {
				log.Fatal("下单失败:", err)
			}

			log.Println("下单成功，开始支付...")
			payResult, err := buy(payData, config)
			if err != nil {
				log.Fatal("支付失败:", err)
			}
			log.Println("支付完成:", payResult)
		}

	case 2:
		// 定时抢购模式
		fmt.Println("\n=== 定时抢购模式 ===")

		targetTime := getTimedPurchaseTime()
		buyNum := config.TimedMode.BuyNum

		if err := timedPurchase(config.Item.ID, targetTime, buyNum, config); err != nil {
			log.Fatal("定时抢购失败:", err)
		}
	}

	log.Println("程序结束.")
}
