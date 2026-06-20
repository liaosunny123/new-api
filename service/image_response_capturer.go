package service

import (
	"bytes"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ImageResponseCapturer 是一个 gin.ResponseWriter 包装器，把响应体缓冲到内存中、
// 记录状态码而不立即下发。用于在 adaptor 写完响应后改写图片 URL，再统一提交给真实 writer。
//
// 注意：仅用于非流式图片响应。Header() 仍委托给底层 writer，因此 adaptor 设置的
// Content-Type 等响应头会被保留；只有状态码与响应体被缓冲。
type ImageResponseCapturer struct {
	gin.ResponseWriter
	body   *bytes.Buffer
	status int
}

func NewImageResponseCapturer(w gin.ResponseWriter) *ImageResponseCapturer {
	return &ImageResponseCapturer{
		ResponseWriter: w,
		body:           &bytes.Buffer{},
		status:         http.StatusOK,
	}
}

func (w *ImageResponseCapturer) WriteHeader(code int) {
	w.status = code
}

// WriteHeaderNow 拦截 gin 的提前写头行为，避免在提交前下发响应头。
func (w *ImageResponseCapturer) WriteHeaderNow() {}

func (w *ImageResponseCapturer) Write(b []byte) (int, error) {
	return w.body.Write(b)
}

func (w *ImageResponseCapturer) WriteString(s string) (int, error) {
	return w.body.WriteString(s)
}

func (w *ImageResponseCapturer) Written() bool {
	return false
}

func (w *ImageResponseCapturer) Status() int {
	return w.status
}

func (w *ImageResponseCapturer) Size() int {
	return w.body.Len()
}

// Flush 拦截 http.Flusher 行为，避免缓冲期间提前下发。
func (w *ImageResponseCapturer) Flush() {}

// Body 返回当前缓冲的响应体。
func (w *ImageResponseCapturer) Body() []byte {
	return w.body.Bytes()
}

// Commit 将最终响应体（可能已被改写）连同状态码下发给底层 writer。
func (w *ImageResponseCapturer) Commit(finalBody []byte) {
	underlying := w.ResponseWriter
	underlying.Header().Set("Content-Length", strconv.Itoa(len(finalBody)))
	underlying.WriteHeader(w.status)
	if len(finalBody) > 0 {
		_, _ = underlying.Write(finalBody)
	}
}
