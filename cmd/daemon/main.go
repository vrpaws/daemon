package main

import (
	"vrc-moments/cmd/daemon/app"
	"vrc-moments/cmd/daemon/components/settings"
	lib "vrc-moments/pkg"
)

func main() {
	model := app.NewModel("~/Pictures/VRChat/***.png")
	program := model.Run()
	defer lib.LogOutput(&model)()

	program.Send(settings.UsernameSet("Username"))
	program.Wait()
}
