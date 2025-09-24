package crud

import (
	"github.com/gin-gonic/gin"
	"github.com/hopeio/scaffold/errcode"

	"log"
	"net/http"
	"reflect"

	clausex "github.com/hopeio/gox/database/sql/gorm/clause"
	errcodex "github.com/hopeio/gox/errors/errcode"
	httpx "github.com/hopeio/gox/net/http"
	"github.com/hopeio/gox/net/http/gin/binding"
	stringsx "github.com/hopeio/gox/strings"
	"github.com/hopeio/gox/terminal/style"
	"github.com/hopeio/gox/types/param"
	"github.com/hopeio/gox/types/result"
	"gorm.io/gorm"
)

const apiPrefix = "/api/v1"

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
			c.JSON(http.StatusOK, &errcodex.ErrRep{Code: errcodex.ErrCode(errcode.InvalidArgument), Msg: err.Error()})
			return
		}
		if err := db.Save(&data).Error; err != nil {
			c.JSON(http.StatusOK, &errcodex.ErrRep{Code: errcodex.ErrCode(errcode.DBError), Msg: err.Error()})
			return
		}
		c.JSON(http.StatusOK, httpx.ResponseOk)
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
			c.JSON(http.StatusOK, &errcodex.ErrRep{Code: errcodex.ErrCode(errcode.InvalidArgument), Msg: err.Error()})
			return
		}
		if err = db.Clauses(clausex.ByPrimaryKey(c.Param("id"))).Updates(&data).Error; err != nil {
			c.JSON(http.StatusOK, &errcodex.ErrRep{Code: errcodex.ErrCode(errcode.DBError), Msg: err.Error()})
			return
		}
		c.JSON(http.StatusOK, httpx.ResponseOk)
	})...)
	Log(http.MethodPut, url, "update "+typ)
}

func Delete[T any](server *gin.Engine, db *gorm.DB, middleware ...gin.HandlerFunc) {
	var v T
	typ := stringsx.LowerCaseFirst(reflect.TypeOf(&v).Elem().Name())
	url := apiPrefix + typ + "/:id"
	server.DELETE(url, append(middleware, func(c *gin.Context) {
		if err := db.Delete(&v, c.Param("id")).Error; err != nil {
			c.JSON(http.StatusOK, &errcodex.ErrRep{Code: errcodex.ErrCode(errcode.DBError), Msg: err.Error()})
			return
		}
		c.JSON(http.StatusOK, httpx.ResponseOk)
	})...)
	Log(http.MethodDelete, url, "delete "+typ)

	handler := append(middleware, func(c *gin.Context) {
		var m map[string]any
		if err := c.ShouldBindJSON(&m); err != nil {
			c.JSON(http.StatusOK, &errcodex.ErrRep{Code: errcodex.ErrCode(errcode.InvalidArgument), Msg: err.Error()})
			return
		}
		if err := db.Delete(&v, m["id"]).Error; err != nil {
			c.JSON(http.StatusOK, &errcodex.ErrRep{Code: errcodex.ErrCode(errcode.DBError), Msg: err.Error()})
			return
		}
		c.JSON(http.StatusOK, httpx.ResponseOk)
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
			c.JSON(http.StatusOK, &errcodex.ErrRep{Code: errcodex.ErrCode(errcode.DBError), Msg: err.Error()})
			return
		}
		c.JSON(http.StatusOK, httpx.NewSuccessRespData(data))
	})...)
	Log(http.MethodGet, url, "get "+typ)
}

func List[T any](server *gin.Engine, db *gorm.DB, middleware ...gin.HandlerFunc) {
	var v T
	typ := stringsx.LowerCaseFirst(reflect.TypeOf(&v).Elem().Name())
	url := apiPrefix + typ
	server.GET(url, append(middleware, func(c *gin.Context) {
		var count int64
		var page param.PageEmbed
		err := binding.Bind(c, &page)
		if err != nil {
			c.JSON(http.StatusOK, &errcodex.ErrRep{Code: errcodex.ErrCode(errcode.InvalidArgument), Msg: err.Error()})
			return
		}
		var list []*T
		if page.PageNo > 0 && page.PageSize > 0 {
			db = db.Offset((page.PageNo - 1) * page.PageSize).Limit(page.PageSize)
			if err := db.Count(&count).Error; err != nil {
				c.JSON(http.StatusOK, &errcodex.ErrRep{Code: errcodex.ErrCode(errcode.DBError), Msg: err.Error()})
				return
			}
		}
		if err := db.Find(&list).Error; err != nil {
			c.JSON(http.StatusOK, &errcodex.ErrRep{Code: errcodex.ErrCode(errcode.DBError), Msg: err.Error()})
			return
		}
		if count == 0 {
			count = int64(len(list))
		}
		c.JSON(http.StatusOK, httpx.NewSuccessRespData(&result.List[*T]{List: list, Total: uint(count)}))
	})...)
	Log(http.MethodGet, url, "get "+typ)
}

func Log(method, path, title string) {
	log.Printf(" %s\t %s %s\t %s",
		style.Green("API:"),
		style.Yellow(stringsx.FormatLen(method, 6)),
		style.Blue(stringsx.FormatLen(path, 50)), style.Magenta(title))
}
