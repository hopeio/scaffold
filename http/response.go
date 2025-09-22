package http

import (
	"fmt"
	httpx "github.com/hopeio/gox/net/http"
	"github.com/hopeio/gox/net/http/consts"
	"github.com/xuri/excelize/v2"
	"net/http"
)

type ExcelFile struct {
	Name string
	*excelize.File
	Options []excelize.Options
}

func (res *ExcelFile) Response(w http.ResponseWriter) (int, error) {
	return res.CommonResponse(httpx.CommonResponseWriter{w})
}

func (res *ExcelFile) CommonResponse(w httpx.ICommonResponseWriter) (int, error) {
	header := w.Header()
	header.Set(consts.HeaderContentDisposition, fmt.Sprintf(consts.AttachmentTmpl, res.Name))
	header.Set(consts.HeaderContentType, consts.ContentTypeOctetStream)
	n, err := res.File.WriteTo(w, res.Options...)
	res.File.Close()
	return int(n), err
}
