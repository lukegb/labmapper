package explorer

import (
	"github.com/lukegb/labmapper/admindb"
	"github.com/lukegb/labmapper"
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