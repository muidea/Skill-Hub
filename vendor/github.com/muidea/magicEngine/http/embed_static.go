package http

import (
	"bytes"
	"embed"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"
)

type EmbedFile struct {
	ModTime     time.Time
	FileContext []byte
}

type EmbedStatic struct {
	embedPath      string
	prefixPath     string
	templateFS     embed.FS
	staticFileInfo sync.Map
}

// EmbedStaticOption configures an EmbedStatic
type EmbedStaticOption func(*EmbedStatic)

// WithPrefixPath sets the prefix path for static files
func WithPrefixPath(path string) EmbedStaticOption {
	return func(es *EmbedStatic) {
		es.prefixPath = path
	}
}

// WithEmbedPath sets the embed path for static files
func WithEmbedPath(path string) EmbedStaticOption {
	return func(es *EmbedStatic) {
		es.embedPath = path
	}
}

// NewEmbedStatic creates a new EmbedStatic with optional configuration
func NewEmbedStatic(templateFS embed.FS, opts ...EmbedStaticOption) *EmbedStatic {
	es := &EmbedStatic{
		templateFS:     templateFS,
		embedPath:      ".",       // default embed path
		prefixPath:     "/static", // default prefix path
		staticFileInfo: sync.Map{},
	}

	for _, opt := range opts {
		opt(es)
	}

	return es
}

// NewEmbedStaticWithPath creates a new EmbedStatic with explicit paths (backward compatibility)
func NewEmbedStaticWithPath(templateFS embed.FS, embedPath, prefixPath string) *EmbedStatic {
	return NewEmbedStatic(templateFS, WithEmbedPath(embedPath), WithPrefixPath(prefixPath))
}

func (s *EmbedStatic) MiddleWareHandle(ctx RequestContext, res http.ResponseWriter, req *http.Request) {
	var err error
	defer func() {
		if err != nil {
			ctx.Next()
		}
	}()

	if !strings.HasPrefix(req.URL.Path, s.prefixPath) {
		ctx.Next()
		return
	}

	if req.Method != "GET" && req.Method != "HEAD" {
		err = ErrMethodNotAllowed
		// log.Errorf("static middleware, url:%s, error: %v", req.URL.Path, err)
		return
	}

	filePath := s.validatePath(req.URL.Path)
	staticFile, staticModTime, staticErr := s.findEmbedFile(filePath)
	if staticErr != nil {
		err = NewStaticError(filePath, staticErr)
		// log.Errorf("static middleware, url:%s, error: %v", req.URL.Path, err)
		return
	}

	http.ServeContent(res, req, filePath, staticModTime, staticFile)
}

func (s *EmbedStatic) validatePath(filePath string) (ret string) {
	if filePath == "" {
		filePath = "/"
	}

	if s.isDir(filePath) {
		filePath = path.Join(filePath, "index.html")
	}

	ret = path.Join(s.embedPath, filePath)
	return
}

func (s *EmbedStatic) isDir(pathVal string) bool {
	return len(pathVal) > 0 && pathVal[len(pathVal)-1] == '/'
}

func (s *EmbedStatic) findEmbedFile(filePath string) (content io.ReadSeeker, modTime time.Time, err error) {
	fileInfo, fileOK := s.staticFileInfo.Load(filePath)
	if fileOK {
		content = bytes.NewReader(fileInfo.(EmbedFile).FileContext)
		modTime = fileInfo.(EmbedFile).ModTime
		return
	}

	fsInfo, fsErr := fs.Stat(s.templateFS, filePath)
	if fsErr != nil {
		err = fsErr
		return
	}
	contentVal, contentErr := fs.ReadFile(s.templateFS, filePath)
	if contentErr != nil {
		err = contentErr
		return
	}
	s.staticFileInfo.Store(filePath, EmbedFile{
		ModTime:     fsInfo.ModTime(),
		FileContext: contentVal,
	})

	content = bytes.NewReader(contentVal)
	modTime = fsInfo.ModTime()
	return
}
