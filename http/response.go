package http

import (
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

func (res *ExcelFile) Respond(w http.ResponseWriter) (int, error) {
	return res.CommonRespond(httpx.CommonResponseWriter{ResponseWriter: w})
}

func (res *ExcelFile) CommonRespond(w httpx.ICommonResponseWriter) (int, error) {
	header := w.Header()
	header.Set(httpx.HeaderContentDisposition, fmt.Sprintf(httpx.AttachmentTmpl, res.Name))
	header.Set(httpx.HeaderContentType, httpx.ContentTypeOctetStream)
	n, err := res.File.WriteTo(w, res.Options...)
	res.File.Close()
	return int(n), err
}
