package main

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/sync/semaphore"
	"io/ioutil"
	"sync"
	"time"
)

// todo change to 30
const DataCount = 10    // How much cars in json file
const RoutineCount = 10 // How much worker routines to start
const BufferSize = DataCount / 2

type (
	Car struct {
		Make    string  `json:"make"`
		Year    int     `json:"year"`
		Mileage float64 `json:"mileage"`
	}

	CarWithAge struct {
		Car Car
		Age int
	}

	DataMonitor struct {
		Cars                  [BufferSize]Car
		In, Out               int
		Count, Space          *semaphore.Weighted
		InputLock, OutputLock sync.Mutex
		Context               context.Context // needed for semaphore
	}

	SortedResultMonitor struct {
		Cars  [DataCount]CarWithAge
		Count int
		Lock  sync.Mutex
	}
)

func NewDataMonitor() *DataMonitor {
	ctx := context.TODO()
	monitor := DataMonitor{
		Context: ctx,
		Count:   semaphore.NewWeighted(BufferSize),
		Space:   semaphore.NewWeighted(BufferSize),
	}
	_ = monitor.Count.Acquire(ctx, BufferSize)
	return &monitor
}

func (m *DataMonitor) addItem(car Car) {
	_ = m.Space.Acquire(m.Context, 1)
	m.InputLock.Lock()
	m.Cars[m.In] = car
	m.In = (m.In + 1) % BufferSize
	m.InputLock.Unlock()
	m.Count.Release(1)
}

func (m *DataMonitor) removeItem() Car {
	_ = m.Count.Acquire(m.Context, 1)
	m.OutputLock.Lock()
	car := m.Cars[m.Out]

	if car.Make == "<EndOfInput>" { // Special message to close consumers
		m.OutputLock.Unlock()
		m.Count.Release(1)
		return car
	}

	m.Out = (m.Out + 1) % BufferSize
	m.OutputLock.Unlock()
	m.Space.Release(1)
	return car
}

func main() {
	dataMonitor := NewDataMonitor()
	var resultMonitor SortedResultMonitor
	var wg sync.WaitGroup
	wg.Add(RoutineCount)

	// 1
	cars := readData("IF-8-1_OdinasT_L1_dat_1.json")

	// 2
	for i := 0; i < RoutineCount; i++ {
		go func() {
			worker(dataMonitor, &resultMonitor)
			wg.Done()
		}()
	}

	// 3
	for _, car := range cars {
		dataMonitor.addItem(car)
	}
	dataMonitor.addItem(Car{Make: "<EndOfInput>"})

	// 4
	wg.Wait()

	// 5
	writeData("IF-8-1_OdinasT_L1_rez.txt", resultMonitor)

}

func worker(in *DataMonitor, out *SortedResultMonitor) {
	for {
		car := in.removeItem() // Will block if no data
		if car.Make == "<EndOfInput>" {
			break
		}
		carAge := time.Now().Year() - car.Year + int(car.Mileage/20_000)
		if carAge < 50 { // todo change
			car := CarWithAge{car, carAge}
			out.addItemSorted(car)
		}
	}
}

func (m *SortedResultMonitor) addItemSorted(car CarWithAge) {
	m.Lock.Lock()
	i := m.Count - 1
	for i >= 0 && m.Cars[i].Age > car.Age {
		m.Cars[i+1] = m.Cars[i]
		i--
	}
	m.Cars[i+1] = car
	m.Count++
	m.Lock.Unlock()
}

func (m *SortedResultMonitor) getItems() {

}

func readData(path string) [DataCount]Car {
	data, _ := ioutil.ReadFile(path)
	var cars [DataCount]Car
	_ = json.Unmarshal(data, &cars)
	return cars
}

func writeData(path string, monitor SortedResultMonitor) {
	for _, car := range monitor.Cars {
		fmt.Println(car)
	}
}
