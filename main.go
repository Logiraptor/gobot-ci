package main

import (
	"encoding/json"
	"log"
	"net/http"
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
		R:        255,
		G:        0,
		B:        0,
	})
	p.Push(Interval{
		Duration: time.Second,
		R:        0,
		G:        255,
		B:        0,
	})
	p.Push(Interval{
		Duration: time.Second,
		R:        0,
		G:        0,
		B:        255,
	})

	http.ListenAndServe(":3000", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("Got request", r.Method, r.URL.Path)

		var i Interval
		err := json.NewDecoder(r.Body).Decode(&i)
		if err != nil {
			log.Println("Error parsing request", err)
			http.Error(w, "Error parsing request", http.StatusBadRequest)
			return
		}

		if i.Duration == 0 {
			log.Println("defaulting to 1 second")
			i.Duration = time.Second
		}

		log.Println("pushing", i)
		p.Push(i)
	}))

	for {
		currentColor, dur, last := p.Pop()
		log.Println("popped", currentColor, dur, last)
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

	c.abs.SetRGB(currentColor.Red, currentColor.Green, currentColor.Blue)

	for {
		select {
		case color := <-c.colors:
			currentColor = color
			c.abs.SetRGB(currentColor.Red, currentColor.Green, currentColor.Blue)

			if currentColor == (Color{}) {
				if timeout == nil {
					timeout = time.After(30 * time.Second)
				}
			} else {
				timeout = nil
			}
		case <-ticker.C:
			c.abs.SetRGB(currentColor.Red, currentColor.Green, currentColor.Blue)
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
	Duration time.Duration `json:"duration"`
	R        uint8         `json:"r"`
	G        uint8         `json:"g"`
	B        uint8         `json:"b"`
}

type Color struct {
	Red   uint8
	Green uint8
	Blue  uint8
}

func (p *Plan) Push(interval Interval) {
	p.Intervals <- interval
}

func (p *Plan) Pop() (Color, time.Duration, bool) {
	interval := <-p.Intervals
	c := Color{
		Red:   interval.R,
		Green: interval.G,
		Blue:  interval.B,
	}
	return c, interval.Duration, len(p.Intervals) == 0
}
