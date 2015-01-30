package admindb

import (
	"database/sql"
	"fmt"
	"net"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// based off Hosts table
type Host struct {
	Id               uint64
	Comment          *string
	AlteredBy        *string
	LastChange       time.Time
	Hostname         string
	Domain           string
	IPAddress        net.IP
	HwAddress        *string
	Arch             *string
	OS               *string
	Room             *string
	MainUser         *string
	Section          string
	Padlock          *string
	Switch           *string
	ScreenLock       *string
	HostGuid         *string
	ProvService      *string
	SupportPrimary   *string
	SupportSecondary *string
	IPv6Address      *net.IP
	Funding          *string
	ScreenType       *string
	AssetNumber      *string

	adminDB *AdminDB
}

type dbScanner interface {
	Scan(dest ...interface{}) error
}

func (a *AdminDB) rowToHost(row dbScanner) (*Host, error) {
	host := Host{adminDB: a}
	var ipv4addr string
	var ipv6addr *string
	err := row.Scan(&host.Id, &host.Comment, &host.AlteredBy, &host.LastChange, &host.Hostname, &host.Domain, &ipv4addr, &host.HwAddress, &host.Arch, &host.OS, &host.Room, &host.MainUser, &host.Section, &host.Padlock, &host.Switch, &host.ScreenLock, &host.HostGuid, &host.ProvService, &host.SupportPrimary, &host.SupportSecondary, &ipv6addr, &host.Funding, &host.ScreenType, &host.AssetNumber)

	host.IPAddress = net.ParseIP(ipv4addr)
	if ipv6addr != nil {
		ipv6addr_ := net.ParseIP(*ipv6addr)
		host.IPv6Address = &ipv6addr_
	}

	if err != nil {
		return nil, err
	}
	return &host, err
}

func (a *AdminDB) GetHostByID(id uint64) (*Host, error) {
	db := a.database
	row := db.QueryRow("SELECT * FROM hosts WHERE ID = $1", id)
	return a.rowToHost(row)
}

func (a *AdminDB) GetHostByHostname(hostname, domain string) (*Host, error) {
	db := a.database
	row := db.QueryRow("SELECT * FROM hosts WHERE hostname = $1 AND domain = $2", hostname, domain)
	return a.rowToHost(row)
}

func (a *AdminDB) getClassesAndMachinesByQuery(class string, query string) ([]string, []string, error) {
	// ordinarily, I'd just use WITH RECURSIVE, but that's not an option here
	// so, starting from the top:
	classes := make(map[string]bool)
	pendingClasses := make(map[string]bool)
	pendingClasses[class] = false
	machines := make(map[string]bool)
	db := a.database

	for len(pendingClasses) != len(classes) {
		for pendingClass, pendingClassAlreadyDone := range pendingClasses {
			if pendingClassAlreadyDone {
				continue
			}

			classes[pendingClass] = true
			pendingClasses[pendingClass] = true

			// select all children
			rows, err := db.Query(query, pendingClass)
			if err != nil {
				return nil, nil, err
			}
			defer rows.Close()
			for rows.Next() {
				var childName *string

				if err := rows.Scan(&childName); err != nil {
					return nil, nil, err
				}

				if childName == nil {
					// we've probably reached the top, if going upwards...
					// so skip this
					continue
				}

				firstRune, _ := utf8.DecodeRuneInString(*childName)

				if _, in := pendingClasses[*childName]; unicode.IsUpper(firstRune) && !in {
					pendingClasses[*childName] = false
				} else if !unicode.IsUpper(firstRune) {
					machines[*childName] = true
				}
			}
		}
	}

	// clean classes to make sure we haven't included classes in it
	for class := range classes {
		firstRune, _ := utf8.DecodeRuneInString(class)
		if !unicode.IsUpper(firstRune) {
			delete(classes, class)
		}
	}

	return mapToArr(classes), mapToArr(machines), nil
}

func (a *AdminDB) getChildClassesAndMachines(class string) ([]string, []string, error) {
	return a.getClassesAndMachinesByQuery(class, `SELECT child FROM host_class WHERE parent = $1`)
}

func (a *AdminDB) getParentClasses(classOrMachine string) ([]string, error) {
	classes, _, err := a.getClassesAndMachinesByQuery(classOrMachine, `SELECT parent FROM host_class WHERE child = $1`)
	return classes, err
}

func (h *Host) Classes() ([]string, error) {
	return h.adminDB.getParentClasses(h.Hostname)
}

func mapToArr(inp map[string]bool) []string {
	outArr := make([]string, 0, len(inp))
	for k := range inp {
		outArr = append(outArr, k)
	}
	return outArr
}

func makeQueryStr(inp []string) (string, []interface{}) {
	inpArr := make([]interface{}, len(inp))
	queryArr := make([]string, len(inp))
	for n, machine := range inp {
		inpArr[n] = machine
		queryArr[n] = fmt.Sprintf("$%d", n+1)
	}
	queryStr := strings.Join(queryArr, ", ")

	return queryStr, inpArr
}

func (a *AdminDB) queryToHosts(rows *sql.Rows, err error) ([]*Host, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	outArr := []*Host{}
	for rows.Next() {
		host, err := a.rowToHost(rows)
		if err != nil {
			return nil, err
		}
		outArr = append(outArr, host)
	}

	return outArr, nil
}

func (a *AdminDB) GetHostsByHostclass(class string) ([]*Host, error) {
	_, machines, err := a.getChildClassesAndMachines(class)
	if err != nil {
		return nil, err
	}

	queryStr, machineArr := makeQueryStr(machines)

	return a.queryToHosts(a.database.Query(fmt.Sprintf(`SELECT * FROM hosts WHERE hostname IN (%s) ORDER BY id`, queryStr), machineArr...))
}
