package agent

import (
	"log"
	"os"
	"strconv"
	"yand/internal/agent"
)

func Run() {
	cp := os.Getenv("COMPUTING_POWER")
	if cp == "" {
		cp = "2"
	}
	computingPower, err := strconv.Atoi(cp)
	if err != nil {
		log.Fatalf("Invalid COMPUTING_POWER value: %v", err)
	}

	agent.AgentWorker(computingPower)

	select {}
}
