package nomaddemo

import nomadapi "github.com/hashicorp/nomad/api"

// demoNode is a simulated Nomad client node. The pool below is shared across
// every simulated job, the same way a real Nomad cluster's clients host
// allocations from many jobs at once.
type demoNode struct {
	id         string
	name       string
	address    string
	datacenter string
}

var demoNodes = buildDemoNodes()

func buildDemoNodes() []demoNode {
	specs := []struct {
		name       string
		address    string
		datacenter string
	}{
		{"nomad-client-01", "10.20.1.11", "dc1"},
		{"nomad-client-02", "10.20.1.12", "dc1"},
		{"nomad-client-03", "10.20.1.13", "dc1"},
		{"nomad-client-04", "10.20.2.11", "dc2"},
		{"nomad-client-05", "10.20.2.12", "dc2"},
		{"nomad-client-06", "10.20.2.13", "dc2"},
		{"nomad-client-07", "10.20.3.11", "dc3"},
		{"nomad-client-08", "10.20.3.12", "dc3"},
	}

	nodes := make([]demoNode, len(specs))
	for i, s := range specs {
		nodes[i] = demoNode{
			id:         deterministicID("node", s.name),
			name:       s.name,
			address:    s.address,
			datacenter: s.datacenter,
		}
	}
	return nodes
}

// nodeAt returns a stable node assignment for a given slot, so the same
// logical replica always lands on the same simulated node.
func nodeAt(slot int) demoNode {
	return demoNodes[slot%len(demoNodes)]
}

func nodeListStubs() []*nomadapi.NodeListStub {
	stubs := make([]*nomadapi.NodeListStub, len(demoNodes))
	for i, n := range demoNodes {
		stubs[i] = &nomadapi.NodeListStub{
			ID:                    n.id,
			Name:                  n.name,
			Address:               n.address,
			Datacenter:            n.datacenter,
			Status:                "ready",
			SchedulingEligibility: "eligible",
		}
	}
	return stubs
}
