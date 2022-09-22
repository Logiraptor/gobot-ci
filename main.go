package main

import (
	"fmt"
	"strconv"

	"gobot.io/x/gobot"
	"gobot.io/x/gobot/api"
	"gobot.io/x/gobot/platforms/sphero/bb8"
	"gobot.io/x/gobot/platforms/sphero/ollie"
)

var bb8Address = "BB-E186"

func main() {
	bleAdaptor := NewClientAdaptor(bb8Address)
	bb8 := bb8.NewDriver(bleAdaptor)

	var color struct {
		Red   int
		Green int
		Blue  int
	}

	work := func() {
		// gobot.Every(1*time.Second, func() {
		// })
	}

	robot := gobot.NewRobot("bbBot",
		[]gobot.Connection{bleAdaptor},
		[]gobot.Device{bb8},
		work,
	)
	m := gobot.NewMaster()
	m.AddRobot(robot)
	api.NewAPI(m).Start()

	robot.AddCommand("set_color", func(params map[string]interface{}) interface{} {
		var err error
		color.Red, err = strconv.Atoi(params["r"].(string))
		if err != nil {
			fmt.Println("Error parsing r", err)
			return err
		}

		color.Green, err = strconv.Atoi(params["g"].(string))
		if err != nil {
			fmt.Println("Error parsing g", err)
			return err
		}

		color.Blue, err = strconv.Atoi(params["b"].(string))
		if err != nil {
			fmt.Println("Error parsing b", err)
			return err
		}

		fmt.Println("Setting color to", color.Red, color.Green, color.Blue)
		bb8.SetRGB(uint8(color.Red), uint8(color.Green), uint8(color.Blue))
		return true
	})

	bb8.On(ollie.Collision, func(s interface{}) {
		fmt.Printf("Collision detected: %v", s)
	})

	bb8.On(ollie.Error, func(s interface{}) {
		fmt.Printf("Error detected: %v", s)
	})

	bb8.On(ollie.SensorData, func(s interface{}) {
		fmt.Printf("Sensor Data: %v", s)
	})

	err := m.Start()
	fmt.Println(err)
}
