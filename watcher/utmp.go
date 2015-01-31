package watcher

// #include <unistd.h>
// #include <stdlib.h>
// #include <stdint.h>
// #include <stdio.h>
// #include <string.h>
// #include <sys/stat.h>
// #include <sys/types.h>
// #include <utmp.h>
//
// struct utmp_mix {
// 	uint64_t count;
//  struct utmp* utmps;
// };
// struct utmp_mix* read_utmp() {
//  struct stat fileStat;
//  if (stat("/var/run/utmp", &fileStat) < 0) { printf("stat\n"); return NULL; }
//  struct utmp_mix* bf = calloc(1, sizeof(struct utmp_mix));
//  if (bf == NULL) { printf("calloc\n"); return NULL; }
//  bf->count = fileStat.st_size / sizeof(struct utmp);
//  void* buf = calloc(1, fileStat.st_size);
//  if (buf == NULL) { printf("calloc\n"); free(bf); return NULL; }
//  bf->utmps = buf;
//  FILE* file = fopen("/var/run/utmp", "r");
//  size_t bread = 0;
//  while (bread < fileStat.st_size) {
//   size_t xread = fread(buf, 1, fileStat.st_size, file);
//   if (xread == 0) {
//    printf("fread\n");
//	  free(bf); free(buf); fclose(file); return NULL;
//   }
//   bread += xread;
//  }
//  return bf;
// }
// void free_utmp(struct utmp_mix* p) {
//  if (p->utmps != NULL) {
//   memset(p->utmps, 0, sizeof(struct utmp) * p->count);
//   free(p->utmps);
//  }
//  memset(p, 0, sizeof(struct utmp_mix));
//  free(p);
// }
import "C"
import "unsafe"
import "reflect"

import (
	// "golang.org/x/exp/inotify"
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

func ReadUtmp() (Utmp) {
	v := C.read_utmp()
	if v == nil {
		return nil
	}
	defer C.free_utmp(v)
	hdr := reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(v.utmps)),
		Len: int(v.count),
		Cap: int(v.count),
	}
	cSlice := *(*[]C.struct_utmp)(unsafe.Pointer(&hdr))

	goSlice := make([]UtmpData, 0, len(cSlice))
	for _, cData := range cSlice {
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
	return Utmp(goSlice)
}