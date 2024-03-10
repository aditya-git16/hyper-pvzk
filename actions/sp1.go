package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
	"github.com/ava-labs/hypersdk/chain"
	"github.com/ava-labs/hypersdk/codec"
	"github.com/ava-labs/hypersdk/consts"
	"github.com/ava-labs/hypersdk/state"
	"github.com/ava-labs/hypersdk/utils"
	mconsts "github.com/sausaging/hyper-pvzk/consts"
	"github.com/sausaging/hyper-pvzk/requester"
	"github.com/sausaging/hyper-pvzk/storage"
)

var _ chain.Action = (*SP1)(nil)

const elfValType = 1

type SP1 struct {
	ImageID      ids.ID `json:"image_id"`
	ProofValType uint64 `json:"proof_val_type"`
}

type SP1RequestArgs struct {
	ELFFilePath   string `json:"elf_file_path"`
	ProofFilePath string `json:"proof_file_path"`
}

type SP1ReplyArgs struct {
	IsValid bool `json:"is_valid"`
}

func (*SP1) GetTypeID() uint8 {
	return mconsts.SP1ID
}

func (s *SP1) StateKeys(actor codec.Address, txID ids.ID) state.Keys {
	return state.Keys{
		string(storage.DeployKey(s.ImageID, elfValType)):     state.All, // ELF key
		string(storage.DeployKey(s.ImageID, s.ProofValType)): state.All, // Proof key
	}
}

func (*SP1) StateKeysMaxChunks() []uint16 {
	return []uint16{consts.MaxUint16, consts.MaxUint16}
}

func (*SP1) OutputsWarpMessage() bool {
	return false
}

func (*SP1) MaxComputeUnits(chain.Rules) uint64 {
	return SP1ComputeUnits
}

func (s SP1) Size() int {
	return consts.IDLen + consts.Uint64Len
}

func (s *SP1) Marshal(p *codec.Packer) {
	p.PackID(s.ImageID)
	p.PackUint64(uint64(s.ProofValType))
}

func UnmarshalSP1(p *codec.Packer, _ *warp.Message) (chain.Action, error) {
	var sp1 SP1
	p.UnpackID(true, &sp1.ImageID)
	sp1.ProofValType = p.UnpackUint64(true)

	return &sp1, nil
}

func (*SP1) ValidRange(chain.Rules) (int64, int64) {
	// Returning -1, -1 means that the action is always valid.
	return -1, -1
}

func (s *SP1) Execute(
	ctx context.Context,
	rules chain.Rules,
	mu state.Mutable,
	_ int64,
	actor codec.Address,
	txID ids.ID,
	_ bool,
) (bool, uint64, []byte, *warp.UnsignedMessage, error) {

	imageID := s.ImageID
	valType := s.ProofValType
	elfBytes, err := storage.GetDeployType(ctx, mu, imageID, elfValType)
	if err != nil {
		return false, 1000, utils.ErrBytes(fmt.Errorf("%s: can't get elf from state", err)), nil, nil
	}

	proofJsonBytes, err := storage.GetDeployType(ctx, mu, imageID, valType)
	if err != nil {
		return false, 2000, utils.ErrBytes(fmt.Errorf("%s: can't get proof at type %d from state", err, valType)), nil, nil
	}
	// store files to disk
	elfFilePath := requester.BASEFILEPATH + "sp1" + imageID.String() + txID.String() + strconv.Itoa(int(elfValType)) + ".json"
	proofFilePath := requester.BASEFILEPATH + "sp1" + imageID.String() + txID.String() + strconv.Itoa(int(valType)) + ".json"

	if err := WriteFile(elfFilePath, elfBytes); err != nil {
		return false, 3000, utils.ErrBytes(fmt.Errorf("%s: can't write elf to disk", err)), nil, nil
	}
	if err := WriteFile(proofFilePath, proofJsonBytes); err != nil {
		return false, 4000, utils.ErrBytes(fmt.Errorf("%s: can't write proof to disk", err)), nil, nil
	}

	cli, uri := requester.GetRequesterInstance(rules)
	endPointUri := uri + requester.SP1ENDPOINT
	sp1Args := SP1RequestArgs{
		ELFFilePath:   elfFilePath,
		ProofFilePath: proofFilePath,
	}

	jsonData, err := json.Marshal(sp1Args)
	if err != nil {
		return false, 5000, utils.ErrBytes(fmt.Errorf("%s: can't marshal json", err)), nil, nil
	}

	req, err := requester.NewRequest(endPointUri, jsonData)
	if err != nil {
		return false, 6000, utils.ErrBytes(fmt.Errorf("%s: can't request http", err)), nil, nil
	}

	resp, err := cli.Do(req)
	if err != nil {
		return false, 7000, utils.ErrBytes(fmt.Errorf("%s: can't do request", err)), nil, nil
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, 8000, utils.ErrBytes(fmt.Errorf("%s: can't decode client response", err)), nil, nil
	}
	reply := new(SP1ReplyArgs)
	err = json.Unmarshal(body, reply)
	if err != nil {
		return false, 8000, utils.ErrBytes(fmt.Errorf("%s: can't decode client response", err)), nil, nil
	}

	return reply.IsValid, 8000, nil, nil, nil
}
