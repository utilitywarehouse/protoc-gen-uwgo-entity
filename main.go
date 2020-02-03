package main

import (
	pgs "github.com/lyft/protoc-gen-star"
	pgsgo "github.com/lyft/protoc-gen-star/lang/go"

	"github.com/utilitywarehouse/protoc-gen-uwgo-entity/internal/entity"
)

func main() {
	pgs.Init(
		pgs.DebugEnv("DEBUG"),
	).RegisterModule(
		entity.NewIdentifierModule(),
	).RegisterPostProcessor(
		pgsgo.GoFmt(),
	).Render()
}
