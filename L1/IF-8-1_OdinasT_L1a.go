package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type Car struct {
	Make    string  `json:"make"`
	Year    int     `json:"year"`
	Mileage float64 `json:"mileage"`
}

func main() {
	cars := readData("IF-8-1_OdinasT_L1_dat_1.json")

	fmt.Println(cars)
}

func readData(path string) []Car{
	data, err := ioutil.ReadFile(path)

	if err != nil {
		panic(fmt.Sprintf("error when reading file %s", path))
	}

	var cars []Car
	err = json.Unmarshal(data, &cars)

	if err != nil {
		panic(fmt.Sprintf("error converting json in file %s", path))
	}

	return cars
}