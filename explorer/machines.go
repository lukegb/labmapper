package explorer

import (
	"github.com/lukegb/labmapper/admindb"
	"github.com/lukegb/labmapper"
	"net"
	"log"
	"time"
	"fmt"
)

const (
	ALIVENESS_TIMEOUT = 2 * time.Second
)

func GetInterestingMachines(admindb *admindb.AdminDB) ([]labmapper.MachineIdentity, error) {
	labHosts, err := admindb.FindLabHosts()
	if err != nil {
		return nil, err
	}

	machines := make([]labmapper.MachineIdentity, len(labHosts))
	for n, labHost := range labHosts {
		machines[n] = labHost.ToMachine()
	}

	return machines, nil
}

type LivenessReport struct {
	Machine labmapper.MachineIdentity
	IsAlive bool
}

func GetAliveMachines(machines []labmapper.MachineIdentity) ([]labmapper.MachineIdentity) {
	// we'll attempt to connect on port 22.
	// if this fails, then the machine is dead.

	aliveOrDeadChan := make(chan LivenessReport)

	for _, machine := range machines {
		go func(machine labmapper.MachineIdentity, aliveOrDeadChan chan LivenessReport) {
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

	output := make([]labmapper.MachineIdentity, 0, len(machines))
	for _ = range machines {
		report := <-aliveOrDeadChan
		log.Println(report.Machine, report.IsAlive)
		if report.IsAlive {
			output = append(output, report.Machine)
		}
	}
	return output
}