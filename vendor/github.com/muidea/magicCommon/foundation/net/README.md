# magicCommon/foundation/net 模块说明

## 概述

`foundation/net` 提供一组面向 HTTP 和网络辅助场景的工具函数，主要覆盖：

- HTTP 请求与响应包装
- JSON body 解析
- 文件上传与下载
- 基于 DNS 缓存的 HTTP Client
- URL、状态码等网络辅助能力

## 核心能力

### HTTP Body 与响应

- `GetHTTPRequestBody(req)`：读取请求体并施加大小限制
- `ParseJSONBody(req, validator, param)`：解析 JSON body 并可选执行校验
- `PackageHTTPResponse(...)`
- `PackageHTTPResponseWithStatusCode(...)`

当前实现要点：

- 默认 body 读取限制为 10MB
- `ParseJSONBody` 仅接受 `application/json`
- 响应封装默认写入 `application/json; charset=utf-8`

### 文件上传与下载

- `MultipartFormFile(...)`
- `HTTPBodyToFile(...)`
- `HTTPDownload(...)`
- `HTTPUpload(...)`
- `HTTPUploadStream(...)`

当前实现要点：

- 会校验目标目录与文件名
- 非法目标目录或文件名会直接返回错误
- 下载、上传、落盘路径都会显式关闭文件句柄和响应体

### DNS Cache HTTP Client

- `NewDNSCacheHttpClient()`

当前实现要点：

- 会基于默认 transport 克隆一个新的 `*http.Transport`
- 只为新建 client 安装自定义 `DialContext`
- 不会再修改全局 `http.DefaultTransport`

这意味着：

- 新建 client 具备基于内部 resolver 的 DNS 查询能力
- 进程里其他 HTTP 调用不会被这个 helper 污染

## 当前限制

- `foundation/net` 当前以 helper 函数为主，没有统一 client 生命周期抽象
- DNS 刷新使用包级全局 resolver
- 文件上传/下载 helper 更偏向轻量包装，不包含更高层的重试、限流和观测能力
