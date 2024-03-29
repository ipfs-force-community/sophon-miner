// Code generated by MockGen. DO NOT EDIT.
// Source: ./miner/util.go

// Package mock is a generated GoMock package.
package mock

import (
	context "context"
	reflect "reflect"

	abi "github.com/filecoin-project/go-state-types/abi"
	network "github.com/filecoin-project/go-state-types/network"
	builtin "github.com/filecoin-project/venus/venus-shared/actors/builtin"
	gomock "github.com/golang/mock/gomock"
)

// MockWinningPoStProver is a mock of WinningPoStProver interface.
type MockWinningPoStProver struct {
	ctrl     *gomock.Controller
	recorder *MockWinningPoStProverMockRecorder
}

// MockWinningPoStProverMockRecorder is the mock recorder for MockWinningPoStProver.
type MockWinningPoStProverMockRecorder struct {
	mock *MockWinningPoStProver
}

// NewMockWinningPoStProver creates a new mock instance.
func NewMockWinningPoStProver(ctrl *gomock.Controller) *MockWinningPoStProver {
	mock := &MockWinningPoStProver{ctrl: ctrl}
	mock.recorder = &MockWinningPoStProverMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockWinningPoStProver) EXPECT() *MockWinningPoStProverMockRecorder {
	return m.recorder
}

// ComputeProof mocks base method.
func (m *MockWinningPoStProver) ComputeProof(arg0 context.Context, arg1 []builtin.ExtendedSectorInfo, arg2 abi.PoStRandomness, arg3 abi.ChainEpoch, arg4 network.Version) ([]builtin.PoStProof, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ComputeProof", arg0, arg1, arg2, arg3, arg4)
	ret0, _ := ret[0].([]builtin.PoStProof)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ComputeProof indicates an expected call of ComputeProof.
func (mr *MockWinningPoStProverMockRecorder) ComputeProof(arg0, arg1, arg2, arg3, arg4 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ComputeProof", reflect.TypeOf((*MockWinningPoStProver)(nil).ComputeProof), arg0, arg1, arg2, arg3, arg4)
}

// GenerateCandidates mocks base method.
func (m *MockWinningPoStProver) GenerateCandidates(arg0 context.Context, arg1 abi.PoStRandomness, arg2 uint64) ([]uint64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GenerateCandidates", arg0, arg1, arg2)
	ret0, _ := ret[0].([]uint64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GenerateCandidates indicates an expected call of GenerateCandidates.
func (mr *MockWinningPoStProverMockRecorder) GenerateCandidates(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GenerateCandidates", reflect.TypeOf((*MockWinningPoStProver)(nil).GenerateCandidates), arg0, arg1, arg2)
}
