package net

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/muidea/magicCommon/foundation/util"
	"log/slog"
)

// maxBytesReader 限制从底层读取器读取的字节数量不超过指定的最大值
type maxBytesReader struct {
	res               http.ResponseWriter // HTTP响应写入器
	reader            io.ReadCloser       // 底层读取器
	maxBytesRemaining int64               // 剩余最大字节数
	err               error               // 持久性错误
}

func (l *maxBytesReader) Read(buffer []byte) (n int, err error) {
	if l.err != nil {
		return 0, l.err
	}
	if len(buffer) == 0 {
		return 0, nil
	}
	// 如果请求读取的字节数比剩余允许的最大字节数大，
	// 则限制读取缓冲区大小为剩余字节数+1，以检测是否超出限制。
	if int64(len(buffer)) > l.maxBytesRemaining+1 {
		buffer = buffer[:l.maxBytesRemaining+1]
	}
	n, err = l.reader.Read(buffer)

	if int64(n) <= l.maxBytesRemaining {
		l.maxBytesRemaining -= int64(n)
		l.err = err
		return n, err
	}

	n = int(l.maxBytesRemaining)
	l.maxBytesRemaining = 0

	// 服务端和客户端代码都使用 maxBytesReader。
	// 这个 "requestTooLarge" 检查只在服务端代码中使用。
	// 为了防止仅使用HTTP客户端代码的二进制文件（如 cmd/go）也链接HTTP服务器，
	// 不要对服务器 "*response" 类型使用静态类型断言，
	// 而是检查这个接口：
	type requestTooLarger interface {
		requestTooLarge()
	}
	if res, ok := l.res.(requestTooLarger); ok {
		res.requestTooLarge()
	}
	l.err = errors.New("http: request body too large")
	return n, l.err
}

func (l *maxBytesReader) Close() error {
	return l.reader.Close()
}

// GetHTTPRemoteAddress get http remote address
func GetHTTPRemoteAddress(req *http.Request) (ret string) {
	ret = req.Header.Get("x-forwarded-for")
	if ret == "" {
		ret = req.RemoteAddr
	}

	ret = strings.Split(ret, ", ")[0]

	return
}

// GetHTTPRequestBody get http request body
func GetHTTPRequestBody(req *http.Request) (ret []byte, err error) {
	var reader io.Reader = req.Body
	maxFormSize := int64(1<<63 - 1)
	if _, ok := req.Body.(*maxBytesReader); !ok {
		maxFormSize = int64(10 << 20) // 10 MB is a lot of text.
		reader = io.LimitReader(req.Body, maxFormSize+1)
	}

	payload, payloadErr := io.ReadAll(reader)
	if payloadErr != nil {
		err = payloadErr
		slog.Error("read request body error", "error", err)
		return
	}

	if int64(len(payload)) > maxFormSize {
		err = errors.New("http: request body too large")
		return
	}

	ret = payload
	return
}

func HTTPBodyToFile(req *http.Request, dstFilePath, fileName string) (err error) {
	var reader io.Reader = req.Body
	var maxFormSize int64
	if _, ok := req.Body.(*maxBytesReader); !ok {
		maxFormSize = int64(10 << 20) // 10 MB is a lot of text.
		reader = io.LimitReader(req.Body, maxFormSize+1)
	}
	// 验证 dstFilePath 是否为合法的目录路径
	if !isValidDirectory(dstFilePath) {
		err = fmt.Errorf("invalid destination directory: %s", dstFilePath)
		slog.Error("invalid destination directory, err", "error", err.Error())
		return
	}

	// 验证文件名是否合法
	if !isValidFileName(fileName) {
		err = fmt.Errorf("invalid file name: %s", fileName)
		slog.Error("invalid file name, err", "error", err.Error())
		return
	}

	// 构建目标文件的完整路径
	dstFullFilePath := filepath.Join(dstFilePath, fileName)
	// 创建目标文件
	dstFileHandle, dstFileErr := os.Create(dstFullFilePath)
	if dstFileErr != nil {
		err = dstFileErr
		slog.Error("create destination file failed, err", "error", err.Error())
		return
	}
	defer func() { _ = dstFileHandle.Close() }()

	_, err = io.Copy(dstFileHandle, reader)
	if err != nil {
		slog.Error("copy destination file failed, err", "error", err.Error())
		return
	}
	return
}

// ParseJSONBody 解析http body请求提交的json数据
func ParseJSONBody(req *http.Request, validator util.Validator, param any) error {
	if err := util.ValidatePtr(param); err != nil {
		return err
	}

	if req.Body == nil {
		return errors.New("missing form body")
	}

	contentType, _, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
	if err != nil {
		slog.Error("parse content-type error", "error", err)
		return err
	}

	switch contentType {
	case "application/json":
		payload, payloadErr := GetHTTPRequestBody(req)
		if payloadErr != nil {
			slog.Error("get http request body error", "error", payloadErr)
			return payloadErr
		}

		err = json.Unmarshal(payload, param)
		if err != nil {
			slog.Error("unmarshal http request body error", "error", err)
			return err
		}

		if validator != nil {
			err = validator.Validate(param)
			if err != nil {
				slog.Error("validate http request body error", "error", err)
				return err
			}
		}

	default:
		return errors.New("invalid contentType, contentType:" + contentType)
	}

	return nil
}

var contentType = textproto.CanonicalMIMEHeaderKey("content-type")

func verifyContentType(res http.ResponseWriter) {
	contentVal := res.Header().Get(contentType)
	if contentVal != "" {
		return
	}
	res.Header().Set(contentType, "application/json; charset=utf-8")
}
func PackageHTTPResponse(res http.ResponseWriter, result any) {
	verifyContentType(res)

	if result == nil {
		res.WriteHeader(http.StatusOK)
		return
	}

	block, err := json.Marshal(result)
	if err == nil {
		_, err := res.Write(block)
		if err != nil {
			slog.Error("write result error", "error", err)
		}
		return
	}

	slog.Error("marshal result error", "error", err)
	res.WriteHeader(http.StatusExpectationFailed)
}

func PackageHTTPResponseWithStatusCode(res http.ResponseWriter, statusCode int, result any) {
	verifyContentType(res)

	res.WriteHeader(statusCode)
	if result == nil {
		return
	}

	block, err := json.Marshal(result)
	if err == nil {
		_, err := res.Write(block)
		if err != nil {
			slog.Error("write result error", "error", err)
		}
		return
	}

	slog.Error("marshal result error", "error", err)
	res.WriteHeader(http.StatusInternalServerError)
}

// HTTPGet http get request
func HTTPGet(httpClient *http.Client, url string, result any, headers ...url.Values) ([]byte, error) {
	config := &HTTPRequestConfig{
		Method:  "GET",
		URL:     url,
		Body:    nil,
		Headers: headers,
	}
	return executeHTTPRequest(httpClient, config, result)
}

// HTTPPost http post request
func HTTPPost(httpClient *http.Client, url string, param any, result any, headers ...url.Values) ([]byte, error) {
	return executeHTTPRequestWithBody(httpClient, "POST", url, param, result, headers...)
}

// HTTPPut http put request
func HTTPPut(httpClient *http.Client, url string, param any, result any, headers ...url.Values) ([]byte, error) {
	return executeHTTPRequestWithBody(httpClient, "PUT", url, param, result, headers...)
}

// HTTPDelete http delete request
func HTTPDelete(httpClient *http.Client, url string, result any, headers ...url.Values) ([]byte, error) {
	config := &HTTPRequestConfig{
		Method:  "DELETE",
		URL:     url,
		Body:    nil,
		Headers: headers,
	}
	return executeHTTPRequest(httpClient, config, result)
}

// HTTPDownload http download file
func HTTPDownload(httpClient *http.Client, url string, filePath string, headers ...url.Values) (string, error) {
	request, requestErr := http.NewRequest("GET", url, nil)
	if requestErr != nil {
		slog.Error("construct request failed", "url", url, "error", requestErr)
		return "", requestErr
	}

	for _, val := range headers {
		for k, v := range val {
			request.Header.Set(k, v[0])
		}
	}

	response, responseErr := httpClient.Do(request)
	if responseErr != nil {
		slog.Error("post request failed", "error", responseErr.Error())
		return "", responseErr
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("unexpect statusCode, statusCode:%d", response.StatusCode)
		return "", errors.New(msg)
	}

	f, err := os.OpenFile(filePath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		slog.Error("open destination file failed, err", "error", err.Error())
		return "", err
	}
	defer func() { _ = f.Close() }()

	_, err = io.Copy(f, response.Body)
	if err != nil {
		slog.Error("write destination file content exception, err", "error", err.Error())
		return "", err
	}

	return filePath, nil
}

// HTTPUpload http upload file
func HTTPUpload(httpClient *http.Client, url, fileItem, filePath string, result any, headers ...url.Values) error {
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	//关键的一步操作
	fileWriter, err := bodyWriter.CreateFormFile(fileItem, filePath)
	if err != nil {
		slog.Error("error writing to buffer")
		return err
	}

	//打开文件句柄操作
	fh, err := os.Open(filePath)
	if err != nil {
		slog.Error("error opening file")
		return err
	}
	defer func() { _ = fh.Close() }()

	//iocopy
	_, err = io.Copy(fileWriter, fh)
	if err != nil {
		return err
	}

	contentType := bodyWriter.FormDataContentType()
	_ = bodyWriter.Close()

	request, requestErr := http.NewRequest("POST", url, bodyBuf)
	if requestErr != nil {
		err = requestErr
		slog.Error("construct request failed", "url", url, "error", err)
		return err
	}

	request.Header.Set("content-type", contentType)
	for _, val := range headers {
		for k, v := range val {
			request.Header.Set(k, v[0])
		}
	}

	response, responseErr := httpClient.Do(request)
	if responseErr != nil {
		slog.Error("post request failed", "error", responseErr)
		return responseErr
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("unexpect statusCode, statusCode:%d", response.StatusCode)
		return errors.New(msg)
	}

	if result != nil {
		content, err := io.ReadAll(response.Body)
		if err != nil {
			slog.Error("read respose data failed, err", "error", err.Error())
			return err
		}

		err = json.Unmarshal(content, result)
		if err != nil {
			slog.Error("unmarshal data failed, err", "error", err.Error())
			return err
		}
	}

	return nil
}

// HTTPUpload http upload file
func HTTPUploadStream(httpClient *http.Client, url string, byteReader io.Reader, result any, headers ...url.Values) error {
	request, requestErr := http.NewRequest("POST", url, byteReader)
	if requestErr != nil {
		slog.Error("construct request failed", "url", url, "error", requestErr)
		return requestErr
	}

	request.Header.Set("content-type", "application/octet-stream")
	for _, val := range headers {
		for k, v := range val {
			request.Header.Set(k, v[0])
		}
	}

	response, responseErr := httpClient.Do(request)
	if responseErr != nil {
		slog.Error("post request failed", "error", responseErr)
		return responseErr
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("unexpect statusCode, statusCode:%d", response.StatusCode)
		return errors.New(msg)
	}

	if result != nil {
		content, err := io.ReadAll(response.Body)
		if err != nil {
			slog.Error("read respose data failed, err", "error", err.Error())
			return err
		}

		err = json.Unmarshal(content, result)
		if err != nil {
			slog.Error("unmarshal data failed, err", "error", err.Error())
			return err
		}
	}

	return nil
}
