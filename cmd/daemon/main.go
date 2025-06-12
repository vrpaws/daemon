package main

import (
	"vrc-moments/cmd/daemon/app"
	lib "vrc-moments/pkg"
)

func main() {
	model := app.NewModel("~/Pictures/VRChat/***.png")
	program := model.Run()
	defer lib.LogOutput(&model)()

	program.Wait()
}
