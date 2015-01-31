package admindb

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