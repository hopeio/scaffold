package crud

import (
	"github.com/gin-gonic/gin"
	"github.com/hopeio/utils/types/model"
	"gorm.io/gorm"
)

func Enum(server *gin.Engine, db *gorm.DB) {
	CRUD[model.Enum](server, db, "id")
	CRUD[model.EnumValue](server, db, "id")
}
