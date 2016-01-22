// porttrunk.go
package main

import (
	"fmt"
	"github.com/vishvananda/netlink"
	"net"
//	"os/exec"
	"strconv"
	"strings"
	"time"
)

// command to show status of lag
// cat /proc/net/bonding/bond-0

var BondedIfIdxBase int

// the id of the agg should be part of the name
// format expected <name>-<#number>
func GetIdByName(AggName string) int {
	var i int
	if strings.Contains(AggName, "-") {
		i, _ = strconv.Atoi(strings.Split(AggName, "-")[1])
	} else {
		i = 0
	}
	return i
}

// BondedLinkCreate will create a bonded interface
// bondname must be in the format of name-<number>
func BondLinkCreate(bondname string, mac string, hashmode int) (link netlink.Link, err error) {
	fmt.Println("in bonnded create for bondname ", bondname)
	hwmac, _ := net.ParseMAC(mac)
	var linkAttrs = netlink.LinkAttrs{
		//	Index:        BondedIfIdxBase + GetIdByName(bondname),
		Name:         bondname,
		HardwareAddr: hwmac,
	}

	fmt.Println("linkAttrs.Name=", linkAttrs.Name)
	bondedif := netlink.NewLinkBond(linkAttrs)
	bondedif.Mode = netlink.BOND_MODE_BALANCE_RR
	bondedif.XmitHashPolicy = netlink.BondXmitHashPolicy(hashmode)
	bondedif.MinLinks = 1
	err = netlink.LinkAdd(bondedif)
	if err != nil {
		fmt.Println("err from Bond LinkAdd = ", err)
		return bondedif, err
	}
	/*
		time.Sleep(time.Second * 1)
		err = netlink.LinkSetUp(bondedif)
		if err != nil {
			fmt.Println("err from Bond LinkSetUp = ", err)
			return bondedif, err
		}
	*/
	return bondedif, err
}

func BondLinkDelete(bondname string) (err error) {
	if bondedif, err := netlink.LinkByName(bondname); err == nil {
		err = netlink.LinkDel(bondedif)
	}
	return err
}

func AddLinkToBond(bondname string, linkname string) (err error) {
	if bondedif, err := netlink.LinkByName(bondname); err == nil {
		if linkif, err := netlink.LinkByName(linkname); err == nil {

			// link should be down before we add it to the bonded interface
			err = netlink.LinkSetDown(linkif)
			if err != nil {
				fmt.Println("err from Link LinkSetDown = ", err)
				return err
			}

			time.Sleep(time.Second * 1)

			linkif.Attrs().ParentIndex = bondedif.Attrs().Index
			err = netlink.LinkSetMasterByIndex(linkif, bondedif.Attrs().Index)
			if err != nil {
				fmt.Println("err from Add Link to Bond LinkSetMasterByIndex = ", err)
				return err
			}
			fmt.Println("Adding interface", linkname, "to bonded interface", bondname)
		}
	}
	return err
}

func DelLinkFromBond(bondname string, linkname string) (err error) {
	if bondedif, err := netlink.LinkByName(bondname); err == nil {
		if linkif, err := netlink.LinkByName(linkname); err == nil {

			// link should be down before we add it to the bonded interface
			err = netlink.LinkSetDown(linkif)
			if err != nil {
				fmt.Println("err from LinkSetDown = ", err)
				return err
			}

			time.Sleep(time.Second * 1)
			linkif.Attrs().ParentIndex = bondedif.Attrs().Index
			err = netlink.LinkSetNoMaster(linkif)
			if err != nil {
				fmt.Println("err from LinkSetNoMaster = ", err)
				return err
			}
			fmt.Println("Deleting interface", linkname, "from bonded interface", bondname)
		}
	}
	return err
}

func main() {

	BondedIfIdxBase = 51
	bondname := "bond0"
	linkname := "eth0"
	BondLinkCreate(bondname, "00:DD:EE:AA:DD:00", 0)

	time.Sleep(time.Second * 1)
	AddLinkToBond(bondname, linkname)
/*
	time.Sleep(time.Second * 1)

	binary, lookErr := exec.LookPath("ifconfig")
	if lookErr != nil {
		fmt.Println("ifconfig not found lookerr = ", lookErr)
	}

	out, err := exec.Command(binary).Output()
	if err != nil {
		fmt.Println("Error executing ifconfig")
	}
	fmt.Println(string(out))
	time.Sleep(time.Second * 1)
	binary2, lookErr2 := exec.LookPath("cat")
	if lookErr2 != nil {
		fmt.Println("cat not found lookerr = ", lookErr2)
	}

	out, err = exec.Command(binary2, fmt.Sprintf("/proc/net/bonding/%s", bondname)).Output()
	if err != nil {
		fmt.Println("Error executing ifconfig")
	}
	fmt.Println(string(out))

	fmt.Println("Deleting link from Bond")
	DelLinkFromBond(bondname, linkname)

	time.Sleep(time.Second * 1)
	fmt.Println("Deleting Bond")
	BondLinkDelete(bondname)

	time.Sleep(time.Second * 1)
	out, err = exec.Command(binary).Output()
	if err != nil {
		fmt.Println("Error executing ifconfig")
	}
	fmt.Println(string(out))
*/
}
