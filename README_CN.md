# WAF Rule Validator

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Release](https://img.shields.io/badge/Release-v1.0.0-blue.svg)](https://github.com/realguoxiufeng/WAF-Rule-Validator/releases)

[English](README.md)

WAF Rule Validator 是一个用于评估 Web 应用安全解决方案（WAF、API 网关、IPS）的工具。它通过生成恶意请求来测试安全防护规则的有效性，支持 REST、GraphQL、gRPC、SOAP、XMLRPC 等多种 API 协议。

## 功能特性

- **多协议支持**: 支持 REST、GraphQL、gRPC、SOAP、XMLRPC 等协议
- **多种编码方式**: Base64、URL、JSUnicode、Plain、XML Entity
- **多位置注入**: URL路径、URL参数、请求头、请求体、JSON、HTML表单等
- **OpenAPI 集成**: 支持基于 OpenAPI 规范生成请求模板
- **WAF 识别**: 自动识别主流 WAF 产品（Akamai、F5、Imperva、ModSecurity 等）
- **多格式报告**: 支持 PDF、HTML、JSON、DOCX 格式的评估报告

## 快速开始

### 环境要求

- Go 1.24 或更高版本
- Chrome 浏览器（用于生成 PDF 报告，可选）

### 构建

```bash
# 克隆仓库
git clone https://github.com/realguoxiufeng/WAF-Rule-Validator.git
cd WAF-Rule-Validator

# 构建二进制文件
make gotestwaf_bin

# 或者直接使用 go build
go build -o gotestwaf ./cmd/gotestwaf
```

### 基本使用

```bash
# 基本扫描
./gotestwaf --url=http://target-url --noEmailReport

# 使用 gRPC 测试
./gotestwaf --url=http://target-url --grpcPort 9000 --noEmailReport

# 使用 OpenAPI 规范生成请求模板
./gotestwaf --url=http://target-url --openapiFile api.yaml --noEmailReport

# 使用自定义测试用例
./gotestwaf --url=http://target-url --testCasesPath ./custom-testcases --noEmailReport
```

### Docker 使用

```bash
# 拉取镜像并运行
docker pull wallarm/gotestwaf
docker run --rm --network="host" -v ${PWD}/reports:/app/reports \
    wallarm/gotestwaf --url=<TARGET_URL> --noEmailReport
```

## 测试用例格式

测试用例使用 YAML 格式定义：

```yaml
payload:
  - "malicious string 1"
  - "malicious string 2"
encoder:
  - Base64Flat
  - URL
placeholder:
  - URLPath
  - JSONRequest
type: SQL Injection
```

- **payload**: 恶意攻击载荷
- **encoder**: 对载荷应用的编码方式
- **placeholder**: 载荷注入的请求位置
- **type**: 攻击类型名称

每个测试用例文件会生成 `len(payload) × len(encoder) × len(placeholder)` 个测试请求。

## 测试用例目录结构

| 目录 | 说明 |
|------|------|
| `testcases/owasp/` | OWASP Top-10 攻击向量（真正例测试，应被拦截） |
| `testcases/owasp-api/` | OWASP API 安全攻击向量 |
| `testcases/false-pos/` | 真负例测试（正常内容，应放行） |
| `testcases/community/` | 社区贡献的攻击向量 |

## 支持的编码器

| 编码器 | 说明 |
|--------|------|
| Base64 | Base64 编码 |
| Base64Flat | 无填充 Base64 编码 |
| URL | URL 编码 |
| JSUnicode | JavaScript Unicode 编码 |
| Plain | 原始文本（不编码） |
| XML Entity | XML 实体编码 |

## 支持的占位符

| 占位符 | 说明 |
|--------|------|
| URLPath | URL 路径 |
| URLParam | URL 参数 |
| Header | HTTP 请求头 |
| UserAgent | User-Agent 头 |
| RequestBody | 请求体 |
| JSONBody | JSON 请求体 |
| JSONRequest | JSON 请求 |
| HTMLForm | HTML 表单 |
| HTMLMultipartForm | 多部分表单 |
| SOAPBody | SOAP 消息体 |
| XMLBody | XML 请求体 |
| gRPC | gRPC 请求 |
| GraphQL | GraphQL 请求 |
| RawRequest | 原始 HTTP 请求 |

## 配置选项

```bash
Usage: ./gotestwaf [OPTIONS] --url <URL>

Options:
      --url string              目标 URL（必需）
      --grpcPort uint16         gRPC 端口
      --graphqlURL string       GraphQL URL
      --openapiFile string      OpenAPI 规范文件路径
      --testCasesPath string    测试用例目录路径（默认 "testcases"）
      --testCase string         仅运行指定测试用例
      --testSet string          仅运行指定测试集
      --httpClient string       HTTP 客户端类型: chrome, gohttp（默认 "gohttp"）
      --workers int             并发工作数（默认 5）
      --blockStatusCodes ints   WAF 拦截时的 HTTP 状态码（默认 [403]）
      --passStatusCodes ints    正常响应的 HTTP 状态码（默认 [200,404]）
      --blockRegex string       用于识别拦截页面的正则表达式
      --passRegex string        用于识别正常页面的正则表达式
      --reportFormat strings    报告格式: none, json, html, pdf, docx（默认 [pdf]）
      --reportPath string       报告存储目录（默认 "reports"）
      --reportName string       报告文件名
      --noEmailReport           保存报告到本地而不发送邮件
      --wafName string          WAF 产品名称（默认 "generic"）
      --skipWAFIdentification   跳过 WAF 识别
      --version                 显示版本信息
```

## 开发指南

### 运行测试

```bash
# 运行所有测试
go test -count=1 -v ./...

# 运行特定包的测试
go test -count=1 -v ./internal/db/...

# 运行集成测试
go test -count=1 -v ./tests/integration/...
```

### 代码检查

```bash
# 运行 linter
golangci-lint -v run ./...

# 格式化代码
go fmt ./...
goimports -local "github.com/wallarm/gotestwaf" -w <files>
```

### 项目结构

```
.
├── cmd/gotestwaf/          # 主入口点
├── internal/
│   ├── config/             # 配置管理
│   ├── db/                 # 测试用例数据库和统计
│   ├── payload/            # 载荷编码和占位符注入
│   │   ├── encoder/        # 编码器实现
│   │   └── placeholder/    # 占位符实现
│   ├── scanner/            # 扫描逻辑和 HTTP 客户端
│   │   └── clients/        # HTTP/gRPC/GraphQL 客户端
│   ├── openapi/            # OpenAPI 规范解析
│   └── report/             # 报告生成
├── pkg/                    # 导出包
│   ├── dnscache/           # DNS 缓存工具
│   └── report/             # 报告验证和辅助工具
├── testcases/              # 测试用例
└── tests/integration/      # 集成测试
```

## 开源许可

本项目基于 [MIT License](LICENSE) 开源。

## 致谢

本项目基于 [GoTestWAF](https://github.com/wallarm/gotestwaf) 开发，感谢 Wallarm 团队的贡献。