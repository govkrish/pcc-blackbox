// Copyright © 2020 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package pcc

import (
	"encoding/json"
	"fmt"

	"github.com/lib/pq"
	avro "github.com/platinasystems/tiles/pccagent/models"
	"github.com/platinasystems/tiles/pccserver/models"
)

const INTERFACE_STATUS_UP = models.INTERFACE_STATUS_UP
const INTERFACE_STATUS_DOWN = models.INTERFACE_STATUS_DOWN

// Valid IntfState
const (
	Ready    = models.IntfState("ready")
	Queued   = models.IntfState("queued")
	Updating = models.IntfState("updating")
	Stale    = models.IntfState("stale")
	Offline  = models.IntfState("offline")
	Unknown  = models.IntfState("unknown")
)

// Temporary duplicate Interface definition until
// pcc-blackbox/lib have a node definition

// Fields with "Desired" suffix can be specified by user
// Otherfields are discovered or managed internally by PCC
type Interface struct {
	models.Interface
}

type InterfaceDetail struct {
	Interface          *Interface        `json:"interface"`
	RemoteLinksDetails []*avro.Interface `json:"remoteLinksDetails"`
}

type InterfaceRequest struct {
	//models.InterfaceRequest
	Id            uint64         `json:"-" gorm:"id"`
	InterfaceId   int64          `json:"interfaceId" gorm:"interface_id"`
	NodeId        uint64         `json:"nodeId" gorm:"node_id"`
	Name          string         `json:"ifName" gorm:"name"`
	Ipv4Addresses pq.StringArray `json:"ipv4Addresses" gorm:"ipv4_addresses"`
	Ipv6Addresses pq.StringArray `json:"ipv6Addresses" gorm:"ipv6_addresses"`
	IsManagement  string         `json:"management" gorm:"is_management"`
	Gateway       string         `json:"gateway" gorm:"gateway"`
	Autoneg       string         `json:"autoneg" gorm:"autoneg"`
	Speed         json.Number    `json:"speed,omitempty" gorm:"speed"`
	FecType       string         `json:"fecType" gorm:"fec_type"`
	MediaType     string         `json:"mediaType" gorm:"media_type"`
	Mtu           json.Number    `json:"mtu" gorm:"Mtu"`
	AdminStatus   string         `json:"adminStatus" gorm:"admin_status"`
	ManagedByPcc  bool           `json:"managedByPcc" gorm:"managed_by_pcc"`

	//kept for backward compatibility
	MacAddress string `json:"macAddress" gorm:"mac_address"`
	Ready      bool   `json:"ready" gorm:"ready"`
	Peer       string `json:"peer" gorm:"peer"`
	PeerId     uint64 `json:"peerID" gorm:"peer_id"`
	Netmask    string `json:"netmask" gorm:"netmask"`
	Dummy      bool   `json:"dummy" gorm:"dummy"`
	Status     string `json:"status" gorm:"status"`
	Dns        string `json:"dns" gorm:"dns"`
	Restore    bool   `json:"-" gorm:"restore"`
}

func (p *PccClient) GetIfacesByNodeId(nodeId uint64) (ifaces []*InterfaceDetail,
	err error) {

	var (
		resp     HttpResp
		endpoint string
		node     NodeDetail
	)
	endpoint = fmt.Sprintf("pccserver/node/%v", nodeId)
	resp, _, err = p.pccGateway("GET", endpoint, nil)
	if err != nil {
		return
	}
	if resp.Status == 200 {
		err = json.Unmarshal(resp.Data, &node)
		if err != nil {
			return
		}
		ifaces = node.Interfaces
		return
	}
	err = fmt.Errorf("GetIfaceByNodeId failed: %v", resp.Error)
	return
}

func (p *PccClient) GetIfaceById(nodeId uint64, ifaceId int64) (iface *InterfaceDetail, err error) {

	ifaces, err := p.GetIfacesByNodeId(nodeId)
	if err != nil {
		return
	}
	for _, i := range ifaces {
		if i.Interface.Id == ifaceId {
			iface = i
			return
		}
	}
	err = fmt.Errorf("error getting interface %v on node %v",
		ifaceId, nodeId)
	return
}

func (p *PccClient) GetIfaceByMacAddress(mAddr string, ifaces []*InterfaceDetail) (iface *InterfaceDetail, err error) {
	if mAddr == "" {
		err = fmt.Errorf("Invalid mac [%v]\n", mAddr)
		return
	}
	for i := 0; i < len(ifaces); i++ {
		if ifaces[i].Interface.MacAddress == mAddr {
			iface = ifaces[i]
			return
		}
	}
	err = fmt.Errorf("Error: couldn't find mac [%v]\n", mAddr)
	return
}

func (p *PccClient) SetIfaceApply(iface InterfaceRequest) (err error) {
	var (
		data     []byte
		resp     HttpResp
		endpoint string
	)
	endpoint = fmt.Sprintf("pccserver/interface")
	data, err = json.Marshal(iface)
	if err != nil {
		return fmt.Errorf("Iface format not valid")
	}
	resp, _, err = p.pccGateway("POST", endpoint, data)
	if err != nil {
		return
	}
	if resp.Status == 200 {
		return nil
	}
	endpoint = fmt.Sprintf("pccserver/interface/apply")
	var jsonData = map[string]string{"nodeId": string(iface.NodeId)}
	data, err = json.Marshal(jsonData)
	if err != nil {
		return
	}
	resp, _, err = p.pccGateway("POST", endpoint, data)
	if err != nil {
		return
	}
	if resp.Status == 200 {
		return nil
	}
	return fmt.Errorf(resp.Message)
}

func (p *PccClient) SetIface(iface InterfaceRequest) (err error) {
	var (
		data     []byte
		resp     HttpResp
		endpoint string
	)
	endpoint = fmt.Sprintf("pccserver/interface")
	data, err = json.Marshal(iface)
	if err != nil {
		err = fmt.Errorf("Iface format not valid")
		return
	}
	resp, _, err = p.pccGateway("POST", endpoint, data)
	if err != nil {
		return
	}
	if resp.Status == 200 {
		return
	}
	return
}

func (p *PccClient) ApplyIface(nodeId uint64) (err error) {
	var (
		data     []byte
		resp     HttpResp
		endpoint string
	)
	endpoint = fmt.Sprintf("pccserver/interface/apply")
	var jsonData = map[string]uint64{"nodeId": nodeId}
	data, err = json.Marshal(jsonData)
	if err != nil {
		return
	}
	resp, _, err = p.pccGateway("POST", endpoint, data)
	if err != nil {
		return
	}
	if resp.Status == 200 {
		return
	}
	err = fmt.Errorf(resp.Error)
	return
}

func (p *PccClient) SetIfaceAdmin(nodeId uint64, ifaceId int64, upDown string) (err error) {
	var (
		ir       InterfaceRequest
		data     []byte
		resp     HttpResp
		endpoint string
	)

	ir.NodeId = nodeId
	ir.InterfaceId = ifaceId
	ir.AdminStatus = upDown

	if upDown == INTERFACE_STATUS_UP {
		endpoint = fmt.Sprintf("pccserver/interface/up")
	} else {
		endpoint = fmt.Sprintf("pccserver/interface/down")
	}

	data, err = json.Marshal(ir)
	if err != nil {
		return
	}
	resp, _, err = p.pccGateway("POST", endpoint, data)
	if err != nil {
		return
	}
	if resp.Status == 200 {
		return
	}
	err = fmt.Errorf(resp.Error)
	return
}
