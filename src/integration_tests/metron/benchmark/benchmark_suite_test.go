package benchmark_test

import (
	"net/http"
	"os/exec"
	"runtime"

	"github.com/cloudfoundry/storeadapter/storerunner/etcdstorerunner"
	"github.com/pivotal-golang/localip"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"

	"github.com/onsi/gomega/gexec"
)

func TestBenchmark(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Benchmark Suite")
}

var metronSession *gexec.Session
var etcdRunner *etcdstorerunner.ETCDClusterRunner
var etcdPort int
var localIPAddress string

var _ = BeforeSuite(func() {
	pathToMetronExecutable, err := gexec.Build("metron")
	Expect(err).ShouldNot(HaveOccurred())

	command := exec.Command(pathToMetronExecutable, "--config=fixtures/metron.json")

	metronSession, err = gexec.Start(command, gexec.NewPrefixedWriter("[o][metron]", GinkgoWriter), gexec.NewPrefixedWriter("[e][metron]", GinkgoWriter))
	Expect(err).ShouldNot(HaveOccurred())

	localIPAddress, _ = localip.LocalIP()

	// wait for server to be up
	Eventually(func() error {
		_, err := http.Get("http://" + localIPAddress + ":1234")
		return err
	}, 3).ShouldNot(HaveOccurred())

	etcdPort = 4001
	etcdRunner = etcdstorerunner.NewETCDClusterRunner(etcdPort, 1)
	etcdRunner.Start()
})

var _ = AfterSuite(func() {
	metronSession.Kill().Wait()
	gexec.CleanupBuildArtifacts()

	etcdRunner.Adapter().Disconnect()
	// etcdRunner.Stop() send interrupt signal which doesn't work on Windows
	if runtime.GOOS == "windows" {
		etcdRunner.KillWithFire()
	} else { 
		etcdRunner.Stop() 
	}
})
