package http

import (
	"context"
	"fmt"
	"net/http"

	httpx "github.com/hopeio/gox/net/http"
	"github.com/xuri/excelize/v2"
)

type ExcelFile struct {
	Name string
	*excelize.File
	Options []excelize.Options
}

func (res *ExcelFile) ServeHTTP(ctx context.Context, w http.ResponseWriter) {
	res.Respond(ctx, httpx.ResponseWriterWrapper{ResponseWriter: w})
}

func (res *ExcelFile) Respond(ctx context.Context, w http.ResponseWriter) {
	if wx, ok := w.(httpx.ResponseWriter); ok {
		header := wx.HeaderX()
		header.Set(httpx.HeaderContentType, httpx.ContentTypeOctetStream)
		header.Set(httpx.HeaderContentDisposition, fmt.Sprintf(httpx.AttachmentTmpl, res.Name))
	} else {
		header := w.Header()
		header.Set(httpx.HeaderContentType, httpx.ContentTypeOctetStream)
		header.Set(httpx.HeaderContentDisposition, fmt.Sprintf(httpx.AttachmentTmpl, res.Name))
	}
	res.File.WriteTo(w, res.Options...)
	res.File.Close()
}
