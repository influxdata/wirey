package plumbing

import (
	"fmt"

	"github.com/vishvananda/netlink"
)

func Something() {
	lo, _ := netlink.LinkByName("lo")
	fmt.Printf("MTU: %d\n", lo.Attrs().MTU)

}
