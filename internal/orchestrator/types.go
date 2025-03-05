package orchestrator

import "time"

type Expression struct {
	ID      string
	Status  string
	Result  float64
	TaskIDs []int
	DepMap  map[int][]int
}

type Node struct {
	Valuse    []string
	Operators []string
}

type Task struct {
	ID           int
	Arg1         interface{}
	Arg2         interface{}
	Operation    string
	Duration     time.Duration
	Result       float64
	Status       string
	Dependencies []int
	Index        int
}
