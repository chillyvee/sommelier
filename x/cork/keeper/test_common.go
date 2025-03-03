package keeper

import (
	"bytes"
	"testing"
	"time"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	ccodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	ccrypto "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/capability"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	"github.com/cosmos/cosmos-sdk/x/distribution"
	distrclient "github.com/cosmos/cosmos-sdk/x/distribution/client"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/evidence"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govtypesv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/cosmos-sdk/x/mint"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	paramsproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/cosmos-sdk/x/upgrade"
	upgradeclient "github.com/cosmos/cosmos-sdk/x/upgrade/client"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/peggyjv/gravity-bridge/module/v4/x/gravity"
	gravitykeeper "github.com/peggyjv/gravity-bridge/module/v4/x/gravity/keeper"
	gravitytypes "github.com/peggyjv/gravity-bridge/module/v4/x/gravity/types"
	"github.com/peggyjv/sommelier/v7/x/cork/types"
	pubsubkeeper "github.com/peggyjv/sommelier/v7/x/pubsub/keeper"
	pubsubtypes "github.com/peggyjv/sommelier/v7/x/pubsub/types"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	dbm "github.com/tendermint/tm-db"
)

var (
	// ModuleBasics is a mock module basic manager for testing
	ModuleBasics = module.NewBasicManager(
		auth.AppModuleBasic{},
		genutil.AppModuleBasic{},
		bank.AppModuleBasic{},
		capability.AppModuleBasic{},
		staking.AppModuleBasic{},
		mint.AppModuleBasic{},
		distribution.AppModuleBasic{},
		gov.NewAppModuleBasic(
			[]govclient.ProposalHandler{paramsclient.ProposalHandler, distrclient.ProposalHandler, upgradeclient.LegacyProposalHandler, upgradeclient.LegacyCancelProposalHandler},
		),
		params.AppModuleBasic{},
		crisis.AppModuleBasic{},
		slashing.AppModuleBasic{},
		upgrade.AppModuleBasic{},
		evidence.AppModuleBasic{},
		vesting.AppModuleBasic{},
		gravity.AppModuleBasic{},
	)

	// Ensure that StakingKeeperMock implements required interface
	_ types.StakingKeeper = &StakingKeeperMock{}
)

var (
	// ConsPrivKeys generate ed25519 ConsPrivKeys to be used for validator operator keys
	ConsPrivKeys = []ccrypto.PrivKey{
		ed25519.GenPrivKey(),
		ed25519.GenPrivKey(),
		ed25519.GenPrivKey(),
		ed25519.GenPrivKey(),
		ed25519.GenPrivKey(),
	}

	// ConsPubKeys holds the consensus public keys to be used for validator operator keys
	ConsPubKeys = []ccrypto.PubKey{
		ConsPrivKeys[0].PubKey(),
		ConsPrivKeys[1].PubKey(),
		ConsPrivKeys[2].PubKey(),
		ConsPrivKeys[3].PubKey(),
		ConsPrivKeys[4].PubKey(),
	}

	// AccPrivKeys generate secp256k1 pubkeys to be used for account pub keys
	AccPrivKeys = []ccrypto.PrivKey{
		secp256k1.GenPrivKey(),
		secp256k1.GenPrivKey(),
		secp256k1.GenPrivKey(),
		secp256k1.GenPrivKey(),
		secp256k1.GenPrivKey(),
	}

	// AccPubKeys holds the pub keys for the account keys
	AccPubKeys = []ccrypto.PubKey{
		AccPrivKeys[0].PubKey(),
		AccPrivKeys[1].PubKey(),
		AccPrivKeys[2].PubKey(),
		AccPrivKeys[3].PubKey(),
		AccPrivKeys[4].PubKey(),
	}

	// AccAddrs holds the sdk.AccAddresses
	AccAddrs = []sdk.AccAddress{
		sdk.AccAddress(AccPubKeys[0].Address()),
		sdk.AccAddress(AccPubKeys[1].Address()),
		sdk.AccAddress(AccPubKeys[2].Address()),
		sdk.AccAddress(AccPubKeys[3].Address()),
		sdk.AccAddress(AccPubKeys[4].Address()),
	}

	// ValAddrs holds the sdk.ValAddresses
	ValAddrs = []sdk.ValAddress{
		sdk.ValAddress(AccPubKeys[0].Address()),
		sdk.ValAddress(AccPubKeys[1].Address()),
		sdk.ValAddress(AccPubKeys[2].Address()),
		sdk.ValAddress(AccPubKeys[3].Address()),
		sdk.ValAddress(AccPubKeys[4].Address()),
	}

	// TODO: generate the eth priv keys here and
	// derive the address from them.

	// EthAddrs holds etheruem addresses
	EthAddrs = []gethcommon.Address{
		gethcommon.BytesToAddress(bytes.Repeat([]byte{byte(1)}, 20)),
		gethcommon.BytesToAddress(bytes.Repeat([]byte{byte(2)}, 20)),
		gethcommon.BytesToAddress(bytes.Repeat([]byte{byte(3)}, 20)),
		gethcommon.BytesToAddress(bytes.Repeat([]byte{byte(4)}, 20)),
		gethcommon.BytesToAddress(bytes.Repeat([]byte{byte(5)}, 20)),
	}

	// TokenContractAddrs holds example token contract addresses
	TokenContractAddrs = []string{
		"0x6b175474e89094c44da98b954eedeac495271d0f", // DAI
		"0x0bc529c00c6401aef6d220be8c6ea1667f6ad93e", // YFI
		"0x1f9840a85d5af5bf1d1762f925bdaddc4201f984", // UNI
		"0xc00e94cb662c3520282e6f5717214004a7f26888", // COMP
		"0xc011a73ee8576fb46f5e1c5751ca3b9fe0af2a6f", // SNX
	}

	// InitTokens holds the number of tokens to initialize an account with
	InitTokens = sdk.TokensFromConsensusPower(110, sdk.DefaultPowerReduction)

	// InitCoins holds the number of coins to initialize an account with
	InitCoins = sdk.NewCoins(sdk.NewCoin(TestingStakeParams.BondDenom, InitTokens))

	// StakingAmount holds the staking power to start a validator with
	StakingAmount = sdk.TokensFromConsensusPower(10, sdk.DefaultPowerReduction)

	// StakingCoins holds the staking coins to start a validator with
	StakingCoins = sdk.NewCoins(sdk.NewCoin(TestingStakeParams.BondDenom, StakingAmount))

	// TestingStakeParams is a set of staking params for testing
	TestingStakeParams = stakingtypes.Params{
		UnbondingTime:     100,
		MaxValidators:     10,
		MaxEntries:        10,
		HistoricalEntries: 10000,
		BondDenom:         "stake",
	}

	// TestingcorkParams is a set of gravity params for testing
	TestingcorkParams = types.Params{
		VoteThreshold: sdk.MustNewDecFromStr(corkVoteThresholdStr),
	}
)

// TestInput stores the various keepers required to test gravity
// TODO This file is mostly unused. Ask Eric/Collin about whether it's needed.
type TestInput struct {
	corkKeeper     Keeper
	GravityKeeper  gravitykeeper.Keeper
	AccountKeeper  authkeeper.AccountKeeper
	StakingKeeper  stakingkeeper.Keeper
	SlashingKeeper slashingkeeper.Keeper
	DistKeeper     distrkeeper.Keeper
	BankKeeper     bankkeeper.BaseKeeper
	GovKeeper      govkeeper.Keeper
	Context        sdk.Context
	Marshaler      codec.Codec
	LegacyAmino    *codec.LegacyAmino
}

// CreateTestEnv creates the keeper testing environment for gravity
func CreateTestEnv(t *testing.T) TestInput {
	t.Helper()

	// Initialize store keys
	keyGravity := sdk.NewKVStoreKey(gravitytypes.StoreKey)
	keyPubsub := sdk.NewKVStoreKey(pubsubtypes.StoreKey)
	keyAcc := sdk.NewKVStoreKey(authtypes.StoreKey)
	keyStaking := sdk.NewKVStoreKey(stakingtypes.StoreKey)
	keyBank := sdk.NewKVStoreKey(banktypes.StoreKey)
	keyDistro := sdk.NewKVStoreKey(distrtypes.StoreKey)
	keyParams := sdk.NewKVStoreKey(paramstypes.StoreKey)
	tkeyParams := sdk.NewTransientStoreKey(paramstypes.TStoreKey)
	keyGov := sdk.NewKVStoreKey(govtypes.StoreKey)
	keySlashing := sdk.NewKVStoreKey(slashingtypes.StoreKey)
	keycork := sdk.NewKVStoreKey(types.StoreKey)

	// Initialize memory database and mount stores on it
	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(keyGravity, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyAcc, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyParams, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyStaking, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyBank, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyDistro, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(tkeyParams, storetypes.StoreTypeTransient, db)
	ms.MountStoreWithDB(keyGov, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keySlashing, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keycork, storetypes.StoreTypeIAVL, db)
	err := ms.LoadLatestVersion()
	require.Nil(t, err)

	// Create sdk.Context
	ctx := sdk.NewContext(ms, tmproto.Header{
		Height: 1234567,
		Time:   time.Date(2020, time.April, 22, 12, 0, 0, 0, time.UTC),
	}, false, log.TestingLogger())

	cdc := MakeTestCodec()
	marshaler := MakeTestMarshaler()

	paramsKeeper := paramskeeper.NewKeeper(marshaler, cdc, keyParams, tkeyParams)
	paramsKeeper.Subspace(authtypes.ModuleName)
	paramsKeeper.Subspace(banktypes.ModuleName)
	paramsKeeper.Subspace(stakingtypes.ModuleName)
	paramsKeeper.Subspace(distrtypes.ModuleName)
	paramsKeeper.Subspace(govtypes.ModuleName)
	paramsKeeper.Subspace(types.DefaultParamspace)
	paramsKeeper.Subspace(slashingtypes.ModuleName)
	paramsKeeper.Subspace(gravitytypes.ModuleName)

	// this is also used to initialize module accounts for all the map keys
	maccPerms := map[string][]string{
		authtypes.FeeCollectorName:     nil,
		distrtypes.ModuleName:          nil,
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
		govtypes.ModuleName:            {authtypes.Burner},
		types.ModuleName:               {authtypes.Minter, authtypes.Burner},
	}

	accountKeeper := authkeeper.NewAccountKeeper(
		marshaler,
		keyAcc, // target store
		getSubspace(paramsKeeper, authtypes.ModuleName),
		authtypes.ProtoBaseAccount, // prototype
		maccPerms,
		"stake",
	)

	blockedAddr := make(map[string]bool, len(maccPerms))
	for acc := range maccPerms {
		blockedAddr[authtypes.NewModuleAddress(acc).String()] = true
	}

	bankKeeper := bankkeeper.NewBaseKeeper(
		marshaler,
		keyBank,
		accountKeeper,
		getSubspace(paramsKeeper, banktypes.ModuleName),
		blockedAddr,
	)
	bankKeeper.SetParams(ctx, banktypes.Params{DefaultSendEnabled: true})

	stakingKeeper := stakingkeeper.NewKeeper(marshaler, keyStaking, accountKeeper, bankKeeper, getSubspace(paramsKeeper, stakingtypes.ModuleName))
	stakingKeeper.SetParams(ctx, TestingStakeParams)

	distKeeper := distrkeeper.NewKeeper(marshaler, keyDistro, getSubspace(paramsKeeper, distrtypes.ModuleName), accountKeeper, bankKeeper, stakingKeeper, authtypes.FeeCollectorName)
	distKeeper.SetParams(ctx, distrtypes.DefaultParams())

	// set genesis items required for distribution
	distKeeper.SetFeePool(ctx, distrtypes.InitialFeePool())

	// total supply to track this
	totalSupply := sdk.NewCoins(sdk.NewInt64Coin("stake", 100000000))

	// set up initial accounts
	for name, perms := range maccPerms {
		mod := authtypes.NewEmptyModuleAccount(name, perms...)
		if name == stakingtypes.NotBondedPoolName {
			require.NoError(t, fundModAccount(ctx, bankKeeper, mod.GetName(), totalSupply))
		} else if name == distrtypes.ModuleName {
			// some big pot to pay out
			amt := sdk.NewCoins(sdk.NewInt64Coin("stake", 500000))
			require.NoError(t, fundModAccount(ctx, bankKeeper, mod.GetName(), amt))
		}

		accountKeeper.SetModuleAccount(ctx, mod)
	}

	// receiver/sender module account maps for the bridge
	receiverModuleAccounts := map[string]string{
		authtypes.NewModuleAddress(distrtypes.ModuleName).String(): distrtypes.ModuleName,
	}
	senderModuleAccounts := receiverModuleAccounts

	stakeAddr := authtypes.NewModuleAddress(stakingtypes.BondedPoolName)
	moduleAcct := accountKeeper.GetAccount(ctx, stakeAddr)
	require.NotNil(t, moduleAcct)

	// Load default wasm config

	govRouter := govtypesv1beta1.NewRouter().
		AddRoute(paramsproposal.RouterKey, params.NewParamChangeProposalHandler(paramsKeeper)).
		AddRoute(govtypes.RouterKey, govtypesv1beta1.ProposalHandler)

	govKeeper := govkeeper.NewKeeper(
		marshaler, keyGov, getSubspace(paramsKeeper, govtypes.ModuleName).WithKeyTable(govtypesv1.ParamKeyTable()), accountKeeper, bankKeeper, stakingKeeper, govRouter, baseapp.NewMsgServiceRouter(), govtypes.DefaultConfig(),
	)

	govKeeper.SetProposalID(ctx, govtypesv1beta1.DefaultStartingProposalID)
	govKeeper.SetDepositParams(ctx, govtypesv1.DefaultDepositParams())
	govKeeper.SetVotingParams(ctx, govtypesv1.DefaultVotingParams())
	govKeeper.SetTallyParams(ctx, govtypesv1.DefaultTallyParams())

	slashingKeeper := slashingkeeper.NewKeeper(
		marshaler,
		keySlashing,
		&stakingKeeper,
		getSubspace(paramsKeeper, slashingtypes.ModuleName).WithKeyTable(slashingtypes.ParamKeyTable()),
	)

	gravityKeeper := gravitykeeper.NewKeeper(
		marshaler,
		keyGravity,
		getSubspace(paramsKeeper, gravitytypes.DefaultParamspace),
		accountKeeper,
		stakingKeeper,
		bankKeeper,
		slashingKeeper,
		distKeeper,
		sdk.DefaultPowerReduction,
		receiverModuleAccounts,
		senderModuleAccounts,
	)

	pubsubKeeper := pubsubkeeper.NewKeeper(
		marshaler,
		keyPubsub,
		getSubspace(paramsKeeper, pubsubtypes.DefaultParamspace),
		stakingKeeper,
		gravityKeeper,
	)

	stakingKeeper = *stakingKeeper.SetHooks(
		stakingtypes.NewMultiStakingHooks(
			distKeeper.Hooks(),
			slashingKeeper.Hooks(),
			gravityKeeper.Hooks(),
		),
	)

	k := NewKeeper(
		marshaler,
		keycork,
		getSubspace(paramsKeeper, types.DefaultParamspace),
		stakingKeeper,
		gravityKeeper,
		pubsubKeeper,
	)

	k.SetParams(ctx, TestingcorkParams)

	return TestInput{
		corkKeeper:     k,
		GravityKeeper:  gravityKeeper,
		AccountKeeper:  accountKeeper,
		BankKeeper:     bankKeeper,
		StakingKeeper:  stakingKeeper,
		SlashingKeeper: slashingKeeper,
		DistKeeper:     distKeeper,
		GovKeeper:      govKeeper,
		Context:        ctx,
		Marshaler:      marshaler,
		LegacyAmino:    cdc,
	}
}

// getSubspace returns a param subspace for a given module name.
func getSubspace(k paramskeeper.Keeper, moduleName string) paramstypes.Subspace {
	subspace, _ := k.GetSubspace(moduleName)
	return subspace
}

// MakeTestCodec creates a legacy amino codec for testing
func MakeTestCodec() *codec.LegacyAmino {
	var cdc = codec.NewLegacyAmino()
	auth.AppModuleBasic{}.RegisterLegacyAminoCodec(cdc)
	bank.AppModuleBasic{}.RegisterLegacyAminoCodec(cdc)
	staking.AppModuleBasic{}.RegisterLegacyAminoCodec(cdc)
	distribution.AppModuleBasic{}.RegisterLegacyAminoCodec(cdc)
	sdk.RegisterLegacyAminoCodec(cdc)
	ccodec.RegisterCrypto(cdc)
	params.AppModuleBasic{}.RegisterLegacyAminoCodec(cdc)
	//types.RegisterCodec(cdc)
	return cdc
}

// MakeTestMarshaler creates a proto codec for use in testing
func MakeTestMarshaler() codec.Codec {
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	ModuleBasics.RegisterInterfaces(interfaceRegistry)
	types.RegisterInterfaces(interfaceRegistry)
	return codec.NewProtoCodec(interfaceRegistry)
}

// MockStakingValidatorData creates mock validator data
type MockStakingValidatorData struct {
	Operator sdk.ValAddress
	Power    int64
}

// NewStakingKeeperWeightedMock creates a new mock staking keeper with some mock validator data
func NewStakingKeeperWeightedMock(t ...MockStakingValidatorData) *StakingKeeperMock {
	r := &StakingKeeperMock{
		BondedValidators: make([]stakingtypes.Validator, len(t)),
		ValidatorPower:   make(map[string]int64, len(t)),
	}

	for i, a := range t {
		pk, err := codectypes.NewAnyWithValue(ed25519.GenPrivKey().PubKey())
		if err != nil {
			panic(err)
		}
		r.BondedValidators[i] = stakingtypes.Validator{
			ConsensusPubkey: pk,
			OperatorAddress: a.Operator.String(),
			Status:          stakingtypes.Bonded,
		}
		r.ValidatorPower[a.Operator.String()] = a.Power
	}
	return r
}

// StakingKeeperMock is a mock staking keeper for use in the tests
type StakingKeeperMock struct {
	BondedValidators []stakingtypes.Validator
	ValidatorPower   map[string]int64
}

// GetBondedValidatorsByPower implements the interface for staking keeper required by gravity
func (s *StakingKeeperMock) GetBondedValidatorsByPower(ctx sdk.Context) []stakingtypes.Validator {
	return s.BondedValidators
}

// GetLastValidatorPower implements the interface for staking keeper required by gravity
func (s *StakingKeeperMock) GetLastValidatorPower(ctx sdk.Context, operator sdk.ValAddress) int64 {
	v, ok := s.ValidatorPower[operator.String()]
	if !ok {
		panic("unknown address")
	}
	return v
}

// GetLastTotalPower implements the interface for staking keeper required by gravity
func (s *StakingKeeperMock) GetLastTotalPower(ctx sdk.Context) (power math.Int) {
	var total int64
	for _, v := range s.ValidatorPower {
		total += v
	}
	return sdk.NewInt(total)
}

// IterateValidators staisfies the interface
func (s *StakingKeeperMock) IterateValidators(ctx sdk.Context, cb func(index int64, validator stakingtypes.ValidatorI) (stop bool)) {
	for i, val := range s.BondedValidators {
		stop := cb(int64(i), val)
		if stop {
			break
		}
	}
}

// IterateBondedValidatorsByPower staisfies the interface
func (s *StakingKeeperMock) IterateBondedValidatorsByPower(ctx sdk.Context, cb func(index int64, validator stakingtypes.ValidatorI) (stop bool)) {
	for i, val := range s.BondedValidators {
		stop := cb(int64(i), val)
		if stop {
			break
		}
	}
}

// IterateLastValidators staisfies the interface
func (s *StakingKeeperMock) IterateLastValidators(ctx sdk.Context, cb func(index int64, validator stakingtypes.ValidatorI) (stop bool)) {
	for i, val := range s.BondedValidators {
		stop := cb(int64(i), val)
		if stop {
			break
		}
	}
}

// Validator staisfies the interface
func (s *StakingKeeperMock) Validator(ctx sdk.Context, addr sdk.ValAddress) stakingtypes.ValidatorI {
	for _, val := range s.BondedValidators {
		if val.GetOperator().Equals(addr) {
			return val
		}
	}
	return nil
}

// ValidatorByConsAddr staisfies the interface
func (s *StakingKeeperMock) ValidatorByConsAddr(ctx sdk.Context, addr sdk.ConsAddress) stakingtypes.ValidatorI {
	for _, val := range s.BondedValidators {
		cons, err := val.GetConsAddr()
		if err != nil {
			panic(err)
		}
		if cons.Equals(addr) {
			return val
		}
	}
	return nil
}

func (s *StakingKeeperMock) GetParams(ctx sdk.Context) stakingtypes.Params {
	return stakingtypes.DefaultParams()
}

func (s *StakingKeeperMock) GetValidator(ctx sdk.Context, addr sdk.ValAddress) (validator stakingtypes.Validator, found bool) {
	panic("unexpected call")
}

func (s *StakingKeeperMock) ValidatorQueueIterator(ctx sdk.Context, endTime time.Time, endHeight int64) sdk.Iterator {
	store := ctx.KVStore(sdk.NewKVStoreKey("staking"))
	return store.Iterator(stakingtypes.ValidatorQueueKey, sdk.InclusiveEndBytes(stakingtypes.GetValidatorQueueKey(endTime, endHeight)))

}

// Slash satisfies the interface
func (s *StakingKeeperMock) Slash(sdk.Context, sdk.ConsAddress, int64, int64, sdk.Dec) math.Int {
	return sdk.NewInt(0)
}

// Jail satisfies the interface
func (s *StakingKeeperMock) Jail(sdk.Context, sdk.ConsAddress) {}

// PowerReduction satisfies the interface
func (s *StakingKeeperMock) PowerReduction(sdk.Context) math.Int {
	return sdk.NewInt(0)
}

// AlwaysPanicStakingMock is a mock staking keeper that panics on usage
type AlwaysPanicStakingMock struct{}

// GetLastTotalPower implements the interface for staking keeper required by gravity
func (s AlwaysPanicStakingMock) GetLastTotalPower(ctx sdk.Context) (power math.Int) {
	panic("unexpected call")
}

// GetBondedValidatorsByPower implements the interface for staking keeper required by gravity
func (s AlwaysPanicStakingMock) GetBondedValidatorsByPower(ctx sdk.Context) []stakingtypes.Validator {
	panic("unexpected call")
}

// GetLastValidatorPower implements the interface for staking keeper required by gravity
func (s AlwaysPanicStakingMock) GetLastValidatorPower(ctx sdk.Context, operator sdk.ValAddress) int64 {
	panic("unexpected call")
}

// IterateValidators satisfies the interface
func (s AlwaysPanicStakingMock) IterateValidators(sdk.Context, func(index int64, validator stakingtypes.ValidatorI) (stop bool)) {
	panic("unexpected call")
}

// IterateBondedValidatorsByPower satisfies the interface
func (s AlwaysPanicStakingMock) IterateBondedValidatorsByPower(sdk.Context, func(index int64, validator stakingtypes.ValidatorI) (stop bool)) {
	panic("unexpected call")
}

// IterateLastValidators satisfies the interface
func (s AlwaysPanicStakingMock) IterateLastValidators(sdk.Context, func(index int64, validator stakingtypes.ValidatorI) (stop bool)) {
	panic("unexpected call")
}

// Validator satisfies the interface
func (s AlwaysPanicStakingMock) Validator(sdk.Context, sdk.ValAddress) stakingtypes.ValidatorI {
	panic("unexpected call")
}

// ValidatorByConsAddr satisfies the interface
func (s AlwaysPanicStakingMock) ValidatorByConsAddr(sdk.Context, sdk.ConsAddress) stakingtypes.ValidatorI {
	panic("unexpected call")
}

// Slash satisfies the interface
func (s AlwaysPanicStakingMock) Slash(sdk.Context, sdk.ConsAddress, int64, int64, sdk.Dec) {
	panic("unexpected call")
}

// Jail satisfies the interface
func (s AlwaysPanicStakingMock) Jail(sdk.Context, sdk.ConsAddress) {
	panic("unexpected call")
}

// PowerReduction satisfies the interface
func (s AlwaysPanicStakingMock) PowerReduction(sdk.Context) math.Int {
	panic("unexpected call")
}

func fundModAccount(ctx sdk.Context, bankKeeper gravitytypes.BankKeeper, recipientMod string, amounts sdk.Coins) error {
	if err := bankKeeper.MintCoins(ctx, types.ModuleName, amounts); err != nil {
		return err
	}

	return bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, recipientMod, amounts)
}
