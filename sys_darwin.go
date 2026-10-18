package main

/*
#include <mach/mach.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <sys/sysctl.h>
#include <net/if.h>
#include <net/route.h>
#include <netinet/in.h>
#include <stdint.h>
#include <stdlib.h>

static mach_port_t host_port;

// Aggregate CPU ticks across all cores: user, system, idle, nice.
static int cpu_ticks(uint32_t out[4]) {
	if (!host_port) host_port = mach_host_self();
	host_cpu_load_info_data_t info;
	mach_msg_type_number_t count = HOST_CPU_LOAD_INFO_COUNT;
	if (host_statistics(host_port, HOST_CPU_LOAD_INFO, (host_info_t)&info, &count) != KERN_SUCCESS)
		return -1;
	out[0] = info.cpu_ticks[CPU_STATE_USER];
	out[1] = info.cpu_ticks[CPU_STATE_SYSTEM];
	out[2] = info.cpu_ticks[CPU_STATE_IDLE];
	out[3] = info.cpu_ticks[CPU_STATE_NICE];
	return 0;
}

static char *fetch(int *mib, size_t *len) {
	for (int try = 0; try < 3; try++) {
		if (sysctl(mib, 6, NULL, len, NULL, 0) < 0) return NULL;
		*len += *len / 8 + 512;
		char *buf = malloc(*len);
		if (!buf) return NULL;
		if (sysctl(mib, 6, buf, len, NULL, 0) == 0) return buf;
		free(buf);
	}
	return NULL;
}

// 64-bit byte counters for one interface.
static int if_bytes(int index, uint64_t *ib, uint64_t *ob) {
	int mib[6] = {CTL_NET, PF_ROUTE, 0, 0, NET_RT_IFLIST2, index};
	size_t len;
	char *buf = fetch(mib, &len);
	if (!buf) return -1;
	int rc = -1;
	for (char *p = buf; p + sizeof(struct if_msghdr) <= buf + len;) {
		struct if_msghdr *ifm = (struct if_msghdr *)p;
		if (ifm->ifm_msglen == 0) break;
		if (ifm->ifm_type == RTM_IFINFO2) {
			struct if_msghdr2 *m = (struct if_msghdr2 *)p;
			if (m->ifm_index == index) {
				*ib = m->ifm_data.ifi_ibytes;
				*ob = m->ifm_data.ifi_obytes;
				rc = 0;
				break;
			}
		}
		p += ifm->ifm_msglen;
	}
	free(buf);
	return rc;
}

#define SA_ROUNDUP(a) ((a) > 0 ? (1 + (((a) - 1) | (sizeof(uint32_t) - 1))) : sizeof(uint32_t))

// zero_addr reports whether the IPv4 address bytes of sa (possibly truncated, as
// netmasks are) are all zero.
static int zero_addr(struct sockaddr *sa) {
	unsigned char *b = (unsigned char *)sa;
	for (int i = 4; i < sa->sa_len && i < 8; i++)
		if (b[i]) return 0;
	return 1;
}

// Interface index of the primary (unscoped) IPv4 default route, or 0.
static int default_ifindex(void) {
	int mib[6] = {CTL_NET, PF_ROUTE, 0, AF_INET, NET_RT_FLAGS, RTF_GATEWAY};
	size_t len;
	char *buf = fetch(mib, &len);
	if (!buf) return 0;
	int idx = 0;
	for (char *p = buf; p + sizeof(struct rt_msghdr) <= buf + len;) {
		struct rt_msghdr *rtm = (struct rt_msghdr *)p;
		if (rtm->rtm_msglen == 0) break;
		int flags = rtm->rtm_flags;
		if ((flags & RTF_UP) && (flags & RTF_GATEWAY) && !(flags & RTF_IFSCOPE) &&
		    (rtm->rtm_addrs & RTA_DST)) {
			char *q = (char *)(rtm + 1);
			struct sockaddr *dst = (struct sockaddr *)q;
			int ok = dst->sa_family == AF_INET && zero_addr(dst);
			if (ok && (rtm->rtm_addrs & RTA_NETMASK)) {
				for (int i = RTAX_DST; i < RTAX_NETMASK; i++)
					if (rtm->rtm_addrs & (1 << i))
						q += SA_ROUNDUP(((struct sockaddr *)q)->sa_len);
				ok = zero_addr((struct sockaddr *)q);
			}
			if (ok) {
				idx = rtm->rtm_index;
				break;
			}
		}
		p += rtm->rtm_msglen;
	}
	free(buf);
	return idx;
}
*/
import "C"

import "errors"

var errSys = errors.New("system call failed")

// cpuTicks returns cumulative user, system, idle and nice ticks summed over all cores.
func cpuTicks() ([4]uint32, error) {
	var t [4]C.uint32_t
	if C.cpu_ticks(&t[0]) != 0 {
		return [4]uint32{}, errSys
	}
	return [4]uint32{uint32(t[0]), uint32(t[1]), uint32(t[2]), uint32(t[3])}, nil
}

// ifBytes returns cumulative received and sent bytes for an interface index.
func ifBytes(index int) (in, out uint64, err error) {
	var ib, ob C.uint64_t
	if C.if_bytes(C.int(index), &ib, &ob) != 0 {
		return 0, 0, errSys
	}
	return uint64(ib), uint64(ob), nil
}

// defaultIfIndex returns the interface index carrying the default route, or 0 if offline.
func defaultIfIndex() int {
	return int(C.default_ifindex())
}
