package filcrypto

import util "github.com/filecoin-project/specs/util"

func (self *VRFResult_I) Verify(input util.Bytes, pk VRFPublicKey) bool {
	panic("TODO")
}

func (self *VRFResult_I) ValidateSyntax() bool {
	panic("TODO")
}

func (self *VDFResult_I) Verify(input util.Bytes) bool {
	panic("TODO")
}

func (self *VDFResult_I) ValidateSyntax() bool {
	panic("TODO")
}
