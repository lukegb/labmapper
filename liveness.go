package labmapper

import (
	"net"
	"fmt"
	"time"
)

const (
	ALIVENESS_TIMEOUT = 2 * time.Second
)

type LivenessReport struct {
	Machine MachineIdentity
	IsAlive bool
}

func GetAliveMachines(machines []MachineIdentity) ([]MachineIdentity, []MachineIdentity) {
	// we'll attempt to connect on port 22.
	// if this fails, then the machine is dead.

	aliveOrDeadChan := make(chan LivenessReport)

	for _, machine := range machines {
		go func(machine MachineIdentity, aliveOrDeadChan chan LivenessReport) {
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s.%s:22", machine.ShortName, machine.Domain), ALIVENESS_TIMEOUT)
			if err != nil {
				aliveOrDeadChan <- LivenessReport{
					Machine: machine,
					IsAlive: false,
				}
				return
			}
			defer conn.Close()

			aliveOrDeadChan <- LivenessReport{
				Machine: machine,
				IsAlive: true,
			}
		}(machine, aliveOrDeadChan)
	}

	aliveMachines := make([]MachineIdentity, 0, len(machines))
	deadMachines := make([]MachineIdentity, 0, len(machines))
	for _ = range machines {
		report := <-aliveOrDeadChan
		if report.IsAlive {
			aliveMachines = append(aliveMachines, report.Machine)
		} else {
			deadMachines = append(deadMachines, report.Machine)
		}
	}
	return aliveMachines, deadMachines
}