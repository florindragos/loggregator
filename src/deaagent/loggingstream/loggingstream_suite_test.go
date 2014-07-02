// +build linux

package loggingstream_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestLoggingstream(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Loggingstream Suite")
}
