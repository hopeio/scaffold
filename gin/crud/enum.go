package crud

import (
	"github.com/gin-gonic/gin"
	"github.com/hopeio/gox/types/model"
	modelpb "github.com/hopeio/protobuf/model"
	"gorm.io/gorm"
)

func Enum(server *gin.Engine, db *gorm.DB) {
	CRUD[model.Enum](server, db)
	CRUD[model.EnumValue](server, db)
}

func EnumPB(server *gin.Engine, db *gorm.DB) {
	CRUD[modelpb.Enum](server, db)
	CRUD[modelpb.EnumValue](server, db)
}
