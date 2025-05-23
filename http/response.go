package http

import (
	"fmt"
	httpi "github.com/hopeio/utils/net/http"
	"github.com/hopeio/utils/net/http/consts"
	"github.com/xuri/excelize/v2"
	"net/http"
)

type ExcelFile struct {
	Name string
	*excelize.File
	Options []excelize.Options
}

func (res *ExcelFile) Response(w http.ResponseWriter) (int, error) {
	return res.CommonResponse(httpi.CommonResponseWriter{w})
}

func (res *ExcelFile) CommonResponse(w httpi.ICommonResponseWriter) (int, error) {
	header := w.Header()
	header.Set(consts.HeaderContentDisposition, fmt.Sprintf(consts.AttachmentTmpl, res.Name))
	header.Set(consts.HeaderContentType, consts.ContentTypeOctetStream)
	n, err := res.File.WriteTo(w, res.Options...)
	res.File.Close()
	return int(n), err
}
