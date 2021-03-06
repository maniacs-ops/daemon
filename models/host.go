/* cSploit - a simple penetration testing suite
 * Copyright (C) 2016 Massimo Dragano aka tux_mind <tux_mind@csploit.org>
 *
 * cSploit is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * cSploit is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with cSploit.  If not, see <http://www.gnu.org/licenses/\>.
 *
 */
package models

import (
	netHelper "github.com/cSploit/daemon/helpers/net"
	"github.com/cSploit/daemon/models/internal"
	"github.com/lair-framework/go-nmap"
	"github.com/op/go-logging"
	"net"
	"time"
)

func init() {
	internal.RegisterModels(&Host{})
}

var log = logging.MustGetLogger("daemon")

type Host struct {
	ID        uint      `json:"id"`
	CreatedAt time.Time `json:"first_seen"`
	UpdatedAt time.Time `json:"last_seen"`
	Name      *string   `json:"name"`
	IpAddr    string    `gorm:"index" json:"ip_addr"`
	HwAddr    *HwAddr   `json:"hw_addr"`
	Ports     []Port    `json:"ports"`
	Network   *Network  `json:"-"`
	NetworkID uint      `json:"network_id,omitempty"`
	Jobs      []Job     `json:"jobs" gorm:"many2many:job_hosts"`
}

func NewHost(h nmap.Host) *Host {
	res := new(Host)

	res.Ports = make([]Port, 0)

	for _, p := range h.Ports {
		res.Ports = append(res.Ports, *NewPort(p))
	}

	for _, a := range h.Addresses {
		if a.AddrType == "mac" {
			var err error
			res.HwAddr, err = NewHwAddr(a)

			if err != nil {
				log.Warningf("unable to load MAC address: %v", err)
			}

			log.Debugf("created HW Addr: %v", res.HwAddr)
		} else {
			res.IpAddr = a.Addr
		}
	}

	return res
}

func NotifyHostSeen(hwAddr net.HardwareAddr, ipAddr net.IP, name *string) {
	hwId, err := netHelper.MacAddrToUInt(hwAddr)

	if err != nil {
		log.Error(err)
		return
	}

	var HwAddrEntity HwAddr

	dbRes := internal.Db.Preload("Host").Find(&HwAddrEntity, hwId)

	if dbRes.RecordNotFound() {
		onNewHost(hwAddr, ipAddr, name)
	} else if dbRes.Error != nil {
		log.Error(dbRes.Error)
	} else if host := HwAddrEntity.Host; host == nil {
		onNewHostWithHwAddr(&HwAddrEntity, ipAddr, name)
	} else {
		onHostSeen(host, ipAddr, name)
	}
}

//TODO: fire an event for each of these functions

func onNewHost(hwAddr net.HardwareAddr, ipAddr net.IP, name *string) {
	if hw, err := NewHwAddr(hwAddr); err != nil {
		log.Error(err)
	} else {
		onNewHostWithHwAddr(hw, ipAddr, name)
	}
}

func onNewHostWithHwAddr(hwAddr *HwAddr, ipAddr net.IP, name *string) {
	host := Host{HwAddr: hwAddr, IpAddr: ipAddr.String(), Name: name}

	if err := internal.Db.Create(&host).Error; err != nil {
		log.Error(err)
	}
}

func onHostSeen(host *Host, ipAddr net.IP, name *string) {
	host.IpAddr = ipAddr.String()
	host.Name = name
	host.UpdatedAt = time.Now()
	if err := internal.Db.Save(host).Error; err != nil {
		log.Error(err)
	}
}
