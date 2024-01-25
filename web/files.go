package web

import (
	"fmt"
	lru "github.com/hashicorp/golang-lru"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"web/context"
	webHandler "web/handler"
)

func WithMoreExtension(extMap map[string]string) StaticResourceHandlerOption {
	return func(h *StaticResourceHandler) {
		for ext, contenType := range extMap {
			h.extensionContentTypeMap[ext] = contenType
		}
	}
}

// WithFileCache 静态文件将会被缓存
// maxFileSizeThreshold 超过这个大小的文件，就被认为是大文件，我们将不会缓存
// maxCacheFileCnt 最多缓存多少个文件
// 所以我们最多缓存 maxFileSizeThreshold * maxCacheFileCnt
func WithFileCache(maxFileSizeThreshold int, maxCacheFileCnt int) StaticResourceHandlerOption {
	return func(h *StaticResourceHandler) {
		cache, err := lru.New(maxCacheFileCnt)
		if err != nil {
			log.Printf("创建缓存失败，将不会缓存静态资源")
		}
		h.maxFileSize = maxFileSizeThreshold
		h.cache = cache
	}

}

type fileCacheItem struct {
	fileName    string
	fileSize    int
	contentType string
	data        []byte
}

func (h *StaticResourceHandler) writeItemAsResponse(item *fileCacheItem, writer http.ResponseWriter) {
	writer.WriteHeader(http.StatusOK)
	writer.Header().Set("Content-Type", item.contentType)
	writer.Header().Set("Content-Length", fmt.Sprintf("%d", item.fileSize))
	_, _ = writer.Write(item.data)
}

func (h *StaticResourceHandler) readFileFromData(filename string) (*fileCacheItem, bool) {
	if h.cache != nil {
		if item, ok := h.cache.Get(filename); ok {
			return item.(*fileCacheItem), true
		}
	}
	return nil, false
}

func (h *StaticResourceHandler) cacheFile(item *fileCacheItem) {
	if h.cache != nil && item.fileSize < h.maxFileSize {
		h.cache.Add(item.fileName, item)
	}
}

// 传入文件名，得到文件的后缀，如txt, zip等
func getFileExt(name string) string {
	index := strings.LastIndex(name, ".")
	if index == len(name)-1 {
		// 此时文件名无后缀
		return ""
	}
	return name[index+1:]
}

func (h *StaticResourceHandler) Handle(ctx *context.Context) {
	filename, _ := ctx.PathValue("file").String()
	if item, ok := h.readFileFromData(filename); ok {
		log.Printf("从缓存中读取数据...")
		h.writeItemAsResponse(item, ctx.Response)
		return
	}
	dirPath := filepath.Join(h.dir, filename)
	f, err := os.Open(dirPath)
	if err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}
	ext := getFileExt(f.Name())
	contentType, ok := h.extensionContentTypeMap[ext]
	if !ok {
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}
	data, err := ioutil.ReadAll(f)
	if err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}
	item := &fileCacheItem{
		fileName:    filename,
		fileSize:    len(data),
		contentType: contentType,
		data:        data,
	}
	h.cacheFile(item)
	h.writeItemAsResponse(item, ctx.Response)
}

func NewStaticResourceHandler(dir string, pathPrefix string,
	options ...StaticResourceHandlerOption) *StaticResourceHandler {
	res := &StaticResourceHandler{
		dir:        dir,
		pathPrefix: pathPrefix,
		extensionContentTypeMap: map[string]string{
			// 这里根据自己的需要不断添加
			"jpeg": "image/jpeg",
			"jpe":  "image/jpeg",
			"jpg":  "image/jpeg",
			"png":  "image/png",
			"pdf":  "image/pdf",
		},
	}
	for _, opt := range options {
		opt(res)
	}
	return res
}

type StaticResourceHandler struct {
	dir        string
	pathPrefix string
	// extensionContentTypeMap用户指定的 ContentType
	extensionContentTypeMap map[string]string

	// 缓存静态资源的限制
	cache       *lru.Cache
	maxFileSize int
}

type StaticResourceHandlerOption func(h *StaticResourceHandler)

// FileDownloader 直接操作了 http.ResponseWriter
// 所以在 Middleware 里面将不能使用 RespData
// 因为没有赋值
type FileDownloader struct {
	Dir string
}

func (f *FileDownloader) Handle() webHandler.HandleFunc {
	return func(ctx *context.Context) {
		path, _ := ctx.QueryValue("file").String()
		dirPath := filepath.Join(f.Dir, filepath.Clean(path))
		// Base返回路径的最后一个元素, fn 为文件名
		fn := filepath.Base(dirPath)
		header := ctx.Response.Header()
		// Content-Dispostion：在指定为 attachment 的 时候，就是保存到本地。同时我们还设置了 filename。
		header.Set("Content-Disposition", "attachment;filename="+fn)
		header.Set("Content-Description", "File Transfer")
		// Content-Type：这里用的是 octet-stream，代表 的是通用的二进制文件。
		// 如果我们知道确切类型， 就可以换别的，例如 video、PDF 之类的。
		header.Set("Content-Type", "application/octet-stream")
		// Content-Transfer-Encoding: 这里设置为 binary，相当于直接传输。
		header.Set("Content-Transfer-Encoding", "binary")
		header.Set("Expires", "0")
		header.Set("Cache-Control", "must-revalidate")
		header.Set("Pragma", "public")
		// 将文件下载到前端
		http.ServeFile(ctx.Response, ctx.Request, dirPath)
	}
}

type FileUploader struct {
	// FileField 对应于文件在表单中的字段名字
	FileField string
	// DstPathFunc 用于计算目标路径
	DstPathFunc func(fh *multipart.FileHeader) string
}

func (f *FileUploader) Handle() webHandler.HandleFunc {
	// 这里可以额外做一些检测
	// if f.FileField == "" {
	// 	// 这种方案默认值我其实不是很喜欢
	// 	// 因为我们需要教会用户说，这个 file 是指什么意思
	// 	f.FileField = "file"
	// }
	return func(ctx *context.Context) {
		// src 代表文件, srcHeader 代表文件头
		src, srcHeader, err := ctx.Request.FormFile(f.FileField)
		if err != nil {
			ctx.RespStatusCode = http.StatusBadRequest
			ctx.RespData = []byte("上传失败，未找到数据")
			log.Fatalln(err)
			return
		}

		defer src.Close()
		//  dst 为文件要上传的目标路径
		dst, err := os.OpenFile(f.DstPathFunc(srcHeader), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o666)
		if err != nil {
			ctx.RespStatusCode = http.StatusInternalServerError
			ctx.RespData = []byte("上传失败")
			log.Fatalln(err)
			return
		}

		defer dst.Close()
		// 第一个参数为要写入的对象， 第二个参数则为要读取的对象， 第三参数为buffuer
		_, err = io.CopyBuffer(dst, src, nil)
		if err != nil {
			ctx.RespStatusCode = http.StatusInternalServerError
			ctx.RespData = []byte("上传失败")
			log.Fatalln(err)
			return
		}
		ctx.RespData = []byte("上传成功")
	}
}

// HandleFunc 这种设计方案也是可以的，但是不如上一种灵活。
// 它可以直接用来注册路由
// 上一种可以在返回 HandleFunc 之前可以继续检测一下传入的字段
// 这种形态和 Option 模式配合就很好
//func (f *FileUploader) HandleFunc(ctx *Context) {
//	src, srcHeader, err := ctx.Request.FormFile(f.FileField)
//	if err != nil {
//		ctx.RespStatusCode = 400
//		ctx.RespData = []byte("上传失败，未找到数据")
//		log.Fatalln(err)
//		return
//	}
//	defer src.Close()
//	dst, err := os.OpenFile(f.DstPathFunc(srcHeader),
//		os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o666)
//	if err != nil {
//		ctx.RespStatusCode = 500
//		ctx.RespData = []byte("上传失败")
//		log.Fatalln(err)
//		return
//	}
//	defer dst.Close()
//
//	_, err = io.CopyBuffer(dst, src, nil)
//	if err != nil {
//		ctx.RespStatusCode = 500
//		ctx.RespData = []byte("上传失败")
//		log.Fatalln(err)
//		return
//	}
//	ctx.RespData = []byte("上传成功")
//}
