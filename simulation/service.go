package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	cruxIPFS "github.com/dedis/student_19_cruxIPFS"
	"github.com/dedis/student_19_cruxIPFS/gentree"
	"github.com/dedis/student_19_cruxIPFS/service"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/app"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
	"go.dedis.ch/onet/v3/simul/monitor"
)

/*
 * Defines the simulation for the service-template
 */

func init() {
	onet.SimulationRegister(cruxIPFS.ServiceName, NewSimulationService)
}

// NewSimulationService returns the new simulation, where all fields are
// initialised using the config-file
func NewSimulationService(config string) (onet.Simulation, error) {
	es := &IPFSSimulation{}
	_, err := toml.Decode(config, es)
	if err != nil {
		return nil, err
	}
	return es, nil
}

// Setup creates the tree used for that simulation
func (s *IPFSSimulation) Setup(dir string, hosts []string) (
	*onet.SimulationConfig, error) {

	app.Copy(dir, "../clean.sh")

	sc := &onet.SimulationConfig{}
	s.CreateRoster(sc, hosts, 2000)
	err := s.CreateTree(sc)
	if err != nil {
		return nil, err
	}
	return sc, nil
}

// Node can be used to initialize each node before it will be run
// by the server. Here we call the 'Node'-method of the
// SimulationBFTree structure which will load the roster- and the
// tree-structure to speed up the first round.
func (s *IPFSSimulation) Node(config *onet.SimulationConfig) error {
	index, _ := config.Roster.Search(config.Server.ServerIdentity.ID)
	if index < 0 {
		log.Fatal("Didn't find this node in roster")
	}
	log.Lvl3("Initializing node-index", index)

	s.ReadNodeInfo(false)

	mymap := s.InitializeMaps(config, true)

	myService := config.GetService(cruxIPFS.ServiceName).(*service.Service)

	serviceReq := &cruxIPFS.InitRequest{
		Nodes:                s.Nodes.All,
		ServerIdentityToName: mymap,
	}

	if s.Nodes.GetServerIdentityToName(config.Server.ServerIdentity) == "node_19" {
		myService.InitRequest(serviceReq)

		for _, trees := range myService.BinaryTree {
			for _, tree := range trees {
				config.Overlay.RegisterTree(tree)
			}
		}
	}

	return s.SimulationBFTree.Node(config)
}

// Run is used on the destination machines and runs a number of
// rounds
func (s *IPFSSimulation) Run(config *onet.SimulationConfig) error {
	size := config.Tree.Size()
	log.Lvl2("Size is:", size, "rounds:", s.Rounds)
	//c := template.NewClient()
	for round := 0; round < s.Rounds; round++ {
		log.Lvl1("Starting round", round)
		round := monitor.NewTimeMeasure("round")
		round.Record()
	}
	return nil
}

func (s *IPFSSimulation) ReadNodeInfo(isLocalTest bool) {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	log.Lvl1(dir)
	if isLocalTest {
		s.ReadNodesFromFile(NODEPATHLOCAL)
	} else {
		s.ReadNodesFromFile(NODEPATHREMOTE)
	}

	for _, nodeRef := range s.Nodes.All {
		node := nodeRef
		log.Lvl1(node.Name, node.X, node.Y, node.IP)
	}
}

func (s *IPFSSimulation) ReadNodesFromFile(filename string) {
	s.Nodes.All = make([]*gentree.LocalityNode, 0)

	readLine := ReadFileLineByLine(filename)

	for true {
		line := readLine()
		//fmt.Println(line)
		if line == "" {
			//fmt.Println("end")
			break
		}

		if strings.HasPrefix(line, "#") {
			continue
		}

		tokens := strings.Split(line, " ")
		coords := strings.Split(tokens[1], ",")
		name, x_str, y_str, IP, level_str := tokens[0], coords[0], coords[1], tokens[2], tokens[3]

		x, _ := strconv.ParseFloat(x_str, 64)
		y, _ := strconv.ParseFloat(y_str, 64)
		level, err := strconv.Atoi(level_str)

		if err != nil {
			log.Lvl1("Error", err)

		}

		//	log.Lvl1("reqd node level", name, level_str, "lvl", level)

		myNode := CreateNode(name, x, y, IP, level)
		s.Nodes.All = append(s.Nodes.All, myNode)
	}
}

func (s *IPFSSimulation) InitializeMaps(config *onet.SimulationConfig, isLocalTest bool) map[*network.ServerIdentity]string {

	s.Nodes.ServerIdentityToName = make(map[network.ServerIdentityID]string)
	ServerIdentityToName := make(map[*network.ServerIdentity]string)

	if isLocalTest {
		for i := range s.Nodes.All {
			treeNode := config.Tree.List()[i]
			s.Nodes.All[i].ServerIdentity = treeNode.ServerIdentity
			s.Nodes.ServerIdentityToName[treeNode.ServerIdentity.ID] = s.Nodes.All[i].Name
			ServerIdentityToName[treeNode.ServerIdentity] = s.Nodes.All[i].Name
			log.Lvl1("associating", treeNode.ServerIdentity.String(), "to", s.Nodes.All[i].Name)
		}
	} else {
		for _, treeNode := range config.Tree.List() {
			serverIP := treeNode.ServerIdentity.Address.Host()
			node := s.Nodes.GetByIP(serverIP)
			node.ServerIdentity = treeNode.ServerIdentity
			s.Nodes.ServerIdentityToName[treeNode.ServerIdentity.ID] = node.Name
			ServerIdentityToName[treeNode.ServerIdentity] = node.Name
			log.Lvl1("associating", treeNode.ServerIdentity.String(), "to", node.Name)
		}
	}

	return ServerIdentityToName
}
