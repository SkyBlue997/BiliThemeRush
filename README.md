# 🎯 BiliThemeRush

> 高效的B站装扮自动抢购工具，支持双模式智能抢购策略

[![Go Version](https://img.shields.io/badge/Go-1.17+-00ADD8?style=flat-square&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue?style=flat-square)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Cross--Platform-lightgrey?style=flat-square)](https://github.com/)

---

## 🚀 快速开始

### 系统要求
- Go 1.17 或更高版本
- 稳定的网络连接

### 安装步骤

```bash
# 克隆项目
git clone https://github.com/SkyBlue997/BiliThemeRush.git
cd BiliThemeRush

# 编译程序（跨平台编译）
CGO_ENABLED=0 go build -o bilibili_rush main.go

# Windows用户编译
CGO_ENABLED=0 GOOS=windows go build -o bilibili_rush.exe main.go

# macOS用户编译  
CGO_ENABLED=0 GOOS=darwin go build -o bilibili_rush main.go

# Linux用户编译
CGO_ENABLED=0 GOOS=linux go build -o bilibili_rush main.go
```

### 初次配置

```bash
# 复制配置模板
cp config.example.json config.json

# 编辑配置文件（填入必要参数）
vim config.json   # 或使用其他编辑器
```

### 运行程序

```bash
# 运行程序
./bilibili_rush

# Windows用户
bilibili_rush.exe
```

---

## 🎮 抢购模式

### 🎯 模式一：特定序号抢购
**适用场景：** 抢购特定粉丝编号（如：豹子号111、999，炸弹号6666等）

**工作原理：**
- 持续监控当前装扮的销售ID进度
- 当接近目标序号时自动触发抢购
- 适合对序号有特殊要求的收藏者

**优势：**
- 精确控制抢购时机
- 可设置观望临界值避免过早抢购
- 支持批量购买增加成功率

### ⏰ 模式二：定时抢购
**适用场景：** 在装扮开售的准确时间点进行抢购

**工作原理：**
- 等待到用户指定的时间点
- 在精确的时间发起抢购请求
- 适合新装扮首发和限时活动

**优势：**
- 时间控制精确到秒
- 适合热门装扮的首发抢购
- 减少不必要的等待时间

---

## ⚙️ 配置文件详解

### 配置文件结构 (`config.json`)

```json
{
  "bp_enough": true,
  "buy_num": "1", 
  "coupon_token": "",
  "device": "android",
  "item_id": "装扮ID",
  "time_before": 30,
  "cookies": {
    "SESSDATA": "你的SESSDATA",
    "bili_jct": "你的bili_jct", 
    "DedeUserID": "你的DedeUserID",
    "DedeUserID__ckMd5": "你的DedeUserID__ckMd5"
  },
  "target_mode": {
    "target_id": 999,
    "limit_v": 1000,
    "buy_num": 1
  },
  "timed_mode": {
    "buy_num": 1
  }
}
```

### 基础配置参数

| 参数 | 说明 | 可选值 | 示例 |
|------|------|--------|------|
| `bp_enough` | 是否检查B币余额 | `true`/`false` | `true` |
| `buy_num` | 默认购买数量 | 字符串数字 | `"1"` |
| `coupon_token` | 优惠券令牌 | 字符串 | `""` |
| `device` | 设备类型 | `"android"`/`"ios"` | `"android"` |
| `item_id` | 装扮ID | 数字字符串 | `"33998"` |
| `time_before` | 提前开始时间(秒) | 数字 | `30` |

### Cookies配置 (必需)

如果配置文件中cookies为空，程序会自动启动扫码登录流程。

| 参数 | 说明 | 获取方法 |
|------|------|----------|
| `SESSDATA` | 会话数据 | 浏览器F12 → 登录B站 → Cookie中获取 |
| `bili_jct` | CSRF令牌 | 同上 |
| `DedeUserID` | 用户ID | 同上 |
| `DedeUserID__ckMd5` | 用户ID的MD5 | 同上 |

### 特定序号模式配置

| 参数 | 说明 | 推荐值 |
|------|------|--------|
| `target_id` | 目标粉丝编号 | 根据需要设定 |
| `limit_v` | 观望临界值 | 冷门:10 / 热门:70 / 超热:140 |
| `buy_num` | 购买数量 | 普通:1 / 热门:3-5 |

### 定时模式配置

| 参数 | 说明 | 推荐值 |
|------|------|--------|
| `buy_num` | 购买数量 | 通常设为1 |

---

## 🔧 Cookie获取指南

### 方法一：浏览器开发者工具

1. **打开浏览器** → 访问 [bilibili.com](https://www.bilibili.com)
2. **按F12** → 打开开发者工具
3. **登录B站账号**
4. **切换到Network标签** → 找到任意请求
5. **查看Request Headers** → 复制Cookie中的对应值

### 方法二：使用程序自动登录

1. **留空配置文件中的cookies部分**
2. **运行程序** → 程序会自动提示扫码登录
3. **选择登录方式**：
   - 终端二维码显示
   - 生成二维码图片
   - 显示登录链接
4. **扫码完成** → 程序自动保存登录信息

---

## 🎯 使用技巧

### 特定序号模式优化

**观望临界值设置策略：**
- 🟢 **冷门装扮** (limit_v: 10-30) - 竞争小，可接近目标
- 🟡 **中等热度** (limit_v: 50-70) - 适中提前量
- 🔴 **热门装扮** (limit_v: 100-200) - 大幅提前，避免miss

**购买数量策略：**
- 🎯 **精准序号** - 买1个，节省B币
- 💎 **热门序号** - 买3-5个，提高成功率
- 🚀 **超热序号** - 买10+个，确保拿到

### 定时模式优化

**时间设置建议：**
- ⏰ **开售前1-2分钟** 启动程序进行准备
- 🎯 **精确到秒** 设置抢购时间
- 🔄 **预留缓冲** 考虑网络延迟

---

## 📊 装扮ID获取

### 方法一：从装扮页面URL获取

```
https://www.bilibili.com/h5/mall/suit/detail?id=33998
                                          ↑
                                      装扮ID
```

### 方法二：从API响应中获取

1. 在装扮详情页按F12打开开发者工具
2. 查找包含`item_id`的API请求
3. 从响应中获取正确的装扮ID

---

## 🛡️ 注意事项

### ⚠️ 重要提醒

- **合法使用** - 本工具仅供个人学习和合法使用
- **风险自担** - 使用本工具的风险由用户自行承担
- **禁止商用** - 严禁用于商业用途或装扮倒卖
- **账号安全** - 请保护好自己的登录信息

### 🔒 账号安全

- 定期更新密码和登录状态
- 不要在公共设备上保存登录信息
- 发现异常登录及时检查账号安全

### 💰 消费提醒

- 建议在测试环境先试用功能
- 确保B币余额充足再进行正式抢购

---

## 🔧 常见问题

### Q1: 编译失败怎么办？
```bash
# 确保Go版本1.17+
go version

# 清理模块缓存
go clean -modcache

# 重新下载依赖
go mod download

# 使用CGO_ENABLED=0编译
CGO_ENABLED=0 go build -o bilibili_rush main.go
```

### Q2: 登录状态失效？
- 重新运行程序，选择扫码登录
- 或手动更新config.json中的cookies信息

### Q3: 抢购失败？
- 检查网络连接是否稳定
- 确认装扮ID和时间设置正确
- 检查B币余额是否充足

### Q4: 找不到装扮ID？
- 从装扮详情页URL中获取
- 或使用浏览器开发者工具查看API请求

---

## 🤝 贡献指南

欢迎提交Issue和Pull Request来改进项目！

### 开发环境
- Go 1.17+
- Git
- 支持跨平台开发

### 贡献流程
1. Fork本项目
2. 创建功能分支
3. 提交代码更改
4. 发起Pull Request

---

## 📜 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情

---

<div align="center">

**🌟 如果这个项目对你有帮助，请给个星标支持一下！**

**⚠️ 请理性消费，合法使用本工具**

Made with ❤️ by developers, for collectors

</div> 