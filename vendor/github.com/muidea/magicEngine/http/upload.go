package http

import (
	"context"
	"net/http"
	"path/filepath"

	fn "github.com/muidea/magicCommon/foundation/net"
)

const DefaultMaxFileSize = 1024 * 1024 * 10
const defaultFildField = "file"

type RelativePath struct{}
type FileName struct{}
type FileField struct{}

type UploadCallbackFunc func(ctx context.Context, res http.ResponseWriter, req *http.Request, filePath string, err error)

type uploadRoute struct {
	uriPattern     string
	method         string
	rootUploadPath string
	relativePath   string
	maxFileSize    int64
	callbackFunc   UploadCallbackFunc
}

func (s *uploadRoute) Pattern() string {
	return s.uriPattern
}

func (s *uploadRoute) Method() string {
	return s.method
}

func (s *uploadRoute) Handler() RouteHandleFunc {
	return s.uploadFun
}

func (s *uploadRoute) getUploadContext(ctx context.Context, _ http.ResponseWriter, _ *http.Request) (relativePath, fileField, fileName string) {
	if rVal := ctx.Value(RelativePath{}); rVal != nil {
		relativePath = rVal.(string)
	} else {
		relativePath = s.relativePath
	}

	if fVal := ctx.Value(FileField{}); fVal != nil {
		fileField = fVal.(string)
	} else {
		fileField = defaultFildField
	}
	if fName := ctx.Value(FileName{}); fName != nil {
		fileName = fName.(string)
	}
	return
}

func (s *uploadRoute) uploadFun(ctx context.Context, res http.ResponseWriter, req *http.Request) {
	relativePath, fileField, fileName := s.getUploadContext(ctx, res, req)

	var filePath string
	var err error
	defer func() {
		if s.callbackFunc != nil {
			s.callbackFunc(ctx, res, req, filePath, err)
			return
		}
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		res.WriteHeader(http.StatusOK)
	}()

	err = req.ParseMultipartForm(s.maxFileSize)
	if err != nil {
		return
	}

	finalFilePath := filepath.Join(s.rootUploadPath, relativePath)
	fileName, fileErr := fn.MultipartFormFile(req, fileField, finalFilePath, fileName)
	if fileErr != nil {
		err = fileErr
		return
	}
	filePath = filepath.Join(relativePath, fileName)
}

func CreateUploadRoute(uriPattern, method, rootUploadPath, relativePath string, maxFileSize int64, callbackFunc UploadCallbackFunc) Route {
	return &uploadRoute{
		uriPattern:     uriPattern,
		method:         method,
		rootUploadPath: rootUploadPath,
		relativePath:   relativePath,
		maxFileSize:    maxFileSize,
		callbackFunc:   callbackFunc,
	}
}
