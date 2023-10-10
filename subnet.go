package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

func beUint64(b []byte) uint64 {
	_ = b[7] // bounds check hint to compiler; see golang.org/issue/14808
	return uint64(b[7]) | uint64(b[6])<<8 | uint64(b[5])<<16 | uint64(b[4])<<24 |
		uint64(b[3])<<32 | uint64(b[2])<<40 | uint64(b[1])<<48 | uint64(b[0])<<56
}

func IPToUint128(ip net.IP) uint128 {
	return uint128{beUint64(ip[:8]), beUint64(ip[8:])}
}

func uint128ToIP(i uint128) net.IP {
	h := i.halves()
	return net.IP{
		byte(*h[0] >> 56),
		byte(*h[0] >> 48),
		byte(*h[0] >> 40),
		byte(*h[0] >> 32),
		byte(*h[0] >> 24),
		byte(*h[0] >> 16),
		byte(*h[0] >> 8),
		byte(*h[0]),
		byte(*h[1] >> 56),
		byte(*h[1] >> 48),
		byte(*h[1] >> 40),
		byte(*h[1] >> 32),
		byte(*h[1] >> 24),
		byte(*h[1] >> 16),
		byte(*h[1] >> 8),
		byte(*h[1]),
	}
}

func getSuffixUint128(prefixLen int, suffix string) (*uint128, error) {
	if strings.Contains(suffix, "/") {
		parts := strings.Split(suffix, "/")
		if len(parts) != 2 {
			goto ERRFMT
		}

		suffixLen, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("failed to convert suffix %s length: %s", suffix, err.Error())
		}
		if suffixLen < 1 || suffixLen > 128 {
			return nil, fmt.Errorf("invalid suffix length")
		}
		if suffixLen+prefixLen > 128 {
			return nil, fmt.Errorf("suffix %s length + prefix length > 128", suffix)
		}

		ip := net.ParseIP(parts[0])
		if ip == nil {
			goto ERR
		}
		ip16 := ip.To16()
		if ip16 == nil {
			goto ERR
		}
		ipUint128 := IPToUint128(ip16)
		suffixMask := mask6(128 - suffixLen)
		if !(ipUint128.and(suffixMask).isZero()) {
			return nil, fmt.Errorf("suffix length > %d", suffixLen)
		}

		return &ipUint128, nil
	} else {
		goto ERRFMT
	}
ERR:
	return nil, fmt.Errorf("invalid suffix")
ERRFMT:
	return nil, fmt.Errorf("invalid suffix format")
}

func genIPv6AddressBySuffix(address string, prefixLen int, suffix uint128) (string, error) {
	addressIP := net.ParseIP(address)
	if addressIP == nil {
		return "", fmt.Errorf("invalid ip address")
	}

	addressUint128 := IPToUint128(addressIP)
	prefixMask := mask6(prefixLen)
	prefixUint128 := addressUint128.and(prefixMask)

	return uint128ToIP(prefixUint128.or(suffix)).String(), nil
}
