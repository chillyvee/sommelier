package integration_tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govtypesv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	gravitytypes "github.com/peggyjv/gravity-bridge/module/v4/x/gravity/types"
	"github.com/peggyjv/sommelier/v7/app/params"
	auctiontypes "github.com/peggyjv/sommelier/v7/x/auction/types"
	axelarcorktypes "github.com/peggyjv/sommelier/v7/x/axelarcork/types"
	cellarfeestypes "github.com/peggyjv/sommelier/v7/x/cellarfees/types"
	corktypes "github.com/peggyjv/sommelier/v7/x/cork/types"
	pubsubtypes "github.com/peggyjv/sommelier/v7/x/pubsub/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/cosmos/cosmos-sdk/server"
	srvconfig "github.com/cosmos/cosmos-sdk/server/config"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	disttypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/suite"
	tmconfig "github.com/tendermint/tendermint/config"
	tmjson "github.com/tendermint/tendermint/libs/json"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
)

const (
	testDenom           = "usomm"
	initBalanceStr      = "210000000000usomm"
	minGasPrice         = "2"
	ethChainID     uint = 15
	exampleCA           = `-----BEGIN CERTIFICATE-----
MIICGzCCAaKgAwIBAgIUVYhZ4+pC7vQAf5FC6pssLk/eq5YwCgYIKoZIzj0EAwMw
RTELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoMGElu
dGVybmV0IFdpZGdpdHMgUHR5IEx0ZDAeFw0yMjAxMDUwNzIwMzFaFw0yNDAxMDUw
NzIwMzFaMEUxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYD
VQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwdjAQBgcqhkjOPQIBBgUrgQQA
IgNiAAQ3jwZd0Xe9w55UyAxRuc4F2u/LDdo7ykCZBO34neXpLR4GRRpx5VjFdHcX
WjvM9j3DnWjptb1fe7TIKSSJRmW1skWkpktOthIPhfga9jBhU4WRUDloKk1tRuiI
e8rRSlSjUzBRMB0GA1UdDgQWBBSTyTULHT9hNAA2Wg4dCtuTuIhiXTAfBgNVHSME
GDAWgBSTyTULHT9hNAA2Wg4dCtuTuIhiXTAPBgNVHRMBAf8EBTADAQH/MAoGCCqG
SM49BAMDA2cAMGQCMEd+Eg6lhStLkWEwmJJGN3Xdh9JmNsgsdff3mI3Y7UmHOB8K
HOqHGS8ApZcunRauDAIwRtgceZpkS92KuP3QOUotAH/nnCzp7X1lVzGOSTBRTVYJ
pohf4PJrfacqpi7PoXBk
-----END CERTIFICATE-----
`
)

var (
	stakeAmount, _  = sdk.NewIntFromString("100000000000")
	stakeAmountCoin = sdk.NewCoin(testDenom, stakeAmount)

	// todo(mvid): split these out into their respective tests
	gravityContract       = common.HexToAddress("0x04C89607413713Ec9775E14b954286519d836FEf")
	counterContract       = common.HexToAddress("0x0000000000000000000000000000000000000000")
	alphaERC20Contract    = common.HexToAddress("0x0000000000000000000000000000000000000000")
	betaERC20Contract     = common.HexToAddress("0x0000000000000000000000000000000000000000")
	unusedGenesisContract = common.HexToAddress("0x0000000000000000000000000000000000000001")

	alphaFeeDenom = ""
	betaFeeDenom  = ""

	// 67%
	corkVoteThreshold = sdk.NewDecWithPrec(67, 2)

	proposerMnemonic = "exit own pull hurry crawl glimpse winter service exclude same dog wrap method online practice deal lend athlete resemble chuckle culture receive autumn cinnamon"
)

func MNEMONICS() []string {
	return []string{
		"receive roof marine sure lady hundred sea enact exist place bean wagon kingdom betray science photo loop funny bargain floor suspect only strike endless",
		"march carpet enact kiss tribe plastic wash enter index lift topic riot try juice replace supreme original shift hover adapt mutual holiday manual nut",
		"assault section bleak gadget venture ship oblige pave fabric more initial april dutch scene parade shallow educate gesture lunar match patch hawk member problem",
		"say monitor orient heart super local purse cricket caution primary bring insane road expect rather help two extend own execute throw nation plunge subject",
	}
}

type IntegrationTestSuite struct {
	suite.Suite

	chain         *chain
	dockerPool    *dockertest.Pool
	dockerNetwork *dockertest.Network
	ethResource   *dockertest.Resource
	valResources  []*dockertest.Resource
	orchResources []*dockertest.Resource
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) SetupSuite() {
	s.T().Log("setting up e2e integration test suite...")

	var err error
	s.chain, err = newChain()
	s.Require().NoError(err)
	s.dockerPool, err = dockertest.NewPool("")
	s.Require().NoError(err)
	s.dockerNetwork, err = s.dockerPool.CreateNetwork(fmt.Sprintf("%s-testnet", s.chain.id))
	s.Require().NoError(err)

	s.T().Logf("starting e2e infrastructure; chain-id: %s; datadir: %s", s.chain.id, s.chain.dataDir)

	// initialization
	mnemonics := MNEMONICS()
	s.initNodesWithMnemonics(mnemonics...)
	s.initEthereumFromMnemonics(mnemonics)

	// run the eth container so that the contract addresses are available
	s.runEthContainer()

	// continue generating node genesis
	s.initGenesis()
	s.initValidatorConfigs()

	// container infrastructure
	s.runValidators()
	s.runOrchestrators()
}

func (s *IntegrationTestSuite) TearDownSuite() {
	if str := os.Getenv("E2E_SKIP_CLEANUP"); len(str) > 0 {
		skipCleanup, err := strconv.ParseBool(str)
		s.Require().NoError(err)

		if skipCleanup {
			s.T().Log("skipping teardown")
			return
		}
	}

	s.T().Log("tearing down e2e integration test suite...")

	s.Require().NoError(os.RemoveAll(s.chain.dataDir))
	s.Require().NoError(s.dockerPool.Purge(s.ethResource))

	for _, vc := range s.valResources {
		s.Require().NoError(s.dockerPool.Purge(vc))
	}

	for _, oc := range s.orchResources {
		s.Require().NoError(s.dockerPool.Purge(oc))
	}

	s.Require().NoError(s.dockerPool.RemoveNetwork(s.dockerNetwork))
}

func (s *IntegrationTestSuite) initNodes(nodeCount int) { //nolint:unused
	s.Require().NoError(s.chain.createAndInitValidators(nodeCount))
	s.Require().NoError(s.chain.createAndInitOrchestrators(nodeCount))

	// initialize a genesis file for the first validator
	val0ConfigDir := s.chain.validators[0].configDir()
	for _, val := range s.chain.validators {
		s.Require().NoError(
			addGenesisAccount(val0ConfigDir, "", initBalanceStr, val.address()),
		)
	}

	// add orchestrator accounts to genesis file
	for _, orch := range s.chain.orchestrators {
		s.Require().NoError(
			addGenesisAccount(val0ConfigDir, "", initBalanceStr, orch.address()),
		)
	}

	// add proposer account to genesis file (so it can bid)
	s.Require().NoError(
		addGenesisAccount(val0ConfigDir, "", initBalanceStr, s.chain.proposer.address()),
	)

	// copy the genesis file to the remaining validators
	for _, val := range s.chain.validators[1:] {
		err := copyFile(
			filepath.Join(val0ConfigDir, "config", "genesis.json"),
			filepath.Join(val.configDir(), "config", "genesis.json"),
		)
		s.Require().NoError(err)
	}
}

func (s *IntegrationTestSuite) initNodesWithMnemonics(mnemonics ...string) {
	s.Require().NoError(s.chain.createAndInitValidatorsWithMnemonics(mnemonics))
	s.Require().NoError(s.chain.createAndInitOrchestratorsWithMnemonics(mnemonics))
	s.Require().NoError(s.chain.createAndInitProposerWithMnemonic(proposerMnemonic))

	//initialize a genesis file for the first validator
	val0ConfigDir := s.chain.validators[0].configDir()
	for _, val := range s.chain.validators {
		// Fund the first validator with some funds to be used by auction module integration tests
		s.Require().NoError(
			addGenesisAccount(val0ConfigDir, "", initBalanceStr, val.address()),
		)
	}

	// add orchestrator accounts to genesis file
	for _, orch := range s.chain.orchestrators {
		// Fund the first orchestrator with some funds to be used by auction module integration tests
		s.Require().NoError(
			addGenesisAccount(val0ConfigDir, "", initBalanceStr, orch.address()),
		)
	}

	// add proposer account to genesis file (so it can bid)
	s.Require().NoError(
		addGenesisAccount(val0ConfigDir, "", initBalanceStr, s.chain.proposer.address()),
	)

	// copy the genesis file to the remaining validators
	for _, val := range s.chain.validators[1:] {
		err := copyFile(
			filepath.Join(val0ConfigDir, "config", "genesis.json"),
			filepath.Join(val.configDir(), "config", "genesis.json"),
		)
		s.Require().NoError(err)
	}
}

func (s *IntegrationTestSuite) initEthereum() { //nolint:unused
	// generate ethereum keys for validators add them to the ethereum genesis
	ethGenesis := EthereumGenesis{
		Difficulty: "0x400",
		GasLimit:   "0xB71B00",
		Config:     EthereumConfig{ChainID: ethChainID},
		Alloc:      make(map[string]Allocation, len(s.chain.validators)+1),
	}

	alloc := Allocation{
		Balance: "0x1337000000000000000000",
	}

	ethGenesis.Alloc["0xBf660843528035a5A4921534E156a27e64B231fE"] = alloc
	for _, val := range s.chain.validators {
		s.Require().NoError(val.generateEthereumKey())
		ethGenesis.Alloc[val.ethereumKey.address] = alloc
	}

	ethGenBz, err := json.MarshalIndent(ethGenesis, "", "  ")
	s.Require().NoError(err)

	// write out the genesis file
	s.Require().NoError(writeFile(filepath.Join(s.chain.configDir(), "eth_genesis.json"), ethGenBz))
}

func (s *IntegrationTestSuite) initEthereumFromMnemonics(mnemonics []string) {
	// generate ethereum keys for validators add them to the ethereum genesis
	ethGenesis := EthereumGenesis{
		Difficulty: "0x400",
		GasLimit:   "0xB71B00",
		Config:     EthereumConfig{ChainID: ethChainID},
		Alloc:      make(map[string]Allocation, len(s.chain.validators)+1),
	}

	alloc := Allocation{
		Balance: "0x1337000000000000000000",
	}

	ethGenesis.Alloc["0xBf660843528035a5A4921534E156a27e64B231fE"] = alloc
	for i, val := range s.chain.validators {
		s.Require().NoError(val.generateEthereumKeyFromMnemonic(mnemonics[i]))
		ethGenesis.Alloc[val.ethereumKey.address] = alloc
	}

	ethGenBz, err := json.MarshalIndent(ethGenesis, "", "  ")
	s.Require().NoError(err)

	// write out the genesis file
	s.Require().NoError(writeFile(filepath.Join(s.chain.configDir(), "eth_genesis.json"), ethGenBz))
}

func (s *IntegrationTestSuite) initGenesis() {
	serverCtx := server.NewDefaultContext()
	config := serverCtx.Config

	config.SetRoot(s.chain.validators[0].configDir())
	config.Moniker = s.chain.validators[0].moniker

	genFilePath := config.GenesisFile()
	appGenState, genDoc, err := genutiltypes.GenesisStateFromGenFile(genFilePath)
	s.Require().NoError(err)

	var bankGenState banktypes.GenesisState
	s.Require().NoError(cdc.UnmarshalJSON(appGenState[banktypes.ModuleName], &bankGenState))

	bankGenState.DenomMetadata = append(bankGenState.DenomMetadata, banktypes.Metadata{
		Description: "The native staking token of the test somm network",
		Display:     testDenom,
		Base:        testDenom,
		Name:        testDenom,
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    testDenom,
				Exponent: 0,
			},
		},
	})

	// Set up auction module with some coins to auction off
	balance := banktypes.Balance{
		Address: authtypes.NewModuleAddress(auctiontypes.ModuleName).String(),
		Coins:   sdk.NewCoins(sdk.NewCoin("gravity0x3506424f91fd33084466f402d5d97f05f8e3b4af", sdk.NewInt(5000000000))),
	}
	distBalance := banktypes.Balance{
		Address: authtypes.NewModuleAddress(disttypes.ModuleName).String(),
		Coins:   sdk.NewCoins(sdk.NewCoin(params.BaseCoinUnit, sdk.NewInt(1000000000))),
	}
	bankGenState.Balances = append(bankGenState.Balances, balance)
	bankGenState.Balances = append(bankGenState.Balances, distBalance)

	bz, err := cdc.MarshalJSON(&bankGenState)
	s.Require().NoError(err)
	appGenState[banktypes.ModuleName] = bz

	var govGenState govtypesv1beta1.GenesisState
	s.Require().NoError(cdc.UnmarshalJSON(appGenState[govtypes.ModuleName], &govGenState))

	// set short voting period to allow gov proposals in tests
	govGenState.VotingParams.VotingPeriod = time.Second * 20
	govGenState.DepositParams.MinDeposit = sdk.Coins{{Denom: testDenom, Amount: sdk.OneInt()}}
	bz, err = cdc.MarshalJSON(&govGenState)
	s.Require().NoError(err)
	appGenState[govtypes.ModuleName] = bz

	// set crisis denom
	var crisisGenState crisistypes.GenesisState
	s.Require().NoError(cdc.UnmarshalJSON(appGenState[crisistypes.ModuleName], &crisisGenState))
	crisisGenState.ConstantFee.Denom = testDenom
	bz, err = cdc.MarshalJSON(&crisisGenState)
	s.Require().NoError(err)
	appGenState[crisistypes.ModuleName] = bz

	// set staking bond denom
	var stakingGenState stakingtypes.GenesisState
	s.Require().NoError(cdc.UnmarshalJSON(appGenState[stakingtypes.ModuleName], &stakingGenState))
	stakingGenState.Params.BondDenom = testDenom
	bz, err = cdc.MarshalJSON(&stakingGenState)
	s.Require().NoError(err)
	appGenState[stakingtypes.ModuleName] = bz

	// set mint denom
	var mintGenState minttypes.GenesisState
	s.Require().NoError(cdc.UnmarshalJSON(appGenState[minttypes.ModuleName], &mintGenState))
	mintGenState.Params.MintDenom = testDenom
	mintGenState.Params.InflationMax = sdk.ZeroDec()
	mintGenState.Params.InflationMin = sdk.ZeroDec()
	mintGenState.Params.InflationRateChange = sdk.ZeroDec()
	mintGenState.Minter.Inflation = sdk.ZeroDec()
	bz, err = cdc.MarshalJSON(&mintGenState)
	s.Require().NoError(err)
	appGenState[minttypes.ModuleName] = bz

	distGenState := disttypes.DefaultGenesisState()
	s.Require().NoError(cdc.UnmarshalJSON(appGenState[minttypes.ModuleName], &mintGenState))
	distGenState.Params.CommunityTax = sdk.ZeroDec()
	distGenState.Params.BaseProposerReward = sdk.ZeroDec()
	distGenState.Params.BonusProposerReward = sdk.ZeroDec()
	distGenState.FeePool.CommunityPool = sdk.NewDecCoins(sdk.NewDecCoin(params.BaseCoinUnit, sdk.NewInt(1000000000)))
	bz, err = cdc.MarshalJSON(distGenState)
	s.Require().NoError(err)
	appGenState[disttypes.ModuleName] = bz

	var genUtilGenState genutiltypes.GenesisState
	s.Require().NoError(cdc.UnmarshalJSON(appGenState[genutiltypes.ModuleName], &genUtilGenState))

	// Add an auction for integration testing of the auction module
	alphaFeeDenom = fmt.Sprintf("gravity%s", alphaERC20Contract.Hex())
	betaFeeDenom = fmt.Sprintf("gravity%s", betaERC20Contract.Hex())
	var auctionGenState auctiontypes.GenesisState
	s.Require().NoError(cdc.UnmarshalJSON(appGenState[auctiontypes.ModuleName], &auctionGenState))
	auctionGenState.TokenPrices = append(auctionGenState.TokenPrices, &auctiontypes.TokenPrice{
		Denom:            alphaFeeDenom,
		Exponent:         6,
		UsdPrice:         sdk.MustNewDecFromStr("1.0"),
		LastUpdatedBlock: 0,
	})
	auctionGenState.TokenPrices = append(auctionGenState.TokenPrices, &auctiontypes.TokenPrice{
		Denom:            betaFeeDenom,
		Exponent:         6,
		UsdPrice:         sdk.MustNewDecFromStr("5.0"),
		LastUpdatedBlock: 0,
	})
	auctionGenState.TokenPrices = append(auctionGenState.TokenPrices, &auctiontypes.TokenPrice{
		Denom:            testDenom,
		Exponent:         6,
		UsdPrice:         sdk.MustNewDecFromStr("0.5"),
		LastUpdatedBlock: 0,
	})
	auctionGenState.Auctions = append(auctionGenState.Auctions, &auctiontypes.Auction{
		Id:                         uint32(1),
		StartingTokensForSale:      sdk.NewCoin("gravity0x3506424f91fd33084466f402d5d97f05f8e3b4af", sdk.NewInt(5000000000)),
		StartBlock:                 uint64(1),
		EndBlock:                   uint64(0),
		InitialPriceDecreaseRate:   sdk.MustNewDecFromStr("0.05"),
		CurrentPriceDecreaseRate:   sdk.MustNewDecFromStr("0.05"),
		PriceDecreaseBlockInterval: uint64(1000),
		InitialUnitPriceInUsomm:    sdk.MustNewDecFromStr("2"),
		CurrentUnitPriceInUsomm:    sdk.MustNewDecFromStr("2"),
		RemainingTokensForSale:     sdk.NewCoin("gravity0x3506424f91fd33084466f402d5d97f05f8e3b4af", sdk.NewInt(5000000000)),
		FundingModuleAccount:       cellarfeestypes.ModuleName,
		ProceedsModuleAccount:      cellarfeestypes.ModuleName,
	})
	bz, err = cdc.MarshalJSON(&auctionGenState)
	s.Require().NoError(err)
	appGenState[auctiontypes.ModuleName] = bz

	axelarcorkGenState := axelarcorktypes.DefaultGenesisState()
	s.Require().NoError(cdc.UnmarshalJSON(appGenState[axelarcorktypes.ModuleName], &axelarcorkGenState))
	axelarcorkGenState.Params = &axelarcorktypes.Params{
		Enabled:           true,
		IbcChannel:        "channel-1",
		IbcPort:           ibctransfertypes.PortID,
		GmpAccount:        "axelar1dv4u5k73pzqrxlzujxg3qp8kvc3pje7jtdvu72npnt5zhq05ejcsn5qme5",
		ExecutorAccount:   "axelar1zl3rxpp70lmte2xr6c4lgske2fyuj3hupcsvcd",
		TimeoutDuration:   uint64(6 * time.Hour),
		CorkTimeoutBlocks: 5000,
	}
	bz, err = cdc.MarshalJSON(&axelarcorkGenState)
	s.Require().NoError(err)
	appGenState[axelarcorktypes.ModuleName] = bz

	// set cellarfees gen state
	cellarfeesGenState := cellarfeestypes.DefaultGenesisState()
	s.Require().NoError(cdc.UnmarshalJSON(appGenState[cellarfeestypes.ModuleName], &cellarfeesGenState))
	cellarfeesGenState.Params = cellarfeestypes.Params{
		FeeAccrualAuctionThreshold: 2,
		RewardEmissionPeriod:       100,
		InitialPriceDecreaseRate:   sdk.MustNewDecFromStr("0.05"),
		PriceDecreaseBlockInterval: uint64(1000),
		AuctionInterval:            50,
	}
	bz, err = cdc.MarshalJSON(&cellarfeesGenState)
	s.Require().NoError(err)
	appGenState[cellarfeestypes.ModuleName] = bz

	// set pubsub gen state
	pubsubGenState := pubsubtypes.DefaultGenesisState()
	s.Require().NoError(cdc.UnmarshalJSON(appGenState[pubsubtypes.ModuleName], &pubsubGenState))
	pubsubGenState.Publishers = []*pubsubtypes.Publisher{
		{
			Address: s.chain.proposer.address().String(),
			Domain:  "example.com",
			CaCert:  exampleCA,
		},
	}
	bz, err = cdc.MarshalJSON(&pubsubGenState)
	s.Require().NoError(err)
	appGenState[pubsubtypes.ModuleName] = bz

	// generate genesis txs
	genTxs := make([]json.RawMessage, len(s.chain.validators))
	for i, val := range s.chain.validators {
		createValmsg, err := val.buildCreateValidatorMsg(stakeAmountCoin)
		s.Require().NoError(err)

		delKeysMsg := val.buildDelegateKeysMsg()
		s.Require().NoError(err)

		signedTx, err := val.signMsg(createValmsg, delKeysMsg)
		s.Require().NoError(err)

		txRaw, err := cdc.MarshalJSON(signedTx)
		s.Require().NoError(err)

		genTxs[i] = txRaw
	}

	genUtilGenState.GenTxs = genTxs

	bz, err = cdc.MarshalJSON(&genUtilGenState)
	s.Require().NoError(err)
	appGenState[genutiltypes.ModuleName] = bz

	var corkGenState corktypes.GenesisState
	s.Require().NoError(cdc.UnmarshalJSON(appGenState[corktypes.ModuleName], &corkGenState))

	// we add the first validator address as a cellar so that it will trigger the cellarfees hook
	// when we send test fees
	corkGenState.CellarIds = corktypes.CellarIDSet{Ids: []string{unusedGenesisContract.String(), s.chain.validators[0].ethereumKey.address}}
	corkGenState.Params = corktypes.DefaultParams()
	corkGenState.Params.VoteThreshold = corkVoteThreshold
	bz, err = cdc.MarshalJSON(&corkGenState)
	s.Require().NoError(err)
	appGenState[corktypes.ModuleName] = bz

	// set contract addr
	var gravityGenState gravitytypes.GenesisState
	s.Require().NoError(cdc.UnmarshalJSON(appGenState[gravitytypes.ModuleName], &gravityGenState))
	gravityGenState.Params.GravityId = "gravitytest"
	gravityGenState.Params.BridgeEthereumAddress = gravityContract.String()
	bz, err = cdc.MarshalJSON(&gravityGenState)
	s.Require().NoError(err)
	appGenState[gravitytypes.ModuleName] = bz

	// incentivesGenState := incentivestypes.DefaultGenesisState()
	// s.Require().NoError(cdc.UnmarshalJSON(appGenState[gravitytypes.ModuleName], &gravityGenState))
	// bz, err = cdc.MarshalJSON(&incentivesGenState)
	// s.Require().NoError(err)
	// appGenState[incentivestypes.ModuleName] = bz

	// serialize genesis state
	bz, err = json.MarshalIndent(appGenState, "", "  ")
	s.Require().NoError(err)

	genDoc.AppState = bz

	bz, err = tmjson.MarshalIndent(genDoc, "", "  ")
	s.Require().NoError(err)

	// write the updated genesis file to each validator
	for _, val := range s.chain.validators {
		s.Require().NoError(writeFile(filepath.Join(val.configDir(), "config", "genesis.json"), bz))
	}
}

func (s *IntegrationTestSuite) initValidatorConfigs() {
	for i, val := range s.chain.validators {
		tmCfgPath := filepath.Join(val.configDir(), "config", "config.toml")

		vpr := viper.New()
		vpr.SetConfigFile(tmCfgPath)
		s.Require().NoError(vpr.ReadInConfig())

		valConfig := &tmconfig.Config{}
		s.Require().NoError(vpr.Unmarshal(valConfig))

		valConfig.P2P.ListenAddress = "tcp://0.0.0.0:26656"
		valConfig.P2P.AddrBookStrict = false
		valConfig.P2P.ExternalAddress = fmt.Sprintf("%s:%d", val.instanceName(), 26656)
		valConfig.RPC.ListenAddress = "tcp://0.0.0.0:26657"
		valConfig.StateSync.Enable = false
		valConfig.LogLevel = "info"

		// speed up blocks
		valConfig.Consensus.TimeoutCommit = 1 * time.Second
		valConfig.Consensus.TimeoutPropose = 1 * time.Second

		var peers []string

		for j := 0; j < len(s.chain.validators); j++ {
			if i == j {
				continue
			}

			peer := s.chain.validators[j]
			peerID := fmt.Sprintf("%s@%s%d:26656", peer.nodeKey.ID(), peer.moniker, j)
			peers = append(peers, peerID)
		}

		valConfig.P2P.PersistentPeers = strings.Join(peers, ",")

		tmconfig.WriteConfigFile(tmCfgPath, valConfig)

		// set application configuration
		appCfgPath := filepath.Join(val.configDir(), "config", "app.toml")

		appConfig := srvconfig.DefaultConfig()
		appConfig.API.Enable = true
		appConfig.Pruning = "nothing"
		appConfig.MinGasPrices = fmt.Sprintf("%s%s", minGasPrice, testDenom)

		srvconfig.WriteConfigFile(appCfgPath, appConfig)
	}
}

func (s *IntegrationTestSuite) runEthContainer() {
	s.T().Log("starting Ethereum container...")
	var err error
	runOpts := dockertest.RunOptions{
		Name:       "ethereum",
		Repository: "ethereum",
		Tag:        "prebuilt",
		NetworkID:  s.dockerNetwork.Network.ID,
		PortBindings: map[docker.Port][]docker.PortBinding{
			"8545/tcp": {{HostIP: "", HostPort: "8545"}},
		},
		ExposedPorts: []string{"8545/tcp"},
	}

	s.ethResource, err = s.dockerPool.RunWithOptions(
		&runOpts,
		noRestart,
	)
	s.Require().NoError(err)

	ethClient, err := ethclient.Dial(fmt.Sprintf("http://%s", s.ethResource.GetHostPort("8545/tcp")))
	s.Require().NoError(err)

	// Wait for the Ethereum node to respond to a request
	s.Require().Eventually(
		func() bool {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			balance, err := ethClient.BalanceAt(ctx, common.HexToAddress(s.chain.validators[0].ethereumKey.address), nil)
			if err != nil {
				s.T().Logf("error querying balance: %e", err)
				return false
			}

			if balance == nil {
				s.T().Logf("balance for first validator is nil")
			}

			if balance.Cmp(big.NewInt(0)) == 0 {
				s.T().Logf("balance for first validator is %s", balance.String())
				return false
			}

			return true
		},
		5*time.Minute,
		10*time.Second,
		"ethereum node failed to respond",
	)

	s.T().Logf("waiting for contract to deploy")
	ethereumLogOutput := bytes.Buffer{}
	err = s.dockerPool.Client.Logs(docker.LogsOptions{
		Container:    s.ethResource.Container.ID,
		OutputStream: &ethereumLogOutput,
		Stdout:       true,
	})
	s.Require().NoError(err, "error getting contract deployer logs")

	s.Require().Eventuallyf(func() bool {

		for _, s := range strings.Split(ethereumLogOutput.String(), "\n") {
			if strings.HasPrefix(s, "gravity contract deployed at") {
				strSpl := strings.Split(s, "-")
				gravityContract = common.HexToAddress(strings.ReplaceAll(strSpl[1], " ", ""))
				// this is not the last contract deployed
				continue
			}
			if strings.HasPrefix(s, "alphaERC20 contract deployed at") {
				strSpl := strings.Split(s, "-")
				alphaERC20Contract = common.HexToAddress(strings.ReplaceAll(strSpl[1], " ", ""))
				// this is not the last contract deployed
				continue
			}
			if strings.HasPrefix(s, "betaERC20 contract deployed at") {
				strSpl := strings.Split(s, "-")
				betaERC20Contract = common.HexToAddress(strings.ReplaceAll(strSpl[1], " ", ""))
				// this is not the last contract deployed
				continue
			}
			if strings.HasPrefix(s, "counter contract deployed at") {
				strSpl := strings.Split(s, "-")
				counterContract = common.HexToAddress(strings.ReplaceAll(strSpl[1], " ", ""))
				return true
			}
		}
		return false
	}, time.Minute*5, time.Second*10, "unable to retrieve gravity address from logs")
	s.T().Logf("gravity contract deployed at %s", gravityContract.String())
	s.T().Logf("alphaERC20 contract deployed at %s", alphaERC20Contract.String())
	s.T().Logf("betaERC20 contract deployed at %s", betaERC20Contract.String())
	s.T().Logf("counter contract deployed at %s", counterContract.String())
	s.T().Logf("started Ethereum container: %s", s.ethResource.Container.ID)
}

func (s *IntegrationTestSuite) runValidators() {
	s.T().Log("starting validator containers...")

	s.valResources = make([]*dockertest.Resource, len(s.chain.validators))
	for i, val := range s.chain.validators {
		runOpts := &dockertest.RunOptions{
			Name:       val.instanceName(),
			NetworkID:  s.dockerNetwork.Network.ID,
			Repository: "sommelier",
			Tag:        "prebuilt",
			Mounts: []string{
				fmt.Sprintf("%s/:/root/.sommelier", val.configDir()),
			},
			Entrypoint: []string{"sommelier", "start", "--trace=true"},
		}

		// expose the first validator for debugging and communication
		if val.index == 0 {
			runOpts.PortBindings = map[docker.Port][]docker.PortBinding{
				"1317/tcp":  {{HostIP: "", HostPort: "1317"}},
				"9090/tcp":  {{HostIP: "", HostPort: "9090"}},
				"26656/tcp": {{HostIP: "", HostPort: "26656"}},
				"26657/tcp": {{HostIP: "", HostPort: "26657"}},
			}
			runOpts.ExposedPorts = []string{"1317/tcp", "9090/tcp", "26656/tcp", "26657/tcp"}
		}

		resource, err := s.dockerPool.RunWithOptions(runOpts, noRestart)
		s.Require().NoError(err)

		s.valResources[i] = resource
		s.T().Logf("started validator container: %s", resource.Container.ID)
	}

	rpcClient, err := rpchttp.New("tcp://localhost:26657", "/websocket")
	s.Require().NoError(err)

	s.Require().Eventually(
		func() bool {
			status, err := rpcClient.Status(context.Background())
			if err != nil {
				s.T().Logf("can't get container status: %s", err.Error())
			}
			if status == nil {
				container, ok := s.dockerPool.ContainerByName("sommelier0")
				if !ok {
					s.T().Logf("no container by 'sommelier0'")
				} else {
					if container.Container.State.Status == "exited" {
						s.Fail("validators exited", "state: %s logs: \n%s", container.Container.State.String(), s.logsByContainerID(container.Container.ID))
						s.T().FailNow()
					}
					s.T().Logf("state: %v, health: %v", container.Container.State.Status, container.Container.State.Health)
				}
				return false
			}

			// let the node produce a few blocks
			if status.SyncInfo.CatchingUp {
				s.T().Logf("catching up: %t", status.SyncInfo.CatchingUp)
				return false
			}
			if status.SyncInfo.LatestBlockHeight < 2 {
				s.T().Logf("block height %d", status.SyncInfo.LatestBlockHeight)
				return false
			}

			return true
		},
		10*time.Minute,
		15*time.Second,
		"validator node failed to produce blocks",
	)
}

func (s *IntegrationTestSuite) runOrchestrators() {
	s.T().Log("starting orchestrator containers...")

	s.orchResources = make([]*dockertest.Resource, len(s.chain.orchestrators))
	for i, orch := range s.chain.orchestrators {
		gorcCfg := fmt.Sprintf(`keystore = "/root/gorc/keystore/"
[gravity]
contract = "%s"
fees_denom = "%s"
[ethereum]
key_derivation_path = "m/44'/60'/0'/0/0"
rpc = "http://%s:8545"
[cosmos]
key_derivation_path = "m/44'/118'/1'/0/0"
grpc = "http://%s:9090"
gas_price = { amount = %s, denom = "%s" }
prefix = "somm"
gas_adjustment = 2.0
msg_batch_size = 5
`,
			gravityContract.String(),
			testDenom,
			// NOTE: container names are prefixed with '/'
			s.ethResource.Container.Name[1:],
			s.valResources[i].Container.Name[1:],
			minGasPrice,
			testDenom,
		)

		val := s.chain.validators[i]

		gorcCfgPath := path.Join(val.configDir(), "gorc")
		s.Require().NoError(os.MkdirAll(gorcCfgPath, 0755))

		filePath := path.Join(gorcCfgPath, "config.toml")
		s.Require().NoError(writeFile(filePath, []byte(gorcCfg)))

		// We must first populate the orchestrator's keystore prior to starting
		// the orchestrator gorc process. The keystore must contain the Ethereum
		// and orchestrator keys. These keys will be used for relaying txs to
		// and from Somm and Ethereum. The gorc_bootstrap.sh scripts encapsulates
		// this entire process.
		//
		// NOTE: If the Docker build changes, the script might have to be modified
		// as it relies on busybox.
		err := copyFile(
			filepath.Join("integration_tests", "gorc_bootstrap.sh"),
			filepath.Join(gorcCfgPath, "gorc_bootstrap.sh"),
		)
		s.Require().NoError(err)

		resource, err := s.dockerPool.RunWithOptions(
			&dockertest.RunOptions{
				Name:       orch.instanceName(),
				NetworkID:  s.dockerNetwork.Network.ID,
				Repository: "orchestrator",
				Tag:        "prebuilt",
				Mounts: []string{
					fmt.Sprintf("%s/:/root/gorc", gorcCfgPath),
				},
				Env: []string{
					fmt.Sprintf("ORCH_MNEMONIC=%s", orch.mnemonic),
					fmt.Sprintf("ETH_PRIV_KEY=%s", val.ethereumKey.privateKey),
					"RUST_BACKTRACE=full",
					"RUST_LOG=debug",
				},
				Entrypoint: []string{
					"sh",
					"-c",
					"chmod +x /root/gorc/gorc_bootstrap.sh && /root/gorc/gorc_bootstrap.sh",
				},
			},
			noRestart,
		)
		s.Require().NoError(err)

		s.orchResources[i] = resource
		s.T().Logf("started orchestrator container: %s", resource.Container.ID)
	}

	// TODO(mvid) Determine if there is a way to check the health or status of
	// the gorc orchestrator processes. For now, we search the logs to determine
	// when each orchestrator resource has synced all batches
	match := "No unsigned batches! Everything good!"
	for _, resource := range s.orchResources {
		resource := resource
		s.T().Logf("waiting for orchestrator to be healthy: %s", resource.Container.ID)

		s.Require().Eventuallyf(
			func() bool {
				var containerLogsBuf bytes.Buffer
				s.Require().NoError(s.dockerPool.Client.Logs(
					docker.LogsOptions{
						Container:    resource.Container.ID,
						OutputStream: &containerLogsBuf,
						Stdout:       true,
						Stderr:       true,
					},
				))

				return strings.Contains(containerLogsBuf.String(), match)
			},
			3*time.Minute,
			1*time.Second,
			"orchestrator %s not healthy",
			resource.Container.ID,
		)
	}
}

func noRestart(config *docker.HostConfig) {
	// in this case we don't want the nodes to restart on failure
	config.RestartPolicy = docker.RestartPolicy{
		Name: "no",
	}
}

func (s *IntegrationTestSuite) logsByContainerID(id string) string {
	var containerLogsBuf bytes.Buffer
	s.Require().NoError(s.dockerPool.Client.Logs(
		docker.LogsOptions{
			Container:    id,
			OutputStream: &containerLogsBuf,
			Stdout:       true,
			Stderr:       true,
		},
	))

	return containerLogsBuf.String()
}

func (s *IntegrationTestSuite) TestBasicChain() {
	// this test verifies that the setup functions all operate as expected
	s.Run("bring up basic chain", func() {
	})
}
