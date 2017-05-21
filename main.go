package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"gobot.io/x/gobot"
	"gobot.io/x/gobot/platforms/ble"
	"gobot.io/x/gobot/platforms/sphero/bb8"
)

//go:generate stringer -type Status

func main() {
	bleAdaptor := ble.NewClientAdaptor("BB-E186")
	bb8 := bb8.NewDriver(bleAdaptor)

	work := func() {
		gobot.Every(5*time.Second, func() {
			s := checkStatus()
			r, g, b := statusToColor(s)
			bb8.SetRGB(r, g, b)
		})
	}

	robot := gobot.NewRobot("bbBot",
		[]gobot.Connection{bleAdaptor},
		[]gobot.Device{bb8},
		work,
	)

	err := robot.Start()
	fmt.Println(err)
}

func statusToColor(s Status) (r, g, b uint8) {
	switch s {
	case Failed:
		return 255, 0, 0
	case Error:
		return 255, 255, 0
	case Success:
		return 0, 255, 0
	case InProgress:
		return 255, 0, 255
	}
	return 0, 255, 255
}

type Status int

const (
	Success Status = iota
	InProgress
	Failed
	Error
)

type BuildStatus struct {
	ID                  int         `json:"id"`
	Slug                string      `json:"slug"`
	Description         interface{} `json:"description"`
	PublicKey           string      `json:"public_key"`
	LastBuildID         int         `json:"last_build_id"`
	LastBuildNumber     string      `json:"last_build_number"`
	LastBuildStatus     *int        `json:"last_build_status"`
	LastBuildResult     interface{} `json:"last_build_result"`
	LastBuildDuration   interface{} `json:"last_build_duration"`
	LastBuildLanguage   interface{} `json:"last_build_language"`
	LastBuildStartedAt  time.Time   `json:"last_build_started_at"`
	LastBuildFinishedAt interface{} `json:"last_build_finished_at"`
	Active              bool        `json:"active"`
}

func checkStatus() Status {
	resp, err := http.Get("https://api.travis-ci.org/repositories/Logiraptor/elang.json")
	if err != nil {
		fmt.Println(err)
		return Error
	}
	defer resp.Body.Close()
	var status BuildStatus
	err = json.NewDecoder(resp.Body).Decode(&status)
	if err != nil {
		fmt.Println(err)
		return Error
	}

	switch {
	case status.LastBuildStatus == nil:
		return InProgress
	case *status.LastBuildStatus == 0:
		return Success
	case *status.LastBuildStatus != 0:
		return Error
	}
	fmt.Printf("Unknown status: %+v", status)
	return Error
}
