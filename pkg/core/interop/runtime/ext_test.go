package runtime_test

import (
	"encoding/json"
	"math"
	"math/big"
	"path/filepath"
	"testing"

	"github.com/nspcc-dev/neo-go/internal/contracts"
	"github.com/nspcc-dev/neo-go/internal/random"
	"github.com/nspcc-dev/neo-go/pkg/config"
	"github.com/nspcc-dev/neo-go/pkg/config/netmode"
	"github.com/nspcc-dev/neo-go/pkg/core"
	"github.com/nspcc-dev/neo-go/pkg/core/block"
	"github.com/nspcc-dev/neo-go/pkg/core/interop"
	"github.com/nspcc-dev/neo-go/pkg/core/interop/interopnames"
	"github.com/nspcc-dev/neo-go/pkg/core/interop/runtime"
	"github.com/nspcc-dev/neo-go/pkg/core/native"
	"github.com/nspcc-dev/neo-go/pkg/core/native/nativenames"
	"github.com/nspcc-dev/neo-go/pkg/core/state"
	"github.com/nspcc-dev/neo-go/pkg/core/transaction"
	"github.com/nspcc-dev/neo-go/pkg/crypto/hash"
	"github.com/nspcc-dev/neo-go/pkg/crypto/keys"
	"github.com/nspcc-dev/neo-go/pkg/encoding/address"
	"github.com/nspcc-dev/neo-go/pkg/io"
	"github.com/nspcc-dev/neo-go/pkg/neotest"
	"github.com/nspcc-dev/neo-go/pkg/neotest/chain"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract/callflag"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract/manifest"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract/nef"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract/trigger"
	"github.com/nspcc-dev/neo-go/pkg/util"
	"github.com/nspcc-dev/neo-go/pkg/vm"
	"github.com/nspcc-dev/neo-go/pkg/vm/emit"
	"github.com/nspcc-dev/neo-go/pkg/vm/opcode"
	"github.com/nspcc-dev/neo-go/pkg/vm/stackitem"
	"github.com/stretchr/testify/require"
)

var pathToInternalContracts = filepath.Join("..", "..", "..", "..", "internal", "contracts")

func getSharpTestTx(sender util.Uint160) *transaction.Transaction {
	tx := transaction.New([]byte{byte(opcode.PUSH2)}, 0)
	tx.Nonce = 0
	tx.Signers = append(tx.Signers, transaction.Signer{
		Account: sender,
		Scopes:  transaction.CalledByEntry,
	})
	tx.Attributes = []transaction.Attribute{}
	tx.Scripts = append(tx.Scripts, transaction.Witness{InvocationScript: []byte{}, VerificationScript: []byte{}})
	return tx
}

func getSharpTestGenesis(t *testing.T) *block.Block {
	const configPath = "../../../../config"

	cfg, err := config.Load(configPath, netmode.MainNet)
	require.NoError(t, err)
	b, err := core.CreateGenesisBlock(cfg.ProtocolConfiguration)
	require.NoError(t, err)
	return b
}

func createVM(t testing.TB) (*vm.VM, *interop.Context, *core.Blockchain) {
	chain, _ := chain.NewSingle(t)
	ic := chain.GetTestVM(trigger.Application, &transaction.Transaction{}, &block.Block{})
	v := ic.SpawnVM()
	return v, ic, chain
}

func loadScriptWithHashAndFlags(ic *interop.Context, script []byte, hash util.Uint160, f callflag.CallFlag, args ...interface{}) {
	ic.SpawnVM()
	ic.VM.LoadScriptWithHash(script, hash, f)
	for i := range args {
		ic.VM.Estack().PushVal(args[i])
	}
	ic.VM.GasLimit = -1
}

func TestBurnGas(t *testing.T) {
	bc, acc := chain.NewSingle(t)
	e := neotest.NewExecutor(t, bc, acc, acc)
	managementInvoker := e.ValidatorInvoker(e.NativeHash(t, nativenames.Management))

	cs, _ := contracts.GetTestContractState(t, pathToInternalContracts, 0, 1, acc.ScriptHash())
	rawManifest, err := json.Marshal(cs.Manifest)
	require.NoError(t, err)
	rawNef, err := cs.NEF.Bytes()
	require.NoError(t, err)
	tx := managementInvoker.PrepareInvoke(t, "deploy", rawNef, rawManifest)
	e.AddNewBlock(t, tx)
	e.CheckHalt(t, tx.Hash())
	cInvoker := e.ValidatorInvoker(cs.Hash)

	t.Run("good", func(t *testing.T) {
		h := cInvoker.Invoke(t, stackitem.Null{}, "burnGas", int64(1))
		res := e.GetTxExecResult(t, h)

		t.Run("gas limit exceeded", func(t *testing.T) {
			tx := e.NewUnsignedTx(t, cs.Hash, "burnGas", int64(2))
			e.SignTx(t, tx, res.GasConsumed, acc)
			e.AddNewBlock(t, tx)
			e.CheckFault(t, tx.Hash(), "GAS limit exceeded")
		})
	})
	t.Run("too big integer", func(t *testing.T) {
		gas := big.NewInt(math.MaxInt64)
		gas.Add(gas, big.NewInt(1))

		cInvoker.InvokeFail(t, "invalid GAS value", "burnGas", gas)
	})
	t.Run("zero GAS", func(t *testing.T) {
		cInvoker.InvokeFail(t, "GAS must be positive", "burnGas", int64(0))
	})
}

func TestCheckWitness(t *testing.T) {
	_, ic, _ := createVM(t)

	script := []byte{byte(opcode.RET)}
	scriptHash := hash.Hash160(script)
	check := func(t *testing.T, ic *interop.Context, arg interface{}, shouldFail bool, expected ...bool) {
		ic.VM.Estack().PushVal(arg)
		err := runtime.CheckWitness(ic)
		if shouldFail {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.NotNil(t, expected)
			actual, ok := ic.VM.Estack().Pop().Value().(bool)
			require.True(t, ok)
			require.Equal(t, expected[0], actual)
		}
	}
	t.Run("error", func(t *testing.T) {
		t.Run("not a hash or key", func(t *testing.T) {
			check(t, ic, []byte{1, 2, 3}, true)
		})
		t.Run("script container is not a transaction", func(t *testing.T) {
			loadScriptWithHashAndFlags(ic, script, scriptHash, callflag.ReadStates)
			check(t, ic, random.Uint160().BytesBE(), true)
		})
		t.Run("check scope", func(t *testing.T) {
			t.Run("CustomGroups, missing ReadStates flag", func(t *testing.T) {
				hash := random.Uint160()
				tx := &transaction.Transaction{
					Signers: []transaction.Signer{
						{
							Account:       hash,
							Scopes:        transaction.CustomGroups,
							AllowedGroups: []*keys.PublicKey{},
						},
					},
				}
				ic.Tx = tx
				callingScriptHash := scriptHash
				loadScriptWithHashAndFlags(ic, script, callingScriptHash, callflag.All)
				ic.VM.LoadScriptWithHash([]byte{0x1}, random.Uint160(), callflag.AllowCall)
				check(t, ic, hash.BytesBE(), true)
			})
			t.Run("Rules, missing ReadStates flag", func(t *testing.T) {
				hash := random.Uint160()
				pk, err := keys.NewPrivateKey()
				require.NoError(t, err)
				tx := &transaction.Transaction{
					Signers: []transaction.Signer{
						{
							Account: hash,
							Scopes:  transaction.Rules,
							Rules: []transaction.WitnessRule{{
								Action:    transaction.WitnessAllow,
								Condition: (*transaction.ConditionGroup)(pk.PublicKey()),
							}},
						},
					},
				}
				ic.Tx = tx
				callingScriptHash := scriptHash
				loadScriptWithHashAndFlags(ic, script, callingScriptHash, callflag.All)
				ic.VM.LoadScriptWithHash([]byte{0x1}, random.Uint160(), callflag.AllowCall)
				check(t, ic, hash.BytesBE(), true)
			})
		})
	})
	t.Run("positive", func(t *testing.T) {
		t.Run("calling scripthash", func(t *testing.T) {
			t.Run("hashed witness", func(t *testing.T) {
				callingScriptHash := scriptHash
				loadScriptWithHashAndFlags(ic, script, callingScriptHash, callflag.All)
				ic.VM.LoadScriptWithHash([]byte{0x1}, random.Uint160(), callflag.All)
				check(t, ic, callingScriptHash.BytesBE(), false, true)
			})
			t.Run("keyed witness", func(t *testing.T) {
				pk, err := keys.NewPrivateKey()
				require.NoError(t, err)
				callingScriptHash := pk.PublicKey().GetScriptHash()
				loadScriptWithHashAndFlags(ic, script, callingScriptHash, callflag.All)
				ic.VM.LoadScriptWithHash([]byte{0x1}, random.Uint160(), callflag.All)
				check(t, ic, pk.PublicKey().Bytes(), false, true)
			})
		})
		t.Run("check scope", func(t *testing.T) {
			t.Run("Global", func(t *testing.T) {
				hash := random.Uint160()
				tx := &transaction.Transaction{
					Signers: []transaction.Signer{
						{
							Account: hash,
							Scopes:  transaction.Global,
						},
					},
				}
				loadScriptWithHashAndFlags(ic, script, scriptHash, callflag.ReadStates)
				ic.Tx = tx
				check(t, ic, hash.BytesBE(), false, true)
			})
			t.Run("CalledByEntry", func(t *testing.T) {
				hash := random.Uint160()
				tx := &transaction.Transaction{
					Signers: []transaction.Signer{
						{
							Account: hash,
							Scopes:  transaction.CalledByEntry,
						},
					},
				}
				loadScriptWithHashAndFlags(ic, script, scriptHash, callflag.ReadStates)
				ic.Tx = tx
				check(t, ic, hash.BytesBE(), false, true)
			})
			t.Run("CustomContracts", func(t *testing.T) {
				hash := random.Uint160()
				tx := &transaction.Transaction{
					Signers: []transaction.Signer{
						{
							Account:          hash,
							Scopes:           transaction.CustomContracts,
							AllowedContracts: []util.Uint160{scriptHash},
						},
					},
				}
				loadScriptWithHashAndFlags(ic, script, scriptHash, callflag.ReadStates)
				ic.Tx = tx
				check(t, ic, hash.BytesBE(), false, true)
			})
			t.Run("CustomGroups", func(t *testing.T) {
				t.Run("unknown scripthash", func(t *testing.T) {
					hash := random.Uint160()
					tx := &transaction.Transaction{
						Signers: []transaction.Signer{
							{
								Account:       hash,
								Scopes:        transaction.CustomGroups,
								AllowedGroups: []*keys.PublicKey{},
							},
						},
					}
					loadScriptWithHashAndFlags(ic, script, scriptHash, callflag.ReadStates)
					ic.Tx = tx
					check(t, ic, hash.BytesBE(), false, false)
				})
				t.Run("positive", func(t *testing.T) {
					targetHash := random.Uint160()
					pk, err := keys.NewPrivateKey()
					require.NoError(t, err)
					tx := &transaction.Transaction{
						Signers: []transaction.Signer{
							{
								Account:       targetHash,
								Scopes:        transaction.CustomGroups,
								AllowedGroups: []*keys.PublicKey{pk.PublicKey()},
							},
						},
					}
					contractScript := []byte{byte(opcode.PUSH1), byte(opcode.RET)}
					contractScriptHash := hash.Hash160(contractScript)
					ne, err := nef.NewFile(contractScript)
					require.NoError(t, err)
					contractState := &state.Contract{
						ContractBase: state.ContractBase{
							ID:   15,
							Hash: contractScriptHash,
							NEF:  *ne,
							Manifest: manifest.Manifest{
								Groups: []manifest.Group{{PublicKey: pk.PublicKey(), Signature: make([]byte, keys.SignatureLen)}},
							},
						},
					}
					require.NoError(t, native.PutContractState(ic.DAO, contractState))
					loadScriptWithHashAndFlags(ic, contractScript, contractScriptHash, callflag.All)
					ic.Tx = tx
					check(t, ic, targetHash.BytesBE(), false, true)
				})
			})
			t.Run("Rules", func(t *testing.T) {
				t.Run("no match", func(t *testing.T) {
					hash := random.Uint160()
					tx := &transaction.Transaction{
						Signers: []transaction.Signer{
							{
								Account: hash,
								Scopes:  transaction.Rules,
								Rules: []transaction.WitnessRule{{
									Action:    transaction.WitnessAllow,
									Condition: (*transaction.ConditionScriptHash)(&hash),
								}},
							},
						},
					}
					loadScriptWithHashAndFlags(ic, script, scriptHash, callflag.ReadStates)
					ic.Tx = tx
					check(t, ic, hash.BytesBE(), false, false)
				})
				t.Run("allow", func(t *testing.T) {
					hash := random.Uint160()
					var cond = true
					tx := &transaction.Transaction{
						Signers: []transaction.Signer{
							{
								Account: hash,
								Scopes:  transaction.Rules,
								Rules: []transaction.WitnessRule{{
									Action:    transaction.WitnessAllow,
									Condition: (*transaction.ConditionBoolean)(&cond),
								}},
							},
						},
					}
					loadScriptWithHashAndFlags(ic, script, scriptHash, callflag.ReadStates)
					ic.Tx = tx
					check(t, ic, hash.BytesBE(), false, true)
				})
				t.Run("deny", func(t *testing.T) {
					hash := random.Uint160()
					var cond = true
					tx := &transaction.Transaction{
						Signers: []transaction.Signer{
							{
								Account: hash,
								Scopes:  transaction.Rules,
								Rules: []transaction.WitnessRule{{
									Action:    transaction.WitnessDeny,
									Condition: (*transaction.ConditionBoolean)(&cond),
								}},
							},
						},
					}
					loadScriptWithHashAndFlags(ic, script, scriptHash, callflag.ReadStates)
					ic.Tx = tx
					check(t, ic, hash.BytesBE(), false, false)
				})
			})
			t.Run("bad scope", func(t *testing.T) {
				hash := random.Uint160()
				tx := &transaction.Transaction{
					Signers: []transaction.Signer{
						{
							Account: hash,
							Scopes:  transaction.None,
						},
					},
				}
				loadScriptWithHashAndFlags(ic, script, scriptHash, callflag.ReadStates)
				ic.Tx = tx
				check(t, ic, hash.BytesBE(), false, false)
			})
		})
	})
}

func TestGasLeft(t *testing.T) {
	const runtimeGasLeftPrice = 1 << 4

	bc, acc := chain.NewSingle(t)
	e := neotest.NewExecutor(t, bc, acc, acc)
	w := io.NewBufBinWriter()

	gasLimit := 1100
	emit.Syscall(w.BinWriter, interopnames.SystemRuntimeGasLeft)
	emit.Syscall(w.BinWriter, interopnames.SystemRuntimeGasLeft)
	require.NoError(t, w.Err)
	tx := transaction.New(w.Bytes(), int64(gasLimit))
	tx.Nonce = neotest.Nonce()
	tx.ValidUntilBlock = e.Chain.BlockHeight() + 1
	e.SignTx(t, tx, int64(gasLimit), acc)
	e.AddNewBlock(t, tx)
	e.CheckHalt(t, tx.Hash())
	res := e.GetTxExecResult(t, tx.Hash())
	l1 := res.Stack[0].Value().(*big.Int)
	l2 := res.Stack[1].Value().(*big.Int)

	require.Equal(t, int64(gasLimit-runtimeGasLeftPrice*interop.DefaultBaseExecFee), l1.Int64())
	require.Equal(t, int64(gasLimit-2*runtimeGasLeftPrice*interop.DefaultBaseExecFee), l2.Int64())
}

func TestGetAddressVersion(t *testing.T) {
	bc, acc := chain.NewSingle(t)
	e := neotest.NewExecutor(t, bc, acc, acc)
	w := io.NewBufBinWriter()

	emit.Syscall(w.BinWriter, interopnames.SystemRuntimeGetAddressVersion)
	require.NoError(t, w.Err)
	e.InvokeScriptCheckHALT(t, w.Bytes(), []neotest.Signer{acc}, stackitem.NewBigInteger(big.NewInt(int64(address.NEO3Prefix))))
}

func TestGetInvocationCounter(t *testing.T) {
	v, ic, _ := createVM(t)

	cs, _ := contracts.GetTestContractState(t, pathToInternalContracts, 4, 5, random.Uint160()) // sender and IDs are not important for the test
	require.NoError(t, native.PutContractState(ic.DAO, cs))

	ic.Invocations[hash.Hash160([]byte{2})] = 42

	t.Run("No invocations", func(t *testing.T) {
		v.Load([]byte{1})
		// do not return an error in this case.
		require.NoError(t, runtime.GetInvocationCounter(ic))
		require.EqualValues(t, 1, v.Estack().Pop().BigInt().Int64())
	})
	t.Run("NonZero", func(t *testing.T) {
		v.Load([]byte{2})
		require.NoError(t, runtime.GetInvocationCounter(ic))
		require.EqualValues(t, 42, v.Estack().Pop().BigInt().Int64())
	})
	t.Run("Contract", func(t *testing.T) {
		script, err := smartcontract.CreateCallScript(cs.Hash, "invocCounter")
		require.NoError(t, err)
		v.LoadWithFlags(script, callflag.All)
		require.NoError(t, v.Run())
		require.EqualValues(t, 1, v.Estack().Pop().BigInt().Int64())
	})
}

func TestGetNetwork(t *testing.T) {
	bc, acc := chain.NewSingle(t)
	e := neotest.NewExecutor(t, bc, acc, acc)
	w := io.NewBufBinWriter()

	emit.Syscall(w.BinWriter, interopnames.SystemRuntimeGetNetwork)
	require.NoError(t, w.Err)
	e.InvokeScriptCheckHALT(t, w.Bytes(), []neotest.Signer{acc}, stackitem.NewBigInteger(big.NewInt(int64(bc.GetConfig().Magic))))
}

func TestGetNotifications(t *testing.T) {
	v, ic, _ := createVM(t)

	ic.Notifications = []state.NotificationEvent{
		{ScriptHash: util.Uint160{1}, Name: "Event1", Item: stackitem.NewArray([]stackitem.Item{stackitem.NewByteArray([]byte{11})})},
		{ScriptHash: util.Uint160{2}, Name: "Event2", Item: stackitem.NewArray([]stackitem.Item{stackitem.NewByteArray([]byte{22})})},
		{ScriptHash: util.Uint160{1}, Name: "Event1", Item: stackitem.NewArray([]stackitem.Item{stackitem.NewByteArray([]byte{33})})},
	}

	t.Run("NoFilter", func(t *testing.T) {
		v.Estack().PushVal(stackitem.Null{})
		require.NoError(t, runtime.GetNotifications(ic))

		arr := v.Estack().Pop().Array()
		require.Equal(t, len(ic.Notifications), len(arr))
		for i := range arr {
			elem := arr[i].Value().([]stackitem.Item)
			require.Equal(t, ic.Notifications[i].ScriptHash.BytesBE(), elem[0].Value())
			name, err := stackitem.ToString(elem[1])
			require.NoError(t, err)
			require.Equal(t, ic.Notifications[i].Name, name)
			ic.Notifications[i].Item.MarkAsReadOnly() // tiny hack for test to be able to compare object references.
			require.Equal(t, ic.Notifications[i].Item, elem[2])
		}
	})

	t.Run("WithFilter", func(t *testing.T) {
		h := util.Uint160{2}.BytesBE()
		v.Estack().PushVal(h)
		require.NoError(t, runtime.GetNotifications(ic))

		arr := v.Estack().Pop().Array()
		require.Equal(t, 1, len(arr))
		elem := arr[0].Value().([]stackitem.Item)
		require.Equal(t, h, elem[0].Value())
		name, err := stackitem.ToString(elem[1])
		require.NoError(t, err)
		require.Equal(t, ic.Notifications[1].Name, name)
		require.Equal(t, ic.Notifications[1].Item, elem[2])
	})
}

func TestGetRandom_DifferentTransactions(t *testing.T) {
	bc, acc := chain.NewSingle(t)
	e := neotest.NewExecutor(t, bc, acc, acc)

	w := io.NewBufBinWriter()
	emit.Syscall(w.BinWriter, interopnames.SystemRuntimeGetRandom)
	require.NoError(t, w.Err)
	script := w.Bytes()

	tx1 := e.PrepareInvocation(t, script, []neotest.Signer{e.Validator}, bc.BlockHeight()+1)
	tx2 := e.PrepareInvocation(t, script, []neotest.Signer{e.Validator}, bc.BlockHeight()+1)
	e.AddNewBlock(t, tx1, tx2)
	e.CheckHalt(t, tx1.Hash())
	e.CheckHalt(t, tx2.Hash())

	res1 := e.GetTxExecResult(t, tx1.Hash())
	res2 := e.GetTxExecResult(t, tx2.Hash())

	r1, err := res1.Stack[0].TryInteger()
	require.NoError(t, err)
	r2, err := res2.Stack[0].TryInteger()
	require.NoError(t, err)
	require.NotEqual(t, r1, r2)
}

// Tests are taken from
// https://github.com/neo-project/neo/blob/master/tests/neo.UnitTests/SmartContract/UT_ApplicationEngine.Runtime.cs
func TestGetRandomCompatibility(t *testing.T) {
	bc, _ := chain.NewSingle(t)

	b := getSharpTestGenesis(t)
	tx := getSharpTestTx(util.Uint160{})
	ic := bc.GetTestVM(trigger.Application, tx, b)
	ic.Network = 860833102 // Old mainnet magic used by C# tests.

	ic.VM = vm.New()
	ic.VM.LoadScript([]byte{0x01})
	ic.VM.GasLimit = 1100_00000000

	require.NoError(t, runtime.GetRandom(ic))
	require.Equal(t, "271339657438512451304577787170704246350", ic.VM.Estack().Pop().BigInt().String())

	require.NoError(t, runtime.GetRandom(ic))
	require.Equal(t, "98548189559099075644778613728143131367", ic.VM.Estack().Pop().BigInt().String())

	require.NoError(t, runtime.GetRandom(ic))
	require.Equal(t, "247654688993873392544380234598471205121", ic.VM.Estack().Pop().BigInt().String())

	require.NoError(t, runtime.GetRandom(ic))
	require.Equal(t, "291082758879475329976578097236212073607", ic.VM.Estack().Pop().BigInt().String())

	require.NoError(t, runtime.GetRandom(ic))
	require.Equal(t, "247152297361212656635216876565962360375", ic.VM.Estack().Pop().BigInt().String())
}
