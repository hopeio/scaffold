package crud

import (
	"github.com/gin-gonic/gin"
	"github.com/hopeio/protobuf/errcode"
	errcode2 "github.com/hopeio/utils/errors/errcode"
	httpi "github.com/hopeio/utils/net/http"
	"github.com/hopeio/utils/net/http/gin/binding"
	stringsi "github.com/hopeio/utils/strings"
	"gorm.io/gorm"
	"net/http"
	"reflect"
)

func CRUD[T any](server *gin.Engine, db *gorm.DB) {
	var v T
	typ := stringsi.LowerCaseFirst(reflect.TypeOf(&v).Elem().Name())
	server.GET("/api/v1/"+typ+"/:id", func(c *gin.Context) {
		var data T
		if err := db.First(&data, c.Param("id")).Error; err != nil {
			c.JSON(http.StatusOK, &errcode2.ErrRep{Code: errcode2.ErrCode(errcode.DBError), Msg: err.Error()})
			return
		}
		c.JSON(http.StatusOK, httpi.NewSuccessResData(data))
	})
	cu := func(c *gin.Context) {
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
	}
	server.POST("/api/v1/"+typ, cu)
	server.PUT("/api/v1/"+typ, cu)
	server.DELETE("/api/v1/"+typ+"/:id", func(c *gin.Context) {
		if err := db.Delete(&v, c.Param("id")).Error; err != nil {
			c.JSON(http.StatusOK, &errcode2.ErrRep{Code: errcode2.ErrCode(errcode.DBError), Msg: err.Error()})
			return
		}
		c.JSON(http.StatusOK, httpi.ResponseOk)
	})
}

func Delete[T any](server *gin.Engine, db *gorm.DB) {
	var v T
	typ := stringsi.LowerCaseFirst(reflect.TypeOf(&v).Elem().Name())
	server.DELETE("/api/v1/"+typ+"/:id", func(c *gin.Context) {
		if err := db.Delete(&v, c.Param("id")).Error; err != nil {
			c.JSON(http.StatusOK, &errcode2.ErrRep{Code: errcode2.ErrCode(errcode.DBError), Msg: err.Error()})
			return
		}
		c.JSON(http.StatusOK, httpi.ResponseOk)
	})
}

func Query[T any](server *gin.Engine, db *gorm.DB) {
	var v T
	typ := stringsi.LowerCaseFirst(reflect.TypeOf(&v).Elem().Name())
	server.GET("/api/v1/"+typ+"/:id", func(c *gin.Context) {
		var data T
		if err := db.First(&data, c.Param("id")).Error; err != nil {
			c.JSON(http.StatusOK, &errcode2.ErrRep{Code: errcode2.ErrCode(errcode.DBError), Msg: err.Error()})
			return
		}
		c.JSON(http.StatusOK, httpi.NewSuccessResData(data))
	})

}
