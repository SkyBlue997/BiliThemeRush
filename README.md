# 🚀 BilibiliSuitRushBuy

> 高效的B站装扮自动抢购工具，支持双模式抢购策略

[![Go Version](https://img.shields.io/badge/Go-1.17+-00ADD8?style=flat-square&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue?style=flat-square)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Cross--Platform-lightgrey?style=flat-square)](https://github.com/)

---

## 🚀 快速开始

### 安装

```bash
# 克隆项目
git clone https://github.com/SkyBlue997/BiliThemeRush
cd BiliThemeRush

# 编译程序
CGO_ENABLED=0 go build -o bilibili_rush main.go
```

### 配置

```bash
# 编辑配置文件
vim config.json
```

### 运行

```bash
./bilibili_rush
```

---

## 📋 抢购模式

<table>
<tr>
<td width="50%">

### 🎯 特定序号模式
**适用场景**  
抢购特定粉丝编号（豹子号、炸弹号等）

**工作原理**  
持续监控当前装扮销售ID，接近目标时自动抢购

**优势**  
精确抢购指定序号，适合收藏特殊编号

</td>
<td width="50%">

### ⏰ 定时抢购模式
**适用场景**  
在装扮开售的准确时间点抢购

**工作原理**  
等待到指定时间后立即发起抢购请求

**优势**  
适合新装扮首发抢购，时间精确控制

</td>
</tr>
</table>

---

## ⚙️ 配置说明

### 配置文件结构

```json
{
  "api": {
    "csrf": "your_csrf_token",
    "cookie": "your_cookie_string", 
    "access_key": "your_access_key",
    "user_agent": "浏览器用户代理"
  },
  "pay": {
    "user_agent": "支付用户代理",
    "buvid": "设备标识",
    "device_id": "设备ID",
    "fp_local": "本地指纹",
    "fp_remote": "远程指纹",
    "session_id": "会话ID",
    "device_fingerprint": "设备指纹"
  },
  "item": {
    "id": 33998,
    "add_month": -1
  },
  "target_mode": {
    "target_id": 18168,
    "limit_v": 20,
    "buy_num": 1
  },
  "timed_mode": {
    "buy_num": 1
  }
}
```

### 参数详细说明

#### API 配置 (必需)
| 参数 | 说明 | 获取位置 | 示例 |
|------|------|----------|------|
| `csrf` | CSRF令牌 | API请求中的csrf参数 | `"abc123def456"` |
| `cookie` | Cookie信息 | API请求头中的Cookie | `"SESSDATA=xxx; bili_jct=yyy"` |
| `access_key` | 访问密钥 | 支付请求中的accessKey | `"1234567890abcdef"` |
| `user_agent` | 用户代理 | 通常不需要修改 | 默认值即可 |

#### 支付配置 (必需)
| 参数 | 说明 | 获取位置 |
|------|------|----------|
| `buvid` | 设备标识 | 支付请求头中的Buvid |
| `device_id` | 设备ID | 支付请求头中的Device-ID |
| `fp_local` | 本地指纹 | 支付请求头中的fp_local |
| `fp_remote` | 远程指纹 | 支付请求头中的fp_remote |
| `session_id` | 会话ID | 支付请求头中的session_id |
| `device_fingerprint` | 设备指纹 | 支付请求头中的deviceFingerprint |

#### 装扮配置
| 参数 | 说明 | 可选值 |
|------|------|--------|
| `id` | 装扮ID | 从装扮详情页URL获取 |
| `add_month` | 购买时长 | `-1`(永久) / `1`(一个月) |

#### 特定序号模式配置
| 参数 | 说明 | 推荐值 |
|------|------|--------|
| `target_id` | 目标粉丝编号 | 根据需要设置 |
| `limit_v` | 观望临界值 | 冷门:10 / 中等:70 / 热门:140 |
| `buy_num` | 购买数量 | 热门号建议多买几个 |

#### 定时模式配置
| 参数 | 说明 | 推荐值 |
|------|------|--------|
| `buy_num` | 购买数量 | 通常设置为1 |

---

## 🛠️ 参数获取

### 抓包工具推荐
- **Fiddler** - 支持发送断点，推荐
- **Charles** - Mac用户友好
- **Burp Suite** - 专业级工具

### 获取步骤

1. **开启抓包工具**
2. **登录B站，进入装扮页面**
3. **尝试购买装扮**（余额充足时请开启断点）
4. **从请求中提取参数**

> ⚠️ **重要提醒**  
> 如果B币余额充足，务必使用带发送断点功能的抓包工具，避免真实购买

---

## 📊 参数调优



### 购买策略

- **热门号码** → 增加购买数量 (`buy_num > 1`)
- **冷门号码** → 单个购买即可 (`buy_num = 1`)

---

## 🔒 免责声明

> **仅限个人使用**  
> 本工具仅供个人学习和合法使用，严禁用于商业用途或装扮倒卖

> **风险提醒**  
> 使用本工具的所有风险由用户自行承担，开发者不承担任何责任

---

## 🤝 贡献

欢迎提交 Issue 和 Pull Request 来改进项目

### 开发环境
- Go 1.17+
- 支持跨平台编译

---

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情

---

<div align="center">

**⭐ 如果这个项目对你有帮助，请给个星标支持一下！**

Made with ❤️ by developers, for developers

</div> 