package storage_mining

import sector "github.com/filecoin-project/specs/systems/filecoin_mining/sector"
import block "github.com/filecoin-project/specs/systems/filecoin_blockchain/struct/block"
import poster "github.com/filecoin-project/specs/systems/filecoin_mining/storage_proving/poster"

// If a Post is missed (either due to faults being not declared on time or
// because the miner run out of time, every sector is reported as failing
// for the current proving period.
func (sm *StorageMinerActor_I) CronAction() {
	// TODO: detemine how much pledge collateral to slash
	// TODO: SendMessage(spa.SlashPledgeCollateral)
	var powerUpdate uint64 = 0

	for sectorNo, _ := range sm.Sectors() {
		state := sm.SectorStates()[sectorNo]
		switch state.StateNumber {
		case SectorCommittedStateNo:
			// SlashStorageDealCollateral
			// SendMessage(sma.slashStorageDealCollateral(sc.DealIDs()))
			sm.failSector(sectorNo)
		case SectorRecoveringStateNo:
			// SlashStorageDealCollateral
			// SendMessage(sma.slashStorageDealCollateral(sc.DealIDs()))
			sm.failSector(sectorNo)
			newFaultCount := sm.SectorStates()[sectorNo].FaultCount
			if newFaultCount > MAX_CONSECUTIVE_FAULT_COUNT {
				// TODO: heavy penalization: slash pledge collateral and delete sector
				// TODO: SendMessage(SPA.SlashPledgeCollateral)
				sm.clearSector(sectorNo)
			}
		case SectorActiveStateNo:
			// SlashStorageDealCollateral
			// SendMessage(sma.slashStorageDealCollateral(sc.DealIDs()))
			sm.failSector(sectorNo)
			powerUpdate = powerUpdate - sm.Info().SectorSize()
		case SectorFailingStateNo:
			sm.checkMFC(sectorNo)
		default:
			// TODO: proper failure
			panic("Invalid sector state in CronAction")
		}
	}

	// Reset Proving Period and report power updates
	// sm.ProvingPeriodEnd_ = PROVING_PERIOD_TIME
	// SendMessage(sma.UpdatePower(powerUpdate))
}

func (sm *StorageMinerActor_I) verifyPoStSubmission(postSubmission poster.PoStSubmission) bool {
	// 1. A proof must be submitted after the postRandomness for this proving
	// period is on chain
	// if rt.ChainEpoch < sm.ProvingPeriodEnd - challengeTime {
	//   panic("too early")
	// }

	// 2. A proof must be a valid snark proof with the correct public inputs
	// 2.1 Get randomness from the chain at the right epoch
	// postRandomness := rt.Randomness(postSubmission.Epoch, 0)
	// 2.2 Generate the set of challenges
	// challenges := GenerateChallengesForPoSt(r, keys(sm.Sectors))
	// 2.3 Verify the PoSt Proof
	// verifyPoSt(challenges, TODO)

	panic("TODO")
	return true
}

// move Sector from Active/Committed/Recovering/Failing
// into Cleared State which means deleting the Sector from state
// remove SectorNumber from all states on chain
func (sm *StorageMinerActor_I) clearSector(sectorNo sector.SectorNumber) {
	delete(sm.Sectors(), sectorNo)
	delete(sm.SectorStates(), sectorNo)
	sm.ProvingSet_.Remove(sectorNo)
}

// move Sector from Committed/Recovering into Active State
// reset FaultCount to zero
func (sm *StorageMinerActor_I) activateSector(sectorNo sector.SectorNumber) {
	sm.SectorStates()[sectorNo] = SectorActive()
}

// move Sector from Active/Committed/Recovering into Failing State
// and increment FaultCount
func (sm *StorageMinerActor_I) failSector(sectorNo sector.SectorNumber) {
	newFaultCount := sm.SectorStates()[sectorNo].FaultCount + 1
	sm.ProvingSet_.Remove(sectorNo)
	sm.SectorStates()[sectorNo] = SectorFailing(newFaultCount)
}

// increment FaultCount and check if greater than MaxFaultCount
// move from Failing to Cleared State if so
func (sm *StorageMinerActor_I) checkMFC(sectorNo sector.SectorNumber) {
	newFaultCount := sm.SectorStates()[sectorNo].FaultCount + 1
	if newFaultCount > MAX_CONSECUTIVE_FAULT_COUNT {
		// TODO: heavy penalization: slash pledge collateral and delete sector
		// TODO: SendMessage(SPA.SlashPledgeCollateral)
		sm.clearSector(sectorNo)
	} else {
		// increment FaultCount
		// TODO: SendMessage(sma.SlashStorageDealCollateral)
		sm.SectorStates()[sectorNo] = SectorFailing(newFaultCount)
	}
}

// Decision is to currently account for power based on sector
// with at least one active deals and deals cannot be updated
// an alternative proposal is to account for power based on active deals
// an improvement proposal is to allow storage deal update in a sector

// TODO: decide whether declared faults sectors should be
// penalized in the same way as undeclared sectors and how

// SubmitPoSt Workflow:
// - Verify PoSt Submission
// - Process ProvingSet.GetOnes()
//   - State Transitions
//     - Committed -> Active and credit power
//     - Recovering -> Active and credit power
//   - Process Active Sectors (pay miners)
// - Process ProvingSet.GetZeros()
//	   - increment FaultCount
//     - clear Sector and slash pledge collateral if count > MAX_CONSECUTIVE_FAULT_COUNT
// - Process Expired Sectors (settle deals and return storage collateral to miners)
//     - State Transition
//       - Failing / Recovering / Active / Committed -> Cleared
//     - Remove SectorNumber from Sectors, SectorStates, ProvingSet
func (sm *StorageMinerActor_I) SubmitPoSt(postSubmission poster.PoStSubmission) {
	// Verify correct PoSt Submission
	isPoStVerified := sm.verifyPoStSubmission(postSubmission)
	if !isPoStVerified {
		// no state transition, just error out and miner should submitPoSt again
		// TODO: proper failure
		panic("TODO")
	}

	// The proof is verified, process ProvingSet.GetOnes():
	// ProvingSet.GetOnes() contains SectorCommitted, SectorActive, SectorRecovering
	// ProvingSet itself does not store states, states are all stored in SectorStates
	var powerUpdate uint64 = 0

	for _, sectorNo := range sm.ProvingSet_.GetOnes() {
		state := sm.SectorStates()[sectorNo]
		switch state.StateNumber {
		case SectorCommittedStateNo:
			sm.activateSector(sectorNo)
			powerUpdate = powerUpdate + sm.Info().SectorSize()
		case SectorRecoveringStateNo:
			// Note: SectorState.FaultCount is also reset to zero here
			sm.activateSector(sectorNo)
			powerUpdate = powerUpdate + sm.Info().SectorSize()
		case SectorActiveStateNo:
			// Process payment in all active deals
			// Note: this must happen before marking sectors as expired.
			// TODO: Pay miner in a single batch message
			// SendMessage(sma.ProcessStorageDealsPayment(sm.Sectors()[sectorNumber].DealIDs()))
		default:
			// TODO: proper failure
			panic("Invalid sector state in ProvingSet.GetOnes()")
		}
	}

	// Process ProvingSet.GetZeros()
	// ProvingSet.GetZeros() contains SectorFailing
	// SectorRecovering is Proving and hence will not be in GetZeros()
	// heavy penalty if Failing for more than or equal to MAX_CONSECUTIVE_FAULT_COUNT
	// otherwise increment FaultCount in SectorStates()
	for _, sectorNo := range sm.ProvingSet_.GetZeros() {
		state := sm.SectorStates()[sectorNo]
		switch state.StateNumber {
		case SectorFailingStateNo:
			sm.checkMFC(sectorNo)
		default:
			// TODO: proper failure
			panic("Invalid sector state in ProvingSet.GetZeros")
		}
	}

	// Process Expiration
	// State change: Active / Committed / Recovering / Failing -> Cleared
	var currEpoch block.ChainEpoch // TODO: replace this with rt.State().Epoch()

	expirationPeek := sm.SectorExpirationQueue().Peek().Expiration()
	for expirationPeek <= currEpoch {
		expiredSectorNo := sm.SectorExpirationQueue().Pop().SectorNumber()

		state := sm.SectorStates()[expiredSectorNo]
		// sc := sm.Sectors()[expiredSectorNo]
		switch state.StateNumber {
		// case SectorCommittedStateNo:
		// return storage deal collateral
		// delete SectorNumber from Sectors, SectorStates, ProvingSet
		// SendMessage(sma.SettleExpiredDeals(sc.DealIDs()))
		// sm.clearSector(expiredSectorNo)
		// case SectorRecoveringStateNo:
		// SendMessage(sma.SettleExpiredDeals(sc.DealIDs()))
		// sm.clearSector(expiredSectorNo)
		case SectorActiveStateNo:
			// Note: in order to verify if something was stored in the past, one must
			// scan the chain. SectorNumber can be re-used.

			// Settle deals
			// SendMessage(sma.SettleExpiredDeals(sc.DealIDs()))
			sm.clearSector(expiredSectorNo)
			powerUpdate = powerUpdate - sm.Info().SectorSize()

		case SectorFailingStateNo:
			// TODO: check if there is any fault that we should handle here
			// If a SectorFailing Expires, return remaining StorageDealCollateral and remove sector
			// SendMessage(sma.SettleExpiredDeals(sc.DealIDs()))
			sm.clearSector(expiredSectorNo)
		default:
			// TODO: proper failure
			panic("Invalid sector state in SectorExpirationQueue")
		}

		expirationPeek = sm.SectorExpirationQueue().Peek().Expiration()

	}

	// Reset Proving Period and report power updates
	// sm.ProvingPeriodEnd_ = PROVING_PERIOD_TIME
	// SendMessage(sma.UpdatePower(powerUpdate))

	// Return PledgeCollateral
	// TODO: SendMessage(spa.Depledge)
}

// RecoverFaults checks if miners have sufficent collateral
// and adds SectorFailing into SectorRecovering
// - State Transition
//   - Failing -> Recovering with the same FaultCount
// - Add SectorNumber to ProvingSet
// Note that power is not updated until it is active
func (sm *StorageMinerActor_I) RecoverFaults(recoveringSet sector.CompactSectorSet) {
	// for all SectorNumber marked as recovering by recoveringSet
	for _, sectorNo := range recoveringSet.GetOnes() {
		state := sm.SectorStates()[sectorNo]
		switch state.StateNumber {
		case SectorFailingStateNo:
			// Check if miners have sufficient balances in sma
			// SendMessage(sma.PublishStorageDeals) or sma.ResumeStorageDeals?
			// throw if miner cannot cover StorageDealCollateral

			// copy over the same FaultCount
			sm.SectorStates()[sectorNo] = SectorRecovering(state.FaultCount)
			sm.ProvingSet_.Add(sectorNo)
		default:
			// TODO: proper failure
			panic("Invalid sector state in RecoverFaults")
		}
	}
}

// DeclareFaults penalizes miners (slashStorageDealCollateral and suspendPower)
// TODO: decide how much storage collateral to slash
// - State Transition
//   - Active / Commited / Recovering -> Failing
// - Update SectorStates
// - Remove Active / Commited / Recovering from ProvingSet
func (sm *StorageMinerActor_I) DeclareFaults(faultSet sector.CompactSectorSet) {

	var powerUpdate uint64 = 0

	// get all SectorNumber marked as Failing by faultSet
	for _, sectorNo := range faultSet.GetOnes() {
		state := sm.SectorStates()[sectorNo]
		// sc := sm.Sectors()[sectorNo]

		switch state.StateNumber {
		case SectorActiveStateNo:
			// SlashStorageDealCollateral
			// SendMessage(sma.slashStorageDealCollateral(sc.DealIDs()))
			sm.failSector(sectorNo)
			powerUpdate = powerUpdate - sm.Info().SectorSize()
		case SectorCommittedStateNo:
			// SlashStorageDealCollateral
			// SendMessage(sma.slashStorageDealCollateral(sc.DealIDs()))
			sm.failSector(sectorNo)
		case SectorRecoveringStateNo:
			// SlashStorageDealCollateral
			// SendMessage(sma.slashStorageDealCollateral(sc.DealIDs()))
			sm.failSector(sectorNo)
		default:
			// TODO: proper failure
			panic("Invalid sector state in DeclareFaults")
		}
	}

	// Suspend power
	// SendMessage(spa.UpdatePower(powerUpdate))
}

func (sm *StorageMinerActor_I) verifySeal(onChainInfo sector.OnChainSealVerifyInfo) bool {
	// TODO: verify seal @nicola
	// TODO: get var sealRandomness sector.SealRandomness from onChainInfo.Epoch
	// TODO: sm.verifySeal(sectorID SectorID, comm sector.OnChainSealVerifyInfo, proof SealProof)
	panic("TODO")
	return true
}

func (sm *StorageMinerActor_I) checkIfSectorExists(sectorNo sector.SectorNumber) bool {
	_, sectorExists := sm.Sectors()[sectorNo]
	if sectorExists {
		return true
	}
	return false
}

// Currently deals must be posted on chain via sma.PublishStorageDeals before CommitSector
// TODO(optimization): CommitSector could contain a list of deals that are not published yet.
func (sm *StorageMinerActor_I) CommitSector(onChainInfo sector.OnChainSealVerifyInfo) {
	isSealVerified := sm.verifySeal(onChainInfo)
	if !isSealVerified {
		// TODO: proper failure
		panic("Seal is not verified")
	}

	sectorExists := sm.checkIfSectorExists(onChainInfo.SectorNumber())
	if sectorExists {
		//TODO: proper failure
		panic("Sector already exists")
	}

	// determine lastDealExpiration from sma
	// TODO: proper onchain transaction
	// lastDealExpiration := SendMessage(sma, GetLastDealExpirationFromDealIDs(onChainInfo.DealIDs()))
	var lastDealExpiration block.ChainEpoch

	// add sector expiration to SectorExpirationQueue
	sm.SectorExpirationQueue().Add(&SectorExpirationQueuItem_I{
		SectorNumber_: onChainInfo.SectorNumber(),
		Expiration_:   lastDealExpiration,
	})

	// no need to store the proof and randomseed in the state tree
	// verify and drop, only SealCommitments{CommD, CommR, DealIDs} on chain
	// TODO: @porcuquine verifies
	sealCommitment := &sector.SealCommitment_I{
		UnsealedCID_: onChainInfo.UnsealedCID(),
		SealedCID_:   onChainInfo.SealedCID(),
		DealIDs_:     onChainInfo.DealIDs(),
		// Expiration_:  lastDealExpiration,
	}

	// add SectorNumber and SealCommitment to Sectors
	// set SectorState to SectorCommitted
	// Note that SectorNumber will only become Active at the next successful PoSt
	sm.Sectors()[onChainInfo.SectorNumber()] = sealCommitment
	sm.SectorStates()[onChainInfo.SectorNumber()] = SectorCommitted()
	sm.ProvingSet_.Add(onChainInfo.SectorNumber())

	// TODO: write state change
}
