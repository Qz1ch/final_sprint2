package main

import (
	"fmt"
	"log"
	"net/http"

	agent "yand/cmd/ag_launch"
	"yand/internal/orchestrator"
	"yand/pkg/add"

	"github.com/go-chi/chi/v5"
)

func main() {
	go agent.Run()
	r := chi.NewRouter()

	r.Post("/api/v1/calculate", orchestrator.HandleCalculate)
	r.Get("/api/v1/expressions", orchestrator.HandleGetExpressions)
	r.Get("/api/v1/expressions/{id}", orchestrator.HandleGetExpression)
	r.Get("/task", orchestrator.HandleGetTask)
	r.Post("/task/result", orchestrator.HandlePostTaskResult)

	log.Println("Server is running on port 8080")
	http.ListenAndServe(":8080", r)

	fmt.Println(add.ParseExpression("1+(1+1)"))
}
