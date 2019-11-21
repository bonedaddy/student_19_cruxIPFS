module github.com/dedis/student_19_cruxIPFS

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/dedis/cothority_template v0.0.0-20191121084815-f73b5bf67b5d
	github.com/satori/go.uuid v1.2.0
	github.com/stretchr/testify v1.4.0
	github.com/urfave/cli v1.22.0
	go.dedis.ch/cothority/v3 v3.3.1
	go.dedis.ch/kyber/v3 v3.0.9
	go.dedis.ch/onet/v3 v3.0.27
	go.dedis.ch/protobuf v1.0.9
	golang.org/x/sys v0.0.0-20190912141932-bc967efca4b8
	gopkg.in/urfave/cli.v1 v1.20.0
)

replace go.dedis.ch/onet/v3 => /mnt/guillaume/Documents/workspace/go/src/github.com/dedis/onet

go 1.13
