package handlers

import "github.com/erigontech/assertoor/pkg/coordinator/buildinfo"

type SidebarData struct {
	ClientCount      uint64         `json:"client_count"`
	CLReadyCount     uint64         `json:"cl_ready_count"`
	CLHeadSlot       uint64         `json:"cl_head_slot"`
	CLHeadRoot       []byte         `json:"cl_head_root"`
	ELReadyCount     uint64         `json:"el_ready_count"`
	ELHeadNumber     uint64         `json:"el_head_number"`
	ELHeadHash       []byte         `json:"el_head_hash"`
	TestDescriptors  []*SidebarTest `json:"tests"`
	AllTestsActive   bool           `json:"all_tests_active"`
	RegistryActive   bool           `json:"registry_active"`
	CanRegisterTests bool           `json:"can_register_tests"`
	Version          string         `json:"version"`
}

type SidebarTest struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

func (fh *FrontendHandler) getSidebarData(activeTestID string) *SidebarData {
	sidebarData := &SidebarData{
		TestDescriptors:  []*SidebarTest{},
		AllTestsActive:   activeTestID == "*",
		RegistryActive:   activeTestID == "*registry",
		CanRegisterTests: !fh.securityTrimmed && fh.isAPIEnabled,
		Version:          buildinfo.GetVersion(),
	}

	// client pool status
	clientPool := fh.coordinator.ClientPool()
	allClients := clientPool.GetAllClients()
	sidebarData.ClientCount = uint64(len(allClients))

	canonicalClFork := clientPool.GetConsensusPool().GetCanonicalFork(2)
	if canonicalClFork != nil {
		sidebarData.CLReadyCount = uint64(len(canonicalClFork.ReadyClients))
		sidebarData.CLHeadSlot = uint64(canonicalClFork.Slot)
		sidebarData.CLHeadRoot = canonicalClFork.Root[:]
	}

	canonicalElFork := clientPool.GetExecutionPool().GetCanonicalFork(2)
	if canonicalElFork != nil {
		sidebarData.ELReadyCount = uint64(len(canonicalElFork.ReadyClients))
		sidebarData.ELHeadNumber = canonicalElFork.Number
		sidebarData.ELHeadHash = canonicalElFork.Hash[:]
	}

	// get test descriptors
	for _, testDescr := range fh.coordinator.TestRegistry().GetTestDescriptors() {
		if testDescr.Err() != nil {
			continue
		}

		testConfig := testDescr.Config()
		sidebarData.TestDescriptors = append(sidebarData.TestDescriptors, &SidebarTest{
			ID:     testDescr.ID(),
			Name:   testConfig.Name,
			Active: activeTestID == testDescr.ID(),
		})
	}

	return sidebarData
}
