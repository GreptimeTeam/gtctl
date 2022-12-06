package tests

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGtctl(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gtctl Suite")
}
