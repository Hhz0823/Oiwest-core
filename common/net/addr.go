package net

import (
	"net"
	"strconv"
)

type AddressFamily byte

const (
	AddressFamilyIPv4   AddressFamily = 1
	AddressFamilyIPv6   AddressFamily = 2
	AddressFamilyDomain AddressFamily = 3
)

type Address struct {
	Family AddressFamily
	IP     net.IP
	Domain string
	Port   uint16
}

func (a *Address) Network() string {
	return "dccp"
}

func (a *Address) String() string {
	switch a.Family {
	case AddressFamilyIPv4, AddressFamilyIPv6:
		return net.JoinHostPort(a.IP.String(), strconv.Itoa(int(a.Port)))
	case AddressFamilyDomain:
		return net.JoinHostPort(a.Domain, strconv.Itoa(int(a.Port)))
	}
	return ""
}

func IPAddress(ip net.IP, port uint16) Address {
	family := AddressFamilyIPv4
	if len(ip) == net.IPv6len {
		family = AddressFamilyIPv6
	}
	return Address{
		Family: family,
		IP:     ip,
		Port:   port,
	}
}

func DomainAddress(domain string, port uint16) Address {
	return Address{
		Family: AddressFamilyDomain,
		Domain: domain,
		Port:   port,
	}
}

func ParseAddress(addr string) (Address, error) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return Address{}, err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return Address{}, err
	}
	ip := net.ParseIP(host)
	if ip != nil {
		return IPAddress(ip, uint16(port)), nil
	}
	return DomainAddress(host, uint16(port)), nil
}

type Destination struct {
	Address Address
	Network string
}

func (d Destination) String() string {
	return d.Address.String()
}

type TCPDestination struct {
	Address
}

func (d TCPDestination) Network() string {
	return "tcp"
}

type UDPDestination struct {
	Address
}

func (d UDPDestination) Network() string {
	return "udp"
}

type DCCPDestination struct {
	Address
}

func (d DCCPDestination) Network() string {
	return "dccp"
}
