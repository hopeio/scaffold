package crud

import (
	"github.com/gin-gonic/gin"
	"github.com/hopeio/gox/errors"
	ginx "github.com/hopeio/gox/net/http/gin"
	"github.com/hopeio/scaffold/errcode"

	"log"
	"net/http"
	"reflect"

	clausex "github.com/hopeio/gox/database/sql/gorm/clause"
	httpx "github.com/hopeio/gox/net/http"
	"github.com/hopeio/gox/net/http/gin/binding"
	stringsx "github.com/hopeio/gox/strings"
	"github.com/hopeio/gox/terminal/style"
	"github.com/hopeio/gox/types/response"
	"gorm.io/gorm"
)

const apiPrefix = "/api/"

func CRUD[T any](server *gin.Engine, db *gorm.DB, middleware ...gin.HandlerFunc) {
	Save[T](server, db, middleware...)
	Query[T](server, db, middleware...)
	Delete[T](server, db, middleware...)
}

func Save[T any](server *gin.Engine, db *gorm.DB, middleware ...gin.HandlerFunc) {
	var v T
	typ := stringsx.LowerCaseFirst(reflect.TypeOf(&v).Elem().Name())
	cu := append(middleware, func(c *gin.Context) {
		var data T
		err := binding.Bind(c, &data)
		if err != nil {
			ginx.Respond(c, &errors.ErrResp{Code: errors.ErrCode(errcode.InvalidArgument), Msg: err.Error()})
			return
		}
		if err := db.Save(&data).Error; err != nil {
			ginx.Respond(c, &errors.ErrResp{Code: errors.ErrCode(errcode.DBError), Msg: err.Error()})
			return
		}
		ginx.Respond(c, &httpx.ErrResp{})
	})
	url := apiPrefix + typ
	server.POST(url, cu...)
	Log(http.MethodPost, url, "create "+typ)
	url = apiPrefix + typ
	server.PUT(url, cu...)
	Log(http.MethodPut, url, "update "+typ)
	url = apiPrefix + typ + "/edit"
	server.POST(url, cu...)
	Log(http.MethodPost, url, "update "+typ)
	url = apiPrefix + typ + "/:id"
	server.PUT(url, append(middleware, func(c *gin.Context) {
		var data T
		err := binding.Bind(c, &data)
		if err != nil {
			ginx.Respond(c, &errors.ErrResp{Code: errors.ErrCode(errcode.InvalidArgument), Msg: err.Error()})
			return
		}
		if err = db.Clauses(clausex.ByPrimaryKey(c.Param("id"))).Updates(&data).Error; err != nil {
			ginx.Respond(c, &errors.ErrResp{Code: errors.ErrCode(errcode.DBError), Msg: err.Error()})
			return
		}
		ginx.Respond(c, &httpx.ErrResp{})
	})...)
	Log(http.MethodPut, url, "update "+typ)
}

func Delete[T any](server *gin.Engine, db *gorm.DB, middleware ...gin.HandlerFunc) {
	var v T
	typ := stringsx.LowerCaseFirst(reflect.TypeOf(&v).Elem().Name())
	url := apiPrefix + typ + "/:id"
	server.DELETE(url, append(middleware, func(c *gin.Context) {
		if err := db.Delete(&v, c.Param("id")).Error; err != nil {
			ginx.Respond(c, &errors.ErrResp{Code: errors.ErrCode(errcode.DBError), Msg: err.Error()})
			return
		}
		ginx.Respond(c, &httpx.ErrResp{})
	})...)
	Log(http.MethodDelete, url, "delete "+typ)

	handler := append(middleware, func(c *gin.Context) {
		var m map[string]any
		if err := c.ShouldBindJSON(&m); err != nil {
			ginx.Respond(c, &errors.ErrResp{Code: errors.ErrCode(errcode.InvalidArgument), Msg: err.Error()})
			return
		}
		if err := db.Delete(&v, m["id"]).Error; err != nil {
			ginx.Respond(c, &errors.ErrResp{Code: errors.ErrCode(errcode.DBError), Msg: err.Error()})
			return
		}
		ginx.Respond(c, &httpx.ErrResp{})
	})
	url = apiPrefix + typ
	server.DELETE(url, handler...)
	Log(http.MethodDelete, url, "delete "+typ)
	url = apiPrefix + typ + "/delete"
	server.POST(url, handler...)
	Log(http.MethodPost, url, "delete "+typ)
}

func Query[T any](server *gin.Engine, db *gorm.DB, middleware ...gin.HandlerFunc) {
	var v T
	typ := stringsx.LowerCaseFirst(reflect.TypeOf(&v).Elem().Name())
	url := apiPrefix + typ + "/:id"
	server.GET(url, append(middleware, func(c *gin.Context) {
		var data T
		if err := db.First(&data, c.Param("id")).Error; err != nil {
			ginx.Respond(c, &errors.ErrResp{Code: errors.ErrCode(errcode.DBError), Msg: err.Error()})
			return
		}
		ginx.Respond(c, httpx.NewSuccessRespData(data))
	})...)
	Log(http.MethodGet, url, "get "+typ)
}

func List[T any](server *gin.Engine, db *gorm.DB, middleware ...gin.HandlerFunc) {
	var v T
	typ := stringsx.LowerCaseFirst(reflect.TypeOf(&v).Elem().Name())
	url := apiPrefix + typ
	server.GET(url, append(middleware, func(c *gin.Context) {
		var count int64
		var page clausex.PaginationEmbedded
		err := binding.Bind(c, &page)
		if err != nil {
			ginx.Respond(c, &errors.ErrResp{Code: errors.ErrCode(errcode.InvalidArgument), Msg: err.Error()})
			return
		}
		var list []*T
		if clauses := page.ToPagination().Clauses(); len(clauses) > 0 {
			db = db.Clauses(clauses...)
		}
		if err = db.Count(&count).Error; err != nil {
			ginx.Respond(c, &errors.ErrResp{Code: errors.ErrCode(errcode.DBError), Msg: err.Error()})
			return
		}
		if err = db.Find(&list).Error; err != nil {
			ginx.Respond(c, &errors.ErrResp{Code: errors.ErrCode(errcode.DBError), Msg: err.Error()})
			return
		}
		if count == 0 {
			count = int64(len(list))
		}
		ginx.Respond(c, httpx.NewSuccessRespData(&response.List[*T]{List: list, Total: uint(count)}))
	})...)
	Log(http.MethodGet, url, "get "+typ)
}

func Log(method, path, title string) {
	log.Printf(" %s\t %s %s\t %s",
		style.Green("API:"),
		style.Yellow(stringsx.FormatLen(method, 6)),
		style.Blue(stringsx.FormatLen(path, 50)), style.Magenta(title))
}
