package imports_test

import (
	"testing"

	"github.com/werf/werf/integration/pkg/suite_init"

	"github.com/onsi/ginkgo"
)

var testSuiteEntrypointFunc = suite_init.MakeTestSuiteEntrypointFunc("Imports suite", suite_init.TestSuiteEntrypointFuncOptions{
	RequiredSuiteTools: []string{"docker", "git"},
})

func TestSuite(t *testing.T) {
	testSuiteEntrypointFunc(t)
}

var SuiteData suite_init.SuiteData

var (
	_ = SuiteData.SetupStubs(suite_init.NewStubsData())
	_ = SuiteData.SetupTmp(suite_init.NewTmpDirData())
	_ = SuiteData.SetupSynchronizedSuiteCallbacks(suite_init.NewSynchronizedSuiteCallbacksData())
	_ = SuiteData.SetupWerfBinary(suite_init.NewWerfBinaryData(SuiteData.SynchronizedSuiteCallbacksData))
	_ = SuiteData.SetupProjectName(suite_init.NewProjectNameData(SuiteData.StubsData))
)

var _ = ginkgo.BeforeEach(func() {
	SuiteData.Stubs.SetEnv("WERF_DISABLE_AUTO_HOST_CLEANUP", "1")
})
