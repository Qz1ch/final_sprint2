package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Task struct {
	ID           int
	Arg1         float64
	Arg2         float64
	Operation    string
	Duration     time.Duration
	Result       float64
	Status       string
	Dependencies []int
}

func WorkSim(task *Task) float64 {
	time.Sleep(task.Duration)
	switch task.Operation {
	case "+":
		return task.Arg1 + task.Arg2
	case "-":
		return task.Arg1 - task.Arg2
	case "*":
		return task.Arg1 * task.Arg2
	case "/":
		if task.Arg2 == 0 {
			panic("division by zero")
		}
		return task.Arg1 / task.Arg2
	default:
		panic("unknown operator")
	}
}

func AgentWorker(computingPower int) {
	for i := 0; i < computingPower; i++ {
		go func() {
			for {
				task := fetchTask()
				fmt.Println("go run 1104", task)
				if task == nil {
					time.Sleep(1 * time.Second)
					continue
				}
				result := WorkSim(task)
				fmt.Println("go run 1103", result, task)
				sendResult(task.ID, result)
			}
		}()
	}
}

func fetchTask() *Task {
	resp, err := http.Get("http://localhost:8080/task")
	if err != nil {
		log.Println("Failed to fetch task:", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}

	var taskResp map[string]Task
	if err := json.NewDecoder(resp.Body).Decode(&taskResp); err != nil {
		log.Println("Failed to decode task:", err)
		return nil
	}
	taskCopy := taskResp["task"]
	return &taskCopy
}

func sendResult(taskID int, result float64) {
	data := map[string]interface{}{
		"id":     taskID,
		"result": result,
	}

	jsonData, _ := json.Marshal(data)
	resp, err := http.Post("http://localhost:8080/task/result", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Println("Failed to send result:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Println("Failed to send result, status:", resp.Status)
	}
}
