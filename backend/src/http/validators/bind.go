package validators

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func BindJSONWithErrors(c *gin.Context, obj any) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			out := make(map[string]string)
			for _, fe := range ve {
				out[fe.Field()] = fe.Tag()
			}
			c.JSON(400, gin.H{"errors": out})
		} else {
			c.JSON(400, gin.H{"error": err.Error()})
		}
		return false
	}
	return true
}
