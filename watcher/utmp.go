package watcher

// #include <utmp.h>
import "C"
import "unsafe"
import "reflect"

import (
	"errors"
	"os"
	"io"
	"io/ioutil"
	"fmt"
	"time"
	"net"
)

type UtmpRecordType int
type UtmpPid uint64
type UtmpExitStatus struct {
	Termination int
	ExitCode int
}

const (
	UTMP_RT_EMPTY = UtmpRecordType(C.EMPTY)
	UTMP_RT_RUN_LVL = UtmpRecordType(C.RUN_LVL)
	UTMP_RT_BOOT_TIME = UtmpRecordType(C.BOOT_TIME)
	UTMP_RT_NEW_TIME = UtmpRecordType(C.NEW_TIME)
	UTMP_RT_OLD_TIME = UtmpRecordType(C.OLD_TIME)
	UTMP_RT_INIT_PROCESS = UtmpRecordType(C.INIT_PROCESS)
	UTMP_RT_LOGIN_PROCESS = UtmpRecordType(C.LOGIN_PROCESS)
	UTMP_RT_USER_PROCESS = UtmpRecordType(C.USER_PROCESS)
	UTMP_RT_DEAD_PROCESS = UtmpRecordType(C.DEAD_PROCESS)
	UTMP_RT_ACCOUNTING = UtmpRecordType(C.ACCOUNTING)
)

func (rt UtmpRecordType) String() string {
	switch rt {
	case UTMP_RT_EMPTY:
		return "EMPTY"
	case UTMP_RT_RUN_LVL:
		return "RUN_LVL"
	case UTMP_RT_BOOT_TIME:
		return "BOOT_TIME"
	case UTMP_RT_NEW_TIME:
		return "NEW_TIME"
	case UTMP_RT_OLD_TIME:
		return "OLD_TIME"
	case UTMP_RT_INIT_PROCESS:
		return "INIT_PROCESS"
	case UTMP_RT_LOGIN_PROCESS:
		return "LOGIN_PROCESS"
	case UTMP_RT_USER_PROCESS:
		return "USER_PROCESS"
	case UTMP_RT_DEAD_PROCESS:
		return "DEAD_PROCESS"
	case UTMP_RT_ACCOUNTING:
		return "ACCOUNTING"
	}
	return "<UNKNOWN>"
}

type UtmpData struct {
	RecordType UtmpRecordType
	Pid UtmpPid
	Line string
	Id string
	User string
	Host string
	Exit UtmpExitStatus
	SessionID uint64
	Time time.Time
	Addr net.IP
}

type Utmp []UtmpData

func (u Utmp) CurrentUsers() (Utmp) {
	return u.FilterByRecordTypes([]UtmpRecordType{UTMP_RT_USER_PROCESS})
}

func (u Utmp) Filter(test func(UtmpData)bool) (Utmp) {
	out := make([]UtmpData, 0)
	for _, row := range u {
		if test(row) {
			out = append(out, row)
		}
	}
	return out

}

func (u Utmp) FilterByRecordTypes(rts []UtmpRecordType) (Utmp) {
	allowedRts := make(map[UtmpRecordType]bool)
	for _, rt := range rts {
		allowedRts[rt] = true
	}

	return u.Filter(func (row UtmpData) bool {
		a, b := allowedRts[row.RecordType]
		return a &&b
	})
}

func (u Utmp) FilterByUsers(users []string) (Utmp) {
	allowedUsers := make(map[string]bool)
	for _, user := range users {
		allowedUsers[user] = true
	}

	return u.Filter(func (row UtmpData) bool {
		a, b := allowedUsers[row.User]
		return a &&b
	})
}

func ParseUtmp(data []byte) (Utmp, error) {
	utmpSize := int(unsafe.Sizeof(C.struct_utmp{}))
	if len(data) % utmpSize != 0 {
		return nil, errors.New(fmt.Sprintf(`utmp: mismatched sizes - %d doesn't divide into %d`, utmpSize, len(data)))
	}

	recordCount := len(data) / utmpSize
	intHdr := reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(&data[0])),
		Len: recordCount,
		Cap: recordCount,
	}
	intSlice := *(*[]C.struct_utmp)(unsafe.Pointer(&intHdr))

	goSlice := make([]UtmpData, 0, len(intSlice))
	for _, cData := range intSlice {
		if UtmpRecordType(cData.ut_type) == UTMP_RT_EMPTY {
			// bin
			continue
		}

		iphdr := reflect.SliceHeader{
			Data: uintptr(unsafe.Pointer(&cData.ut_addr_v6[0])),
			Len: 16,
			Cap: 16,
		}
		cIpSlice := *(*[]byte)(unsafe.Pointer(&iphdr))

		var ipAddr net.IP
		if cData.ut_addr_v6[1] == 0 && cData.ut_addr_v6[2] == 0 && cData.ut_addr_v6[3] == 0 {
			ipAddr = make(net.IP, 4)
			for x := 0; x < 4; x++ {
				ipAddr[x] = cIpSlice[x]
			}
		} else {
			ipAddr = make(net.IP, 16)
			for x := 0; x < 16; x++ {
				ipAddr[x] = cIpSlice[x]
			}
		}

		goSlice = append(goSlice, UtmpData{
			RecordType: UtmpRecordType(cData.ut_type),
			Pid: UtmpPid(cData.ut_pid),
			Line: C.GoString(&cData.ut_line[0]),
			Id: C.GoString(&cData.ut_id[0]),
			User: C.GoString(&cData.ut_user[0]),
			Host: C.GoString(&cData.ut_host[0]),
			Exit: UtmpExitStatus{
				Termination: int(cData.ut_exit.e_termination),
				ExitCode: int(cData.ut_exit.e_exit),
			},
			SessionID: uint64(cData.ut_session),
			Time: time.Unix(int64(cData.ut_tv.tv_sec), int64(cData.ut_tv.tv_usec)*1000),
			Addr: ipAddr,
		})
	}
	return Utmp(goSlice), nil
}

func ReadUtmp(file io.Reader) (Utmp, error) {
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return ParseUtmp(data)
}

const (
	UTMP_LOCATION = "/var/run/utmp"
	WTMP_LOCATION = "/var/log/wtmp"
)

func ReadUtmpFromFile(filename string) (Utmp, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return ReadUtmp(file)

}

func ReadUtmpFromSystem() (Utmp, error) {
	return ReadUtmpFromFile(UTMP_LOCATION)
}
func ReadWtmpFromSystem() (Utmp, error) {
	return ReadUtmpFromFile(WTMP_LOCATION)
}