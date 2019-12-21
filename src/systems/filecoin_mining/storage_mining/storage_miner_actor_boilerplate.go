package storage_mining

import (
	ipld "github.com/filecoin-project/specs/libraries/ipld"
	actor "github.com/filecoin-project/specs/systems/filecoin_vm/actor"
	actor_util "github.com/filecoin-project/specs/systems/filecoin_vm/actor_util"
	vmr "github.com/filecoin-project/specs/systems/filecoin_vm/runtime"
	util "github.com/filecoin-project/specs/util"
)

type SectorStorageWeightDesc = actor_util.SectorStorageWeightDesc
type SectorTerminationType = actor_util.SectorTerminationType

////////////////////////////////////////////////////////////////////////////////
// Boilerplate
//
// This boilerplate should be essentially identical for all actors, and
// conceptually belongs in the runtime/VM. It is only duplicated here as a
// workaround due to the lack of generics support in Go.
////////////////////////////////////////////////////////////////////////////////

type Any = util.Any
type Bool = util.Bool
type Bytes = util.Bytes
type InvocOutput = vmr.InvocOutput
type Runtime = vmr.Runtime

var TODO = util.TODO
var IMPL_FINISH = util.IMPL_FINISH
var IMPL_TODO = util.IMPL_TODO

func (a *StorageMinerActorCode_I) State(rt Runtime) (vmr.ActorStateHandle, StorageMinerActorState) {
	h := rt.AcquireState()
	stateCID := h.Take()
	stateBytes := rt.IpldGet(ipld.CID(stateCID))
	if stateBytes.Which() != vmr.Runtime_IpldGet_FunRet_Case_Bytes {
		rt.AbortAPI("IPLD lookup error")
	}
	state := DeserializeState(stateBytes.As_Bytes())
	return h, state
}
func Release(rt Runtime, h vmr.ActorStateHandle, st StorageMinerActorState) {
	checkCID := actor.ActorSubstateCID(rt.IpldPut(st.Impl()))
	h.Release(checkCID)
}
func UpdateRelease(rt Runtime, h vmr.ActorStateHandle, st StorageMinerActorState) {
	newCID := actor.ActorSubstateCID(rt.IpldPut(st.Impl()))
	h.UpdateRelease(newCID)
}
func (st *StorageMinerActorState_I) CID() ipld.CID {
	panic("TODO")
}
func DeserializeState(x Bytes) StorageMinerActorState {
	panic("TODO")
}
