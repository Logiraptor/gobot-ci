package main

import (
	"fmt"
	"strconv"

	"gobot.io/x/gobot"
	"gobot.io/x/gobot/api"
	"gobot.io/x/gobot/platforms/sphero/bb8"
)

var bb8Address = "BB-E186"

func main() {
	bleAdaptor := NewClientAdaptor(bb8Address)
	bb8 := bb8.NewDriver(bleAdaptor)

	work := func() {}

	robot := gobot.NewRobot("bbBot",
		[]gobot.Connection{bleAdaptor},
		[]gobot.Device{bb8},
		work,
	)

	m := gobot.NewMaster()
	m.AddRobot(robot)

	api.NewAPI(m).Start()

	robot.AddCommand("set_color", func(params map[string]interface{}) interface{} {
		r, err := strconv.Atoi(params["r"].(string))
		if err != nil {
			fmt.Println("Error parsing r", err)
			return err
		}

		g, err := strconv.Atoi(params["g"].(string))
		if err != nil {
			fmt.Println("Error parsing g", err)
			return err
		}

		b, err := strconv.Atoi(params["b"].(string))
		if err != nil {
			fmt.Println("Error parsing b", err)
			return err
		}

		fmt.Println("Setting color to", r, g, b)
		bb8.SetRGB(uint8(r), uint8(g), uint8(b))

		return true
	})

	err := m.Start()
	fmt.Println(err)
}
