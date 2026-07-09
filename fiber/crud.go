package fiber

import (
	"log"
	"net/http"
	"reflect"

	"github.com/gofiber/fiber/v3"
	sqlx "github.com/hopeio/gox/database/sql"
	gormx "github.com/hopeio/gox/database/sql/gorm"
	stringsx "github.com/hopeio/gox/strings"
	"github.com/hopeio/gox/terminal/style"
	responsex "github.com/hopeio/gox/types/response"
	gateway "github.com/hopeio/mix/contrib/fiber"
	"github.com/hopeio/mix"
	response "github.com/hopeio/protobuf/response"
	"github.com/hopeio/scaffold/errcode"
	"gorm.io/gorm"
)

const apiPrefix = "/api/"

func registerRoute(route fiber.Router, method, path string, handlers []fiber.Handler) {
	args := make([]any, len(handlers))
	for i, h := range handlers {
		args[i] = h
	}
	route.Add([]string{method}, path, args[0], args[1:]...)
}

func CRUD[T any](server *fiber.App, db *gorm.DB, middleware ...fiber.Handler) {
	Save[T](server, db, middleware...)
	Query[T](server, db, middleware...)
	Delete[T](server, db, middleware...)
}

func Save[T any](server *fiber.App, db *gorm.DB, middleware ...fiber.Handler) {
	var v T
	typ := stringsx.LowerCaseFirst(reflect.TypeOf(&v).Elem().Name())
	cu := append(middleware, func(c fiber.Ctx) error {
		var data T
		err := gateway.Bind(c, &data)
		if err != nil {
			gateway.Respond(c, &mix.ErrResp{Code: mix.ErrCode(mix.InvalidArgument), Msg: err.Error()})
			return nil
		}
		if err := db.Save(&data).Error; err != nil {
			gateway.Respond(c, &mix.ErrResp{Code: mix.ErrCode(errcode.DBError), Msg: err.Error()})
			return nil
		}
		gateway.Respond(c, nil)
		return nil
	})
	url := apiPrefix + typ
	registerRoute(server, http.MethodPost, url, cu)
	Log(http.MethodPost, url, "create "+typ)
	registerRoute(server, http.MethodPut, url, cu)
	Log(http.MethodPut, url, "update "+typ)
	registerRoute(server, http.MethodPost, url, cu)
	Log(http.MethodPost, url, "update "+typ)
	url = apiPrefix + typ + "/:id"
	registerRoute(server, http.MethodPut, url, append(middleware, func(c fiber.Ctx) error {
		var data T
		err := gateway.Bind(c, &data)
		if err != nil {
			gateway.Respond(c, &mix.ErrResp{Code: mix.ErrCode(mix.InvalidArgument), Msg: err.Error()})
			return nil
		}
		if err = db.Clauses(gormx.ByPrimaryKey(c.Params("id"))).Updates(&data).Error; err != nil {
			gateway.Respond(c, &mix.ErrResp{Code: mix.ErrCode(errcode.DBError), Msg: err.Error()})
			return nil
		}
		gateway.Respond(c, nil)
		return nil
	}))
	Log(http.MethodPut, url, "update "+typ)
}

func Delete[T any](server *fiber.App, db *gorm.DB, middleware ...fiber.Handler) {
	var v T
	typ := stringsx.LowerCaseFirst(reflect.TypeOf(&v).Elem().Name())
	url := apiPrefix + typ + "/:id"
	registerRoute(server, http.MethodDelete, url, append(middleware, func(c fiber.Ctx) error {
		if err := db.Delete(&v, c.Params("id")).Error; err != nil {
			gateway.Respond(c, &mix.ErrResp{Code: mix.ErrCode(errcode.DBError), Msg: err.Error()})
			return nil
		}
		gateway.Respond(c, nil)
		return nil
	}))
	Log(http.MethodDelete, url, "delete "+typ)

	handler := append(middleware, func(c fiber.Ctx) error {
		var m map[string]any
		if err := c.Bind().JSON(&m); err != nil {
			gateway.Respond(c, &mix.ErrResp{Code: mix.ErrCode(mix.InvalidArgument), Msg: err.Error()})
			return nil
		}
		if err := db.Delete(&v, m["id"]).Error; err != nil {
			gateway.Respond(c, &mix.ErrResp{Code: mix.ErrCode(errcode.DBError), Msg: err.Error()})
			return nil
		}
		gateway.Respond(c, nil)
		return nil
	})
	url = apiPrefix + typ
	registerRoute(server, http.MethodDelete, url, handler)
	Log(http.MethodDelete, url, "delete "+typ)
	url = apiPrefix + typ + "/delete"
	registerRoute(server, http.MethodPost, url, handler)
	Log(http.MethodPost, url, "delete "+typ)
}

func Query[T any](server *fiber.App, db *gorm.DB, middleware ...fiber.Handler) {
	var v T
	typ := stringsx.LowerCaseFirst(reflect.TypeOf(&v).Elem().Name())
	url := apiPrefix + typ + "/:id"
	registerRoute(server, http.MethodGet, url, append(middleware, func(c fiber.Ctx) error {
		var data T
		if err := db.First(&data, c.Params("id")).Error; err != nil {
			gateway.Respond(c, &mix.ErrResp{Code: mix.ErrCode(errcode.DBError), Msg: err.Error()})
			return nil
		}
		gateway.Respond(c, data)
		return nil
	}))
	Log(http.MethodGet, url, "get "+typ)
}

func List[T any](server *fiber.App, db *gorm.DB, middleware ...fiber.Handler) {
	var v T
	typ := stringsx.LowerCaseFirst(reflect.TypeOf(&v).Elem().Name())
	url := apiPrefix + typ
	registerRoute(server, http.MethodGet, url, append(middleware, func(c fiber.Ctx) error {
		var count int64
		var req sqlx.List
		err := gateway.Bind(c, &req)
		if err != nil {
			gateway.Respond(c, &response.ErrResp{Code: int32(mix.InvalidArgument), Msg: err.Error()})
			return nil
		}
		var list []*T
		if clause := gormx.PaginationExpr(req.Pagination.No, req.Pagination.Size); clause != nil {
			db = db.Clauses(clause)
		}
		if clause := gormx.SortExpr(nil, req.Sort...); clause != nil {
			db = db.Clauses(clause)
		}
		if err = db.Count(&count).Error; err != nil {
			gateway.Respond(c, &response.ErrResp{Code: int32(errcode.DBError), Msg: err.Error()})
			return nil
		}
		if err = db.Find(&list).Error; err != nil {
			gateway.Respond(c, &response.ErrResp{Code: int32(errcode.DBError), Msg: err.Error()})
			return nil
		}
		if count == 0 {
			count = int64(len(list))
		}
		gateway.Respond(c, &responsex.List[*T]{List: list, Total: uint(count)})
		return nil
	}))
	Log(http.MethodGet, url, "get "+typ)
}

func Log(method, path, title string) {
	log.Printf(" %s\t %s %s\t %s",
		style.Green("API:"),
		style.Yellow(stringsx.FormatLen(method, 6)),
		style.Blue(stringsx.FormatLen(path, 50)), style.Magenta(title))
}
