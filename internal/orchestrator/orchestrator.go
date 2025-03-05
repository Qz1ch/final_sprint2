package orchestrator

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
	"yand/pkg/add"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

var (
	expressions = make(map[string]*Expression)
	tasks       = make(map[int]Task)
	mu          sync.RWMutex
	taskID      int
	thisnode    Node
	IsResult    = false
	glavIndex   []int
	glavResult  []string
)

func checkTaskCompletion(taskID int) {
	mu.Lock()
	defer mu.Unlock()

	exprID := ""
	for exprID, expr := range expressions {
		for _, tid := range expr.TaskIDs {
			if tid == taskID {
				break
			}
		}
		if exprID != "" {
			break
		}
	}

	if exprID == "" {
		return
	}

	expr := expressions[exprID]
	allCompleted := true
	for _, tid := range expr.TaskIDs {
		if tasks[tid].Status != "completed" {
			allCompleted = false
			break
		}
	}

	if allCompleted {
		finalResult := calculateFinalResult(expr.TaskIDs, expr.DepMap)
		expr.Status = "completed"
		expr.Result = finalResult
		expressions[exprID] = expr
	}
}

func getOperationDuration(op string) time.Duration {
	switch op {
	case "+", "-":
		return time.Duration(getEnvInt("TIME_ADDITION_MS", 3000)) * time.Millisecond
	case "*":
		return time.Duration(getEnvInt("TIME_MULTIPLICATIONS_MS", 4000)) * time.Millisecond
	case "/":
		return time.Duration(getEnvInt("TIME_DIVISIONS_MS", 5000)) * time.Millisecond
	default:
		return 0
	}
}

func getEnvInt(key string, defaultValue int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	v, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return v
}
func HandleCalculate(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received expression: %s", r.Body)

	var req struct {
		Expression string `json:"expression"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusUnprocessableEntity)
		return
	}

	mu.Lock()
	defer mu.Unlock()
	id := uuid.New().String()
	expr := &Expression{
		ID:      id,
		Status:  "pending",
		TaskIDs: []int{},
		DepMap:  make(map[int][]int),
	}
	expressions[id] = expr
	go func() {
		values, ops, err := add.ParseExpression(req.Expression)

		if err != nil {
			log.Printf("Error building AST: %v", err)
			http.Error(w, "Invalid expression", http.StatusUnprocessableEntity)
			return
		}
		var fulltasks []Task
		fulldepmap := make(map[int][]int)
		for i, n := range values {
			if n[0] == '(' {
				for len(thisnode.Valuse) != 1 {
					values1, ops1, err := add.ParseExpression(n[1 : len(n)-1])
					if err != nil {
						log.Printf("Error building AST: %v", err)
						http.Error(w, "Invalid expression", http.StatusUnprocessableEntity)
						return
					}
					thisnode = Node{
						Valuse:    values1,
						Operators: ops1,
					}

					tasksList1, depMap1, err := createTasksFromAST(fulldepmap, fulltasks)
					fulldepmap = depMap1
					fulltasks = tasksList1
					for _, t := range fulltasks {
						tasks[len(tasks)] = t
					}
					for !IsResult {
						time.Sleep(time.Second)
					}
					for _, n := range tasks {
						n.Status = "completed"
					}
					IsResult = false
				}
				glavIndex = append(glavIndex, i)
				glavResult = append(glavResult, thisnode.Valuse[0])

			}
		}
		thisnode = Node{}
		proshI := 0
		for i, n := range glavIndex {
			don := append(values[proshI:n], glavResult[i])
			thisnode.Valuse = append(thisnode.Valuse, don...)
			proshI = n + 1
		}
		thisnode = Node{
			Valuse:    thisnode.Valuse,
			Operators: ops,
		}
		for len(thisnode.Valuse) != 1 {
			tasksList, depMap, err := createTasksFromAST(fulldepmap, fulltasks)
			if err != nil {
				log.Printf("Error building AST: %v", err)
				http.Error(w, "Invalid expression", http.StatusUnprocessableEntity)
				return
			}
			fulldepmap = depMap
			fulltasks = tasksList
			for _, t := range fulltasks {
				tasks[len(tasks)] = t
			}
			for !IsResult {
				time.Sleep(time.Second)
			}
			for _, n := range tasks {
				n.Status = "completed"
			}
			IsResult = false
		}
		res, err := strconv.ParseFloat(thisnode.Valuse[0], 64)
		if err != nil {
			log.Printf("Error parsing result: %v", err)
		}
		expr.Result = res
		if err != nil {
			http.Error(w, "Invalid expression", http.StatusUnprocessableEntity)
			return
		}

		for _, task := range tasks {
			taskID++
			tasks[taskID] = task
			expr.TaskIDs = append(expr.TaskIDs, taskID)
		}

		expr.DepMap = fulldepmap
		expr.Status = "completed"
		expressions[id] = expr
	}()
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": id})
}

func createTasksFromAST(depmap1 map[int][]int, tasks1 []Task) ([]Task, map[int][]int, error) {
	var tasks []Task
	values := thisnode.Valuse
	ops := thisnode.Operators
	depMap := make(map[int][]int)
	dm := make(map[int][]int)
	for i, op := range ops {
		if i != 0 && i != len(ops)-1 && len(ops) != 1 {
			if op == "+" || op == "-" {
				if ops[i-1] != "*" || ops[i+1] != "/" || ops[i+1] != "*" || ops[i-1] != "/" {
					value1, err := strconv.ParseFloat(values[i], 64)
					value2, err := strconv.ParseFloat(values[i+1], 64)
					if err != nil {
						return nil, nil, err
					}
					task := Task{
						Arg1:         value1,
						Arg2:         value2,
						Operation:    op,
						Duration:     getOperationDuration(op),
						Status:       "pending",
						Dependencies: []int{},
						ID:           taskID,
						Index:        i,
					}
					taskID++
					dm[len(depMap)] = append(dm[len(depMap)], taskID)
					tasks = append(tasks, task)
				}
			} else if op == "*" || op == "/" {
				if ops[i-1] != "/" || ops[i-1] != "*" {
					value1, err := strconv.ParseFloat(values[i], 64)
					value2, err := strconv.ParseFloat(values[i+1], 64)
					if err != nil {
						return nil, nil, err
					}
					task := Task{
						Arg1:         value1,
						Arg2:         value2,
						Operation:    op,
						Duration:     getOperationDuration(op),
						Status:       "pending",
						Dependencies: []int{},
						ID:           taskID,
						Index:        i,
					}
					taskID++
					dm[len(depMap)] = append(dm[len(depMap)], taskID)
					tasks = append(tasks, task)
				}
			}
		} else if len(ops) == 1 {
			value1, err := strconv.ParseFloat(values[i], 64)

			if err != nil {
				return nil, nil, err
			}
			value2, err := strconv.ParseFloat(values[i+1], 64)
			if err != nil {
				return nil, nil, err
			}
			task := Task{
				Arg1:         value1,
				Arg2:         value2,
				Operation:    op,
				Duration:     getOperationDuration(op),
				Status:       "pending",
				Dependencies: []int{},
				ID:           taskID,
				Index:        i,
			}
			taskID++
			dm[len(depMap)] = append(dm[len(depMap)], taskID)
			tasks = append(tasks, task)
		} else if i == 0 {
			if op == "+" || op == "-" {
				if ops[i+1] != "/" || ops[i+1] != "*" {
					value1, err := strconv.ParseFloat(values[i], 64)
					value2, err := strconv.ParseFloat(values[i+1], 64)
					if err != nil {
						return nil, nil, err
					}
					task := Task{
						Arg1:         value1,
						Arg2:         value2,
						Operation:    op,
						Duration:     getOperationDuration(op),
						Status:       "pending",
						Dependencies: []int{},
						ID:           taskID,
						Index:        i,
					}
					taskID++
					dm[len(depMap)] = append(dm[len(depMap)], taskID)
					tasks = append(tasks, task)
				}
			} else if op == "*" || op == "/" {
				value1, err := strconv.ParseFloat(values[i], 64)
				value2, err := strconv.ParseFloat(values[i+1], 64)
				if err != nil {
					return nil, nil, err
				}
				task := Task{
					Arg1:         value1,
					Arg2:         value2,
					Operation:    op,
					Duration:     getOperationDuration(op),
					Status:       "pending",
					Dependencies: []int{},
					ID:           taskID,
					Index:        i,
				}
				taskID++
				dm[len(depMap)] = append(dm[len(depMap)], taskID)
				tasks = append(tasks, task)
			}
		} else if i == len(ops)-1 {
			if op == "+" || op == "-" {
				if ops[i-1] != "*" || ops[i-1] != "/" {
					value1, err := strconv.ParseFloat(values[i], 64)
					value2, err := strconv.ParseFloat(values[i+1], 64)
					if err != nil {
						return nil, nil, err
					}
					task := Task{
						Arg1:         value1,
						Arg2:         value2,
						Operation:    op,
						Duration:     getOperationDuration(op),
						Status:       "pending",
						Dependencies: []int{},
						ID:           taskID,
						Index:        i,
					}
					taskID++
					dm[len(depMap)] = append(dm[len(depMap)], taskID)
					tasks = append(tasks, task)
				}
			} else if op == "*" || op == "/" {
				if ops[i-1] != "/" || ops[i-1] != "*" {
					value1, err := strconv.ParseFloat(values[i], 64)
					value2, err := strconv.ParseFloat(values[i+1], 64)
					if err != nil {
						return nil, nil, err
					}
					task := Task{
						Arg1:         value1,
						Arg2:         value2,
						Operation:    op,
						Duration:     getOperationDuration(op),
						Status:       "pending",
						Dependencies: []int{},
						ID:           taskID,
						Index:        i,
					}
					taskID++
					dm[len(depMap)] = append(dm[len(depMap)], taskID)
					tasks = append(tasks, task)
				}
			}
		}
	}
	depMap[len(dm)] = append(depMap[len(dm)], dm[len(dm)-1]...)
	for _, t := range tasks {
		for _, values := range depMap {
			for _, value := range values {
				t.Dependencies = append(t.Dependencies, value)
			}
		}
	}
	if len(tasks1) != 0 {
		tasks = append(tasks1, tasks...)
	}
	return tasks, depMap, nil
}

func HandleGetExpressions(w http.ResponseWriter, r *http.Request) {

	mu.RLock()
	defer mu.RUnlock()
	resp := map[string][]*Expression{"expressions": {}}
	for _, expr := range expressions {
		resp["expressions"] = append(resp["expressions"], expr)
	}
	json.NewEncoder(w).Encode(resp)
}

func HandleGetExpression(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	defer mu.RUnlock()

	id := chi.URLParam(r, "id")
	expr, exists := expressions[id]
	if !exists {
		http.Error(w, "Expression not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]*Expression{"expression": expr})
}

func HandleGetTask(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	for _, task := range tasks {
		if task.Status == "pending" {
			taskCopy := task
			taskCopy.Status = "in_progress"
			json.NewEncoder(w).Encode(map[string]Task{"task": taskCopy})
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "No tasks available"})
}

func HandlePostTaskResult(w http.ResponseWriter, r *http.Request) {
	var resultData map[string]interface{}

	if err := json.NewDecoder(r.Body).Decode(&resultData); err != nil {
		http.Error(w, "Invalid data", http.StatusUnprocessableEntity)
		return
	}

	id, ok := resultData["id"].(float64)

	if !ok {
		http.Error(w, "Invalid task ID", http.StatusUnprocessableEntity)
		return
	}

	taskID := int(id)
	task, exists := tasks[taskID]
	if !exists {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	result, ok := resultData["result"].(float64)
	if !ok {
		http.Error(w, "Invalid result", http.StatusUnprocessableEntity)
		return
	}
	newSlice := append(thisnode.Valuse[:task.Index], fmt.Sprintf("%f", result))
	if task.Index+2 < len(thisnode.Valuse) {
		newSlice = append(newSlice, thisnode.Valuse[task.Index+2:]...)
	}

	thisnode = Node{
		Valuse:    newSlice,
		Operators: append(thisnode.Operators[:task.Index], thisnode.Operators[:task.Index+1]...),
	}
	task.Status = "completed"
	task.Result = result
	tasks[taskID] = task
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Result received"})

	IsResult = true
}

func calculateFinalResult(taskIDs []int, depMap map[int][]int) float64 {
	results := make(map[int]float64)
	for _, tid := range taskIDs {
		task := tasks[tid]
		if task.Operation == "" {
			continue
		}
		var arg1, arg2 float64
		switch v := task.Arg1.(type) {
		case int:
			arg1 = results[v]
		case float64:
			arg1 = v
		default:
			panic("unknown type for Arg1")
		}
		switch v := task.Arg2.(type) {
		case int:
			arg2 = results[v]
		case float64:
			arg2 = v
		default:
			panic("unknown type for Arg2")
		}
		results[tid] = performOperation(arg1, arg2, task.Operation)
	}

	return results[taskIDs[len(taskIDs)-1]]
}

func topologicalSort(taskIDs []int, depMap map[int][]int) []int {
	inDegree := make(map[int]int)
	for _, deps := range depMap {
		for _, dep := range deps {
			inDegree[dep]++
		}
	}

	queue := []int{}
	for _, tid := range taskIDs {
		if inDegree[tid] == 0 {
			queue = append(queue, tid)
		}
	}

	sorted := []int{}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		sorted = append(sorted, current)

		for _, tid := range taskIDs {
			if contains(depMap[current], tid) {
				inDegree[tid]--
				if inDegree[tid] == 0 {
					queue = append(queue, tid)
				}
			}
		}
	}

	return sorted
}

func contains(slice []int, item int) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}
