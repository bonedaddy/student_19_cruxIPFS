package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"time"
)

// EditIPFSConfig edit the ipfs configuration file (mainly the ip)
func (s *Service) EditIPFSConfig() {
	addr := IPVersion + s.MyIP + TransportProtocol

	// select available ports
	ports, err := GetNextAvailablePorts(s.MinPort, s.MaxPort, IPFSPortNumber)
	checkErr(err)

	// [\"/ip4/0.0.0.0/tcp/5001\", \"/ip6/::/tcp/5001\"]
	//swarmList := []string{addr + SwarmPort}
	Swarm := MakeJSONArray([]string{addr +
		strconv.Itoa((*ports)[0])})

	// /ip4/127.0.0.1/tcp/5001
	API := MakeJSONElem(addr + strconv.Itoa((*ports)[1]))
	// /ip4/127.0.0.1/tcp/8080
	Gateway := MakeJSONElem(addr + strconv.Itoa((*ports)[2]))

	EditIPFSField(s.MyIPFSPath, "Addresses.API", API)
	EditIPFSField(s.MyIPFSPath, "Addresses.Gateway", Gateway)
	EditIPFSField(s.MyIPFSPath, "Addresses.Swarm", Swarm)
}

// EditIPFSField with the native IPFS config command
func EditIPFSField(path, field, value string) {
	cmd := "ipfs -c " + path + " config --json " + field + " " + value
	o, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		fmt.Println(cmd)
		fmt.Println(string(o))
		fmt.Println(err)
	}
}

// SetClusterLeaderConfig set the configs for the leader of a cluster
func SetClusterLeaderConfig(path, ip, peername string,
	replmin, replmax int, ports ClusterInstance) (
	string, string, error) {

	// generate random secret
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return "", "", errors.New("could not generate secret")
	}
	secret := hex.EncodeToString(key)

	vars := GetClusterVariables(path, ip, peername, secret,
		replmin, replmax, ports)
	return vars, secret, nil
}

// GetClusterVariables get the cluster variables
func GetClusterVariables(path, ip, secret, peername string,
	replmin, replmax int, ports ClusterInstance) string {

	apiIPFSAddr := IPVersion + ip +
		TransportProtocol + strconv.Itoa(ports.IPFSAPIPort) // 5001
	restAPIAddr := IPVersion + ip +
		TransportProtocol + strconv.Itoa(ports.RestAPIPort) // 9094
	IPFSProxyAddr := IPVersion + ip +
		TransportProtocol + strconv.Itoa(ports.IPFSProxyPort) // 9095
	clusterAddr := IPVersion + ip +
		TransportProtocol + strconv.Itoa(ports.ClusterPort) // 9096

	cmd := ""

	// edit peername
	cmd += GetEnvVar("CLUSTER_PEERNAME", peername)

	// edit the secret
	cmd += GetEnvVar("CLUSTER_SECRET", secret)

	// replace IPFS API port (5001)
	cmd += GetEnvVar("CLUSTER_IPFSPROXY_NODEMULTIADDRESS", apiIPFSAddr) // 5001
	cmd += GetEnvVar("CLUSTER_IPFSHTTP_NODEMULTIADDRESS", apiIPFSAddr)  // 5001

	// replace Cluster ports (9094, 9095, 9096)
	cmd += GetEnvVar("CLUSTER_RESTAPI_HTTPLISTENMULTIADDRESS", restAPIAddr) // 9094
	cmd += GetEnvVar("CLUSTER_IPFSPROXY_LISTENMULTIADDRESS", IPFSProxyAddr) // 9095
	cmd += GetEnvVar("CLUSTER_LISTENMULTIADDRESS", clusterAddr)             // 9096

	// replace replication factor
	cmd += GetEnvVar("CLUSTER_REPLICATIONFACTORMIN", strconv.Itoa(replmin))
	cmd += GetEnvVar("CLUSTER_REPLICATIONFACTORMAX", strconv.Itoa(replmax))

	return cmd
}

// GetEnvVar get the environnment variable for the given field and value
func GetEnvVar(field, value string) string {
	// `CLUSTER_FIELD="value" `
	return field + "=\"" + value + "\" "
}

// MakeJSONElem make a JSON single element
func MakeJSONElem(elem string) string {
	// \"elem\"
	return "\\\"" + elem + "\\\""
}

// MakeJSONArray make a json array from the given elements
func MakeJSONArray(elements []string) string {
	// "[
	str := "\"["
	for _, e := range elements {
		// \"elem\"
		str += "\\\"" + e + "\\\""
	}
	// str + ]"
	return str + "]\""
}

// SetupClusterLeader setup a cluster instance for the ARA leader
func SetupClusterLeader(configPath, nodeID, ip string,
	replmin, replmax int) (string, *ClusterInstance, error) {

	// generate random secret
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return "", nil, errors.New("could not generate secret")
	}
	secret := hex.EncodeToString(key)

	// path for config files
	path := configPath + "/cluster_" + secret
	err = CreateEmptyDir(path)
	if err != nil {
		return "", nil, err
	}

	ints, err := GetNextAvailablePorts(14000, 15000, 3)
	if err != nil {
		return "", nil, err
	}

	// set the ports that the cluster will use
	ports := ClusterInstance{
		IP:            IPVersion + ip + TransportProtocol,
		IPFSAPIPort:   5001,
		RestAPIPort:   (*ints)[0],
		IPFSProxyPort: (*ints)[1],
		ClusterPort:   (*ints)[2],
	}

	// get the environment variables to set cluster configs
	vars := GetClusterVariables(path, ip, secret, nodeID,
		replmin, replmax, ports)

	// init command to be run
	cmd := vars + "ipfs-cluster-service -c " + path + " init"
	o, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		fmt.Println(cmd)
		fmt.Println(string(o))
		fmt.Println(err)
		return "", nil, err
	}

	// start cluster daemon
	cmd = "ipfs-cluster-service -c " + path + " daemon"
	go func() {
		exec.Command("bash", "-c", cmd).Run()
		fmt.Println(ip + " cluster crashed")
	}()

	addr := IPVersion + ip + TransportProtocol + strconv.Itoa(ports.RestAPIPort)
	fmt.Println("Started ipfs-cluster at " + addr)

	// wait for the daemon to be launched
	time.Sleep(2 * time.Second)

	return secret, &ports, nil
}

// SetupClusterSlave setup a cluster slave instance
func SetupClusterSlave(configPath, nodeID, ip, bootstrap, secret string,
	replmin, replmax int) (*ClusterInstance, error) {

	// create the config directory, identified by the secret of the cluster
	path := configPath + "/cluster_" + nodeID + "_" + secret
	err := CreateEmptyDir(path)
	if err != nil {
		return nil, err
	}

	ints, err := GetNextAvailablePorts(14000, 15000, 3)
	if err != nil {
		return nil, err
	}

	// set the ports that the cluster will use
	ports := ClusterInstance{
		IP:            IPVersion + ip + TransportProtocol,
		IPFSAPIPort:   DefaultIPFSAPIPort,
		RestAPIPort:   (*ints)[0],
		IPFSProxyPort: (*ints)[1],
		ClusterPort:   (*ints)[2],
	}

	// get the environment variables to set cluster configs
	vars := GetClusterVariables(path, ip, secret, nodeID,
		replmin, replmax, ports)

	// init command to be run
	cmd := vars + "ipfs-cluster-service -c " + path + " init"
	err = exec.Command("bash", "-c", cmd).Run()
	if err != nil {
		return nil, err
	}

	// start cluster daemon
	cmd = "ipfs-cluster-service -c " + path + " daemon --bootstrap " + bootstrap
	go exec.Command("bash", "-c", cmd).Run()

	// wait for the daemon to be launched
	time.Sleep(2 * time.Second)

	addr := ports.IP + strconv.Itoa(ports.RestAPIPort)
	fmt.Println("Started ipfs-cluster at " + addr)

	return &ports, nil
}

// Protocol to start all clusters in an ARA
func Protocol(configPath, nodeID, ip string, replmin, replmax int) error {

	// for all ARAs (trees ?) where nodeID is the leader: do

	// setup the leader of the cluster
	secret, p, err := SetupClusterLeader(configPath, "master", ip,
		replmin, replmax)
	bootstrap := p.IP + strconv.Itoa(p.ClusterPort)
	if err != nil {
		return err
	}
	// for all nodes in this ARA
	_, err = SetupClusterSlave(configPath, "slave1", ip, bootstrap, secret,
		replmin, replmax)
	if err != nil {
		return err
	}
	_, err = SetupClusterSlave(configPath, "slave2", ip, bootstrap, secret,
		replmin, replmax)
	if err != nil {
		return err
	}

	return nil
}
