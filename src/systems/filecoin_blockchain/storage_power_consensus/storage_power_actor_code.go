package storage_power_consensus

import (
	"math"

	ipld "github.com/filecoin-project/specs/libraries/ipld"
	libp2p "github.com/filecoin-project/specs/libraries/libp2p"
	block "github.com/filecoin-project/specs/systems/filecoin_blockchain/struct/block"
	sector "github.com/filecoin-project/specs/systems/filecoin_mining/sector"
	actor "github.com/filecoin-project/specs/systems/filecoin_vm/actor"
	addr "github.com/filecoin-project/specs/systems/filecoin_vm/actor/address"
	vmr "github.com/filecoin-project/specs/systems/filecoin_vm/runtime"
	exitcode "github.com/filecoin-project/specs/systems/filecoin_vm/runtime/exitcode"
	util "github.com/filecoin-project/specs/util"
)

const (
	MethodProcessPowerReport         = actor.MethodPlaceholder
	MethodSlashPledgeForStorageFault = actor.MethodPlaceholder
	EnsurePledgeCollateralSatisfied  = actor.MethodPlaceholder
)

////////////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////////////
type InvocOutput = vmr.InvocOutput
type Runtime = vmr.Runtime
type Bytes = util.Bytes
type State = StoragePowerActorState

func (a *StoragePowerActorCode_I) State(rt Runtime) (vmr.ActorStateHandle, State) {
	h := rt.AcquireState()
	stateCID := h.Take()
	stateBytes := rt.IpldGet(ipld.CID(stateCID))
	if stateBytes.Which() != vmr.Runtime_IpldGet_FunRet_Case_Bytes {
		rt.Abort("IPLD lookup error")
	}
	state := DeserializeState(stateBytes.As_Bytes())
	return h, state
}
func Release(rt Runtime, h vmr.ActorStateHandle, st State) {
	checkCID := actor.ActorSubstateCID(rt.IpldPut(st.Impl()))
	h.Release(checkCID)
}
func UpdateRelease(rt Runtime, h vmr.ActorStateHandle, st State) {
	newCID := actor.ActorSubstateCID(rt.IpldPut(st.Impl()))
	h.UpdateRelease(newCID)
}
func (st *StoragePowerActorState_I) CID() ipld.CID {
	panic("TODO")
}
func DeserializeState(x Bytes) State {
	panic("TODO")
}

////////////////////////////////////////////////////////////////////////////////

func (a *StoragePowerActorCode_I) AddBalance(rt Runtime) {

	var msgValue actor.TokenAmount

	// TODO: this should be enforced somewhere else
	if msgValue < 0 {
		rt.Abort("negative message value.")
	}

	// TODO: convert msgSender to MinerActorID
	var minerID addr.Address

	h, st := a.State(rt)

	currEntry, found := st.PowerTable()[minerID]

	if !found {
		// AddBalance will just fail if miner is not created before hand
		rt.Abort("minerID not found.")
	}
	currEntry.Impl().AvailableBalance_ = currEntry.AvailableBalance() + msgValue
	st.Impl().PowerTable_[minerID] = currEntry

	UpdateRelease(rt, h, st)
}

func (a *StoragePowerActorCode_I) WithdrawBalance(rt Runtime, amount actor.TokenAmount) {

	if amount < 0 {
		rt.Abort("negative amount.")
	}

	// TODO: convert msgSender to MinerActorID
	var minerID addr.Address

	h, st := a.State(rt)

	currEntry, found := st.PowerTable()[minerID]
	if !found {
		rt.Abort("minerID not found.")
	}

	if currEntry.AvailableBalance() < amount {
		rt.Abort("insufficient balance.")
	}

	currEntry.Impl().AvailableBalance_ = currEntry.AvailableBalance() - amount
	st.Impl().PowerTable_[minerID] = currEntry

	UpdateRelease(rt, h, st)

	// TODO: send funds to msgSender
}

func (a *StoragePowerActorCode_I) CreateStorageMiner(
	rt Runtime,
	ownerAddr addr.Address,
	workerAddr addr.Address,
	peerId libp2p.PeerID,
) addr.Address {

	// TODO: anything to check here?
	newMiner := &PowerTableEntry_I{
		ActivePower_:            block.StoragePower(0),
		InactivePower_:          block.StoragePower(0),
		AvailableBalance_:       actor.TokenAmount(0),
		LockedPledgeCollateral_: actor.TokenAmount(0),
	}

	// TODO: call constructor of StorageMinerActor
	// store ownerAddr and workerAddr there
	// and return StorageMinerActor address

	// TODO: minerID should be a MinerActorID
	// which is smaller than MinerAddress
	var minerID addr.Address

	h, st := a.State(rt)

	st.PowerTable()[minerID] = newMiner

	UpdateRelease(rt, h, st)

	return minerID

}

func (a *StoragePowerActorCode_I) RemoveStorageMiner(rt Runtime, address addr.Address) {

	// TODO: make explicit address type
	var minerID addr.Address

	h, st := a.State(rt)

	if (st.PowerTable()[minerID].ActivePower() + st.PowerTable()[minerID].InactivePower()) > 0 {
		rt.Abort("power still remains.")
	}

	delete(st.PowerTable(), minerID)

	UpdateRelease(rt, h, st)
}

func (a *StoragePowerActorCode_I) GetTotalPower(rt Runtime) block.StoragePower {

	totalPower := block.StoragePower(0)

	h, st := a.State(rt)

	for _, miner := range st.PowerTable() {
		totalPower = totalPower + miner.ActivePower() + miner.InactivePower()
	}

	Release(rt, h, st)

	return totalPower
}

func (a *StoragePowerActorCode_I) EnsurePledgeCollateralSatisfied(rt Runtime) InvocOutput {

	// TODO: convert msgSender to MinerActorID
	var minerID addr.Address

	h, st := a.State(rt)

	powerEntry := st._safeGetPowerEntry(rt, minerID)
	pledgeCollateralRequired := st._getPledgeCollateralReq(rt, powerEntry.ActivePower()+powerEntry.InactivePower())

	if pledgeCollateralRequired < powerEntry.LockedPledgeCollateral() {
		return rt.SuccessReturn()
	} else if pledgeCollateralRequired < (powerEntry.LockedPledgeCollateral() + powerEntry.AvailableBalance()) {
		st._lockPledgeCollateral(rt, minerID, (pledgeCollateralRequired - powerEntry.LockedPledgeCollateral()))
		return rt.SuccessReturn()
	}

	UpdateRelease(rt, h, st)

	return rt.ErrorReturn(exitcode.InsufficientPledgeCollateral)
}

// slash pledge collateral for Declared, Detected and Terminated faults
func (a *StoragePowerActorCode_I) SlashPledgeForStorageFaults(rt Runtime, affectedPower block.StoragePower, faultType sector.StorageFaultType) {

	var msgSender addr.Address // TODO replace this

	// placeholder values
	const unitDeclaredFaultSlash = 1    // placeholder
	const unitDetectedFaultSlash = 5    // placeholder
	const unitTerminatedFaultSlash = 10 // placeholder

	h, st := a.State(rt)

	switch faultType {
	case sector.DeclaredFault:
		amountToSlash := actor.TokenAmount(unitDeclaredFaultSlash * affectedPower)
		st._slashPledgeCollateral(rt, msgSender, amountToSlash)
	case sector.DetectedFault:
		amountToSlash := actor.TokenAmount(unitDetectedFaultSlash * affectedPower)
		st._slashPledgeCollateral(rt, msgSender, amountToSlash)
	case sector.TerminatedFault:
		amountToSlash := actor.TokenAmount(unitTerminatedFaultSlash * affectedPower)
		st._slashPledgeCollateral(rt, msgSender, amountToSlash)
	}

	UpdateRelease(rt, h, st)
}

// @param PowerReport with ActivePower and InactivePower
// update miner power based on the power report
func (a *StoragePowerActorCode_I) ProcessPowerReport(rt Runtime, report PowerReport) {

	// TODO: convert msgSender to MinerActorID
	var minerID addr.Address

	h, st := a.State(rt)

	powerEntry := st._safeGetPowerEntry(rt, minerID)
	powerEntry.Impl().ActivePower_ = report.ActivePower()
	powerEntry.Impl().InactivePower_ = report.InactivePower()
	st.Impl().PowerTable_[minerID] = powerEntry

	UpdateRelease(rt, h, st)
}

func (a *StoragePowerActorCode_I) ReportConsensusFault(rt Runtime, slasherAddr addr.Address, faultType ConsensusFaultType, proof []block.Block) {
	panic("TODO")

	// Use EC's IsValidConsensusFault method to validate the proof
	// slash block miner's pledge collateral
	// reward slasher

	// include ReportUncommittedPowerFault(cheaterAddr addr.Address, numSectors util.UVarint) as case
	// Quite a bit more straightforward since only called by the cron actor (ie publicly verified)
	// slash cheater pledge collateral accordingly based on num sectors faulted

}

// Surprise is in the storage power actor because it is a singleton actor and surprise helps miners maintain power
// TODO: add Surprise to the cron actor
func (a *StoragePowerActorCode_I) Surprise(rt Runtime) {

	PROVING_PERIOD := 0 // defined in storage_mining, TODO: move constants somewhere else
	SURPRISE_CHALLENGE_FREQUENCY := 0

	// sample the actor addresses
	h, st := a.State(rt)

	randomness := rt.Randomness(rt.CurrEpoch(), 0)
	challengeCount := math.Ceil(float64(SURPRISE_CHALLENGE_FREQUENCY*len(st.PowerTable())) / float64(PROVING_PERIOD))
	surprisedMiners := st._sampleMinersToSurprise(rt, int(challengeCount), randomness)

	UpdateRelease(rt, h, st)

	// now send the messages
	for _, addr := range surprisedMiners {
		// For each miner here check if they should be challenged and send message
		// rt.SendMessage(addr, ...)
		panic(addr)
	}
}

func (a *StoragePowerActorCode_I) InvokeMethod(rt Runtime, method actor.MethodNum, params actor.MethodParams) InvocOutput {
	panic("TODO")
}
