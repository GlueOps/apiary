package estate

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/app"
)

func RegisterUserProtected(hapi huma.API, deps *app.Deps) {
	Register(hapi, deps)
}
