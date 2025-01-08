package crud

import (
	"github.com/gin-gonic/gin"
	"github.com/hopeio/protobuf/errcode"
	errcode2 "github.com/hopeio/utils/errors/errcode"
	httpi "github.com/hopeio/utils/net/http"
	"github.com/hopeio/utils/net/http/gin/binding"
	stringsi "github.com/hopeio/utils/strings"
	"github.com/hopeio/utils/terminal/style"
	"gorm.io/gorm"
	"log"
	"net/http"
	"reflect"
)

func CRUD[T any](server *gin.Engine, db *gorm.DB, keyParam string, middleware ...gin.HandlerFunc) {
	Save[T](server, db, keyParam, middleware...)
	Query[T](server, db, keyParam, middleware...)
	Delete[T](server, db, keyParam, middleware...)
}

func Save[T any](server *gin.Engine, db *gorm.DB, keyParam string, middleware ...gin.HandlerFunc) {
	var v T
	typ := stringsi.LowerCaseFirst(reflect.TypeOf(&v).Elem().Name())
	cu := append(middleware, func(c *gin.Context) {
		var data T
		err := binding.Bind(c, &data)
		if err != nil {
			c.JSON(http.StatusOK, &errcode2.ErrRep{Code: errcode2.ErrCode(errcode.InvalidArgument), Msg: err.Error()})
			return
		}
		if err := db.Save(&data).Error; err != nil {
			c.JSON(http.StatusOK, &errcode2.ErrRep{Code: errcode2.ErrCode(errcode.DBError), Msg: err.Error()})
			return
		}
		c.JSON(http.StatusOK, httpi.ResponseOk)
	})
	url := "/api/v1/" + typ
	server.POST(url, cu...)
	Log(http.MethodPost, url, "create "+typ)
	url = "/api/v1/" + typ
	server.PUT(url, cu...)
	Log(http.MethodPut, url, "update "+typ)
	url = "/api/v1/" + typ + "/edit"
	server.POST(url, cu...)
	Log(http.MethodPost, url, "update "+typ)
	url = "/api/v1/" + typ + "/:" + keyParam
	server.PUT(url, append(middleware, func(c *gin.Context) {
		var data T
		err := binding.Bind(c, &data)
		if err != nil {
			c.JSON(http.StatusOK, &errcode2.ErrRep{Code: errcode2.ErrCode(errcode.InvalidArgument), Msg: err.Error()})
			return
		}
		if err = db.Where(keyParam+" =", c.Param(keyParam)).Updates(&data).Error; err != nil {
			c.JSON(http.StatusOK, &errcode2.ErrRep{Code: errcode2.ErrCode(errcode.DBError), Msg: err.Error()})
			return
		}
		c.JSON(http.StatusOK, httpi.ResponseOk)
	})...)
	Log(http.MethodPut, url, "update "+typ)
}

func Delete[T any](server *gin.Engine, db *gorm.DB, keyParam string, middleware ...gin.HandlerFunc) {
	var v T
	typ := stringsi.LowerCaseFirst(reflect.TypeOf(&v).Elem().Name())
	url := "/api/v1/" + typ + "/:" + keyParam
	server.DELETE(url, append(middleware, func(c *gin.Context) {
		if err := db.Delete(&v, c.Param(keyParam)).Error; err != nil {
			c.JSON(http.StatusOK, &errcode2.ErrRep{Code: errcode2.ErrCode(errcode.DBError), Msg: err.Error()})
			return
		}
		c.JSON(http.StatusOK, httpi.ResponseOk)
	})...)
	Log(http.MethodDelete, url, "delete "+typ)

	handler := append(middleware, func(c *gin.Context) {
		var m map[string]any
		if err := c.ShouldBindJSON(&m); err != nil {
			c.JSON(http.StatusOK, &errcode2.ErrRep{Code: errcode2.ErrCode(errcode.InvalidArgument), Msg: err.Error()})
			return
		}
		if err := db.Delete(&v, m[keyParam]).Error; err != nil {
			c.JSON(http.StatusOK, &errcode2.ErrRep{Code: errcode2.ErrCode(errcode.DBError), Msg: err.Error()})
			return
		}
		c.JSON(http.StatusOK, httpi.ResponseOk)
	})
	url = "/api/v1/" + typ
	server.DELETE(url, handler...)
	Log(http.MethodDelete, url, "delete "+typ)
	url = "/api/v1/" + typ + "/delete"
	server.POST(url, handler...)
	Log(http.MethodPost, url, "delete "+typ)
}

func Query[T any](server *gin.Engine, db *gorm.DB, keyParam string, middleware ...gin.HandlerFunc) {
	var v T
	typ := stringsi.LowerCaseFirst(reflect.TypeOf(&v).Elem().Name())
	url := "/api/v1/" + typ + "/:" + keyParam
	server.GET(url, append(middleware, func(c *gin.Context) {
		var data T
		if err := db.First(&data, c.Param(keyParam)).Error; err != nil {
			c.JSON(http.StatusOK, &errcode2.ErrRep{Code: errcode2.ErrCode(errcode.DBError), Msg: err.Error()})
			return
		}
		c.JSON(http.StatusOK, httpi.NewSuccessResData(data))
	})...)
	Log(http.MethodGet, url, "get "+typ)
}

func Log(method, path, title string) {
	log.Printf(" %s\t %s %s\t %s",
		style.Green("API:"),
		style.Yellow(stringsi.FormatLen(method, 6)),
		style.Blue(stringsi.FormatLen(path, 50)), style.Magenta(title))
}
