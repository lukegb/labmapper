package admindb

import (
	"strings"
)

func (a *AdminDB) FindLabHosts() ([]*Host, error) {
	db := a.database
	return a.queryToHosts(db.Query(`
		SELECT h.*, d.full_domain_name
		FROM hosts h
		LEFT JOIN domains d
			ON d.domain = h.domain
		WHERE h.hostname IN (
			SELECT h.child FROM host_class h WHERE h.parent IN (SELECT DISTINCT h.parent FROM host_class h LEFT JOIN host_class hc ON hc.child = h.parent WHERE h.child LIKE '%0%' AND hc.parent = 'LAB' AND h.parent != 'THEATRE') ORDER BY h.child
		)`))
}

func (a *AdminDB) IsLabHost(hostname string) (bool, error) {
	firstbit := hostname
	domainname := ""
	if strings.Contains(firstbit, ".") {
		splitbit := strings.SplitN(firstbit, ".", 2)
		firstbit = splitbit[0]
		domainname = splitbit[1]
	}

	db := a.database
	rows, err := db.Query(`
		SELECT hc.child
		FROM host_class hc
		LEFT JOIN hosts h ON h.hostname = hc.child
		LEFT JOIN domains d ON d.domain = h.domain
		WHERE
			hc.parent IN
				(SELECT DISTINCT h.parent FROM host_class h LEFT JOIN host_class hc ON hc.child = h.parent WHERE h.child LIKE '%0%' AND hc.parent = 'LAB' AND h.parent != 'THEATRE')
			AND (hc.child = $1 OR (h.hostname = $2 AND d.full_domain_name = $3))`, hostname, firstbit, domainname)
	if err != nil {
		return false, err
	}

	return rows.Next(), nil
}
