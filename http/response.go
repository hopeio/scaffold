package http

import (
	"fmt"
	httpi "github.com/hopeio/utils/net/http"
	"github.com/xuri/excelize/v2"
	"io"
	"net/http"
)

type ExcelFile struct {
	Name string
	*excelize.File
	Options []excelize.Options
}

func (res *ExcelFile) StatusCode() int {
	return http.StatusOK
}

func (res *ExcelFile) RespHeader() map[string]string {
	return map[string]string{httpi.HeaderContentType: httpi.ContentTypeOctetStream, httpi.HeaderContentDisposition: fmt.Sprintf(httpi.AttachmentTmpl, res.Name)}
}

func (res *ExcelFile) WriteTo(writer io.Writer) (int64, error) {
	return res.File.WriteTo(writer, res.Options...)
}

func (res *ExcelFile) Close() error {
	return res.File.Close()
}
