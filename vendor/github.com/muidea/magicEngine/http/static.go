package http

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/muidea/magicCommon/foundation/helper"
)

// StaticOptions 是指定静态文件服务配置选项的结构体
type StaticOptions struct {
	RootPath string
	// PrefixUri 是用于提供静态目录内容的可选前缀
	PrefixUri string
	// SkipLogging 在提供静态文件时禁用 [Static] 日志消息
	SkipLogging bool
	// IndexFile 定义作为索引服务的文件（如果存在）
	IndexFile string
	// Expires 定义用于生成 HTTP Expires 头的用户自定义函数
	// https://developers.google.com/speed/docs/insights/LeverageBrowserCaching
	Expires func() string
	// Fallback 定义在找不到请求资源时提供默认 URL
	Fallback string
	// ExcludeUri 定义此处理器不应处理的 URL 模式
	ExcludeUri string
}

func prepareStaticOptions(option *StaticOptions) StaticOptions {
	opt := *option

	if len(opt.IndexFile) == 0 {
		opt.IndexFile = "index.html"
	}
	if opt.PrefixUri != "" {
		if opt.PrefixUri[0] != '/' {
			opt.PrefixUri = "/" + opt.PrefixUri
		}
		opt.PrefixUri = strings.TrimRight(opt.PrefixUri, "/")
	}
	return opt
}

func resolveRootDirectory(rootPath string, relativeTo string) string {
	if filepath.IsAbs(rootPath) {
		return rootPath
	}
	dir := filepath.Join(relativeTo, rootPath)
	if filepath.IsAbs(dir) {
		return dir
	}
	return filepath.Join(Root, dir)
}

func serveStaticFile(dir http.Dir, opt StaticOptions, fileUri string, res http.ResponseWriter, req *http.Request, skipLogging bool) error {
	openFile, openErr := dir.Open(fileUri)
	if openErr != nil {
		if opt.Fallback != "" {
			fileUri = opt.Fallback
			openFile, openErr = dir.Open(opt.Fallback)
		}
		if openErr != nil {
			return openErr
		}
	}
	defer func(file http.File) {
		_ = file.Close()
	}(openFile)

	fileInfo, statErr := openFile.Stat()
	if statErr != nil {
		return statErr
	}

	if fileInfo.IsDir() {
		if !strings.HasSuffix(req.URL.Path, "/") {
			dest := url.URL{
				Path:     req.URL.Path + "/",
				RawQuery: req.URL.RawQuery,
				Fragment: req.URL.Fragment,
			}
			http.Redirect(res, req, dest.String(), http.StatusFound)
			return nil
		}

		fileUri = path.Join(fileUri, opt.IndexFile)
		openFile, openErr = dir.Open(fileUri)
		if openErr != nil {
			return openErr
		}
		defer func(file http.File) {
			_ = file.Close()
		}(openFile)

		fileInfo, statErr = openFile.Stat()
		if statErr != nil {
			return statErr
		}
		if fileInfo.IsDir() {
			return ErrURLNotFound
		}
	}

	if !skipLogging {
		slog.Info("serving static file", "uri", fileUri)
	}

	if opt.Expires != nil {
		res.Header().Set("Expires", opt.Expires())
	}

	http.ServeContent(res, req, fileUri, fileInfo.ModTime(), openFile)
	return nil
}

// static 静态文件处理器
type static struct {
	rootPath     string
	subPrefixUri string
}

// MiddleWareHandle 处理静态文件请求的中间件
func (s *static) MiddleWareHandle(ctx RequestContext, res http.ResponseWriter, req *http.Request) {
	staticOpt, staticOK := helper.GetValueFromContext[*StaticOptions](ctx.Context(), StaticOptionsKey{})
	if !staticOK {
		panicInfo("无法获取静态处理器")
	}

	if req.Method != GET && req.Method != HEAD {
		ctx.Next()
		return
	}

	opt := prepareStaticOptions(staticOpt)
	if opt.ExcludeUri != "" && strings.HasPrefix(req.URL.Path, opt.ExcludeUri) {
		ctx.Next()
		return
	}

	rootDirectory := resolveRootDirectory(staticOpt.RootPath, s.rootPath)
	dir := http.Dir(rootDirectory)

	fileUri := req.URL.Path
	prefixUrl := filepath.Join(opt.PrefixUri, s.subPrefixUri)
	if prefixUrl != "" {
		if !strings.HasPrefix(fileUri, prefixUrl) {
			ctx.Next()
			return
		}
		fileUri = fileUri[len(opt.PrefixUri):]
		if fileUri != "" && fileUri[0] != '/' {
			ctx.Next()
			return
		}
	}

	err := serveStaticFile(dir, opt, fileUri, res, req, opt.SkipLogging)
	if err != nil {
		ctx.Next()
	}
}

func StaticHandler(ctx context.Context, res http.ResponseWriter, req *http.Request) {
	staticOpt, staticOK := helper.GetValueFromContext[*StaticOptions](ctx, StaticOptionsKey{})
	if !staticOK {
		panicInfo("无法获取静态处理器")
	}

	rootDirectory := resolveRootDirectory(staticOpt.RootPath, Root)
	dir := http.Dir(rootDirectory)
	opt := prepareStaticOptions(staticOpt)
	uriFilePath := req.URL.Path

	if !strings.HasPrefix(uriFilePath, opt.PrefixUri) {
		res.WriteHeader(http.StatusNotFound)
		return
	}

	uriFilePath = uriFilePath[len(opt.PrefixUri):]

	err := serveStaticFile(dir, opt, uriFilePath, res, req, false)
	if err != nil {
		slog.Warn("failed to serve static file", "path", uriFilePath, "err", err)
		res.WriteHeader(http.StatusInternalServerError)
	}
}
