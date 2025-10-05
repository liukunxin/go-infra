package utils

import (
	"errors"
	"fmt"
	"net"
	"runtime"
)

func LocalIp() (string, error) {
	var localIP string

	switch runtime.GOOS {
	case "linux":
		addrs, err := net.InterfaceAddrs()
		if err != nil {
			return "", err
		}
		for _, address := range addrs {
			// 检查ip地址是否为ipv4，并排除回环地址
			if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
				localIP = ipnet.IP.String()
				break
			}
		}
	case "windows":
		interfaces, err := net.Interfaces()
		if err != nil {
			return "", err
		}
		for _, iface := range interfaces {
			if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0 {
				addrs, err := iface.Addrs()
				if err != nil {
					return "", err
				}
				for _, address := range addrs {
					// 检查ip地址是否为ipv4，并排除回环地址
					if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
						localIP = ipnet.IP.String()
						break
					}
				}
			}
		}
	default:
		return "", errors.New("unable to determine local ip: unsupported operating system")
	}

	return localIP, nil
}

func ResolveHostAndPort(addr string) (string, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return "", err
	}

	if tcpAddr.IP.IsGlobalUnicast() {
		return tcpAddr.String(), nil
	} else if tcpAddr.IP == nil || tcpAddr.IP.IsUnspecified() {
		localIp, err := LocalIp()
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s:%d", localIp, tcpAddr.Port), nil
	} else {
		return "", fmt.Errorf("failed to resolve host and port: %s", addr)
	}
}
