package main

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"gobot.io/x/gobot"
	"gobot.io/x/gobot/platforms/sphero/bb8"
)

var bb8Address = "BB-E186"

func main() {
	adp := NewGobotAdapter()
	worker := NewBgConn(adp)
	go worker.worker()

	p := NewPlan()
	p.Push(Interval{
		Duration: time.Second,
		Color:    Color{Red: 255, Green: 0, Blue: 0},
	})
	p.Push(Interval{
		Duration: time.Second,
		Color:    Color{Red: 0, Green: 255, Blue: 0},
	})
	p.Push(Interval{
		Duration: time.Second,
		Color:    Color{Red: 0, Green: 0, Blue: 255},
	})

	http.ListenAndServe(":3000", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("Got request", r.Method, r.URL.Path, r.URL.RawQuery)
		red, err := strconv.Atoi(r.FormValue("r"))
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		green, err := strconv.Atoi(r.FormValue("g"))
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		blue, err := strconv.Atoi(r.FormValue("b"))
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		duration, err := time.ParseDuration(r.FormValue("duration"))
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if duration == 0 {
			log.Println("defaulting to 1 second")
			duration = time.Second
		}

		log.Println("pushing", red, green, blue, duration)
		p.Push(Interval{
			Duration: duration,
			Color:    Color{Red: (red), Green: (green), Blue: (blue)},
		})
	}))

	for {
		currentColor, dur, last := p.Pop()
		worker.colors <- currentColor
		<-time.After(dur)

		if last {
			worker.colors <- Color{}
		}
	}
}

type bb8Abstraction interface {
	Start()
	Stop()
	SetRGB(r, g, b uint8)
}

func NewGobotAdapter() *gobotAdapter {
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
	m.AutoRun = false

	return &gobotAdapter{
		m:      m,
		driver: bb8,
	}
}

type gobotAdapter struct {
	m      *gobot.Master
	driver *bb8.BB8Driver
}

func (x *gobotAdapter) Start() {
	x.m.Start()
}

func (x *gobotAdapter) Stop() {
	x.m.Stop()
}

func (x *gobotAdapter) SetRGB(r, g, b uint8) {
	x.driver.SetRGB(r, g, b)
}

type bgconn struct {
	colors chan Color
	abs    bb8Abstraction
}

func NewBgConn(abs bb8Abstraction) *bgconn {
	return &bgconn{
		colors: make(chan Color),
		abs:    abs,
	}
}

func (c *bgconn) worker() {
	for color := range c.colors {
		c.liveLoop(color)
	}
}

func (c *bgconn) liveLoop(startingColor Color) {
	c.abs.Start()
	defer c.abs.Stop()

	ticker := time.NewTicker(1 * time.Minute)
	var timeout <-chan time.Time

	currentColor := startingColor

	c.abs.SetRGB(uint8(currentColor.Red), uint8(currentColor.Green), uint8(currentColor.Blue))

	for {
		select {
		case color := <-c.colors:
			currentColor = color
			c.abs.SetRGB(uint8(currentColor.Red), uint8(currentColor.Green), uint8(currentColor.Blue))

			if currentColor == (Color{}) {
				if timeout == nil {
					timeout = time.After(30 * time.Second)
				}
			} else {
				timeout = nil
			}
		case <-ticker.C:
			c.abs.SetRGB(uint8(currentColor.Red), uint8(currentColor.Green), uint8(currentColor.Blue))
		case <-timeout:
			return
		}
	}
}

func iterate() {
}

type Plan struct {
	Intervals chan Interval
}

func NewPlan() *Plan {
	return &Plan{
		Intervals: make(chan Interval, 100),
	}
}

type Interval struct {
	Duration time.Duration
	Color    Color
}

type Color struct {
	Red   int
	Green int
	Blue  int
}

func (p *Plan) Push(interval Interval) {
	p.Intervals <- interval
}

func (p *Plan) Pop() (Color, time.Duration, bool) {
	interval := <-p.Intervals
	return interval.Color, interval.Duration, len(p.Intervals) == 0
}
