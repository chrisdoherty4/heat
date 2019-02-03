package main

import (
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/gizak/termui"
	"github.com/gizak/termui/widgets"
)

type Node struct {
	widgets.Paragraph
	Name              string
	Temperature       int
	Rate              float64
	Input             chan int
	Output            []chan int
	InputTemperatures [4]int
	Index             uint8
	mutex             *sync.Mutex
}

func NewNode(name string) *Node {
	node := &Node{
		Name:              name,
		Paragraph:         *widgets.NewParagraph(),
		Temperature:       0,
		Rate:              0.05,
		Input:             make(chan int),
		Output:            make([]chan int, 4),
		InputTemperatures: *new([4]int),
		Index:             0,
		mutex:             &sync.Mutex{},
	}

	node.Paragraph.SetRect(0, 0, 25, 25)
	node.Paragraph.Text = "0"

	return node
}

func (self *Node) AverageInputTemperature() int {
	var sum int

	self.mutex.Lock()
	for t := range self.InputTemperatures {
		sum += t
	}
	self.mutex.Unlock()

	return sum / len(self.InputTemperatures)
}

func (self *Node) PushTemperature(temperature int) {
	self.mutex.Lock()
	self.InputTemperatures[self.Index] = temperature
	self.mutex.Unlock()
	self.Index = (self.Index + 1) % uint8(len(self.InputTemperatures))
}

func (self *Node) Start() {
	go func() {
		for {
			select {
			case t := <-self.Input:
				self.PushTemperature(t)
				fmt.Printf("%v %v", self.Name, self.InputTemperatures)
			}
		}
	}()

	go func() {
		duration, _ := time.ParseDuration("1000ms")
		tick := time.NewTicker(duration).C
		const rate int = 2
		rateSquared := math.Pow(float64(rate), 2)
		for {
			select {
			case <-tick:
				delta := self.Temperature - self.AverageInputTemperature()
				if math.Pow(float64(delta), 2) > rateSquared {
					if delta < 0 {
						self.Temperature -= rate
					} else {
						self.Temperature += rate
					}
					//self.Temperature += int(float64(self.AverageInputTemperature()-self.Temperature) * self.Rate)
					self.UpdateDisplay()
					self.Notify()
				}
			}
		}
	}()
}

func (self *Node) Notify() {
	for _, ch := range self.Output {
		if ch != nil {
			ch <- self.Temperature
		}
	}
}

func (self *Node) UpdateDisplay() {
	self.Paragraph.Text = fmt.Sprintf("%v: %v (average: %v)", self.Name, self.Temperature, self.AverageInputTemperature())
}

func (self *Node) OutputTo(node *Node) {
	self.Output = append(self.Output, node.Input)
}

type Grid struct {
	termui.Grid
	Nodes [][]*Node
}

func NewGrid(rows, cols uint) *Grid {
	grid := &Grid{
		Grid:  *termui.NewGrid(),
		Nodes: make([][]*Node, rows),
	}

	w, h := termui.TerminalDimensions()
	grid.SetRect(0, 0, w, h)

	for i, _ := range grid.Nodes {
		grid.Nodes[i] = make([]*Node, cols)
	}

	return grid
}

func main() {
	if err := termui.Init(); err != nil {
		log.Fatalf("Failed to initialize termui")
	}

	defer termui.Close()

	input := NewNode("Input")
	node1 := NewNode("Node 1")
	input.OutputTo(node1)

	input.Start()
	node1.Start()

	grid := NewGrid(1, 2)
	grid.Grid.Set(termui.NewRow(1, termui.NewCol(1.0/2, node1)))

	termui.Render(grid)

	events := termui.PollEvents()
	duration, _ := time.ParseDuration("100ms")
	ticks := time.NewTicker(duration).C
	for {
		select {
		case e := <-events:
			switch e.ID {
			case "q":
				return
			case "<Up>":
				input.Temperature += 11
			case "<Down>":
				input.Temperature -= 11
			}
			input.Notify()
		case <-ticks:
			termui.Render(grid)
		}
	}

}
