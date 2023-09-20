package dataset_test

import (
	. "github.com/Troublor/erebus-redgiant/dataset"
	"github.com/Troublor/erebus-redgiant/global"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
)

var _ = Describe("Attack", func() {
	searchForAttack := func(
		attackTx, victimTx common.Hash,
		profitTx *common.Hash,
	) (*SearchWindow, *Attack) {
		history := NewTxHistory(global.BlockchainReader(), nil)
		searcher := NewAttackSearcher(global.BlockchainReader(), history)
		searcher.SetPool(global.GoroutinePool())
		var attack *Attack
		searcher.SetAttackHandler(func(session *TxHistorySession, a *Attack) {
			attack = a
		})
		attackReceipt, err := global.BlockchainReader().TransactionReceipt(global.Ctx(), attackTx)
		Expect(err).NotTo(HaveOccurred())
		victimReceipt, err := global.BlockchainReader().TransactionReceipt(global.Ctx(), victimTx)
		Expect(err).NotTo(HaveOccurred())
		var profitReceipt *types.Receipt
		if profitTx != nil {
			profitReceipt, err = global.BlockchainReader().
				TransactionReceipt(global.Ctx(), *profitTx)
			Expect(err).NotTo(HaveOccurred())
		}
		var from, to uint64
		from = attackReceipt.BlockNumber.Uint64()
		if profitReceipt != nil {
			to = profitReceipt.BlockNumber.Uint64()
		} else {
			to = victimReceipt.BlockNumber.Uint64()
		}

		window := searcher.OpenSearchWindow(global.Ctx(), from, int(to-from+1))
		window.SetFocus(attackTx, victimTx, profitTx)
		window.Search(global.Ctx())
		return window, attack
	}

	Context("OpenSea buying case", Ordered, func() {
		// the victim and attack tries to buy ERC721 token by OpenSea.atomicMatch_() using Ether.
		// the attacker succeeds but the victim fails.
		attackTx := common.HexToHash(
			"0xb0f9e58b10488ac2f5b549f4c0a8816f9004f98fc06fdd5ea53966d6f421ad31",
		)
		victimTx := common.HexToHash(
			"0x79162c3322cc7079ea9f3f882b5acfa305e9404a8f2e43f29b9ffc52808e8d9a",
		)
		var attack *Attack
		var window *SearchWindow

		AfterAll(func() {
			if window != nil {
				window.Close()
			}
		})

		It("should not be found by searcher", func() {
			// the victim's overall profits are not strictly smaller under attack.
			window, attack = searchForAttack(attackTx, victimTx, nil)
			Expect(attack).To(BeNil())
		})
	})

	Context("UniswapV2Router swapping case", Ordered, func() {
		// swap price slippage
		attackTx := common.HexToHash(
			"0x5b4360a6ebdadc234d2b23dbdbc472d9002fca863e757c44e4dbda740c498588",
		)
		victimTx := common.HexToHash(
			"0x048394c15d45f429c92bd6bda07649f0faa16f2e59845d4a985f0998f568974c",
		)
		var attack *Attack
		var window *SearchWindow

		AfterAll(func() {
			if window != nil {
				window.Close()
			}
		})

		It("should be found by searcher", func() {
			window, attack = searchForAttack(attackTx, victimTx, nil)
			Expect(attack).NotTo(BeNil())
			Expect(attack.Attacker.Hex()).To(Equal("0xe4013B5bbA21556CC1f30A581CB0b5D0E98A56b0"))
			Expect(attack.Victim.Hex()).To(Equal("0xEc44B704a51C27C63478700a5F90d5DA53F93097"))
		})

		It("should be analyzed", func() {
			analysis, err := attack.Analyze(global.BlockchainReader(), window.TxHistorySession())
			Expect(err).NotTo(HaveOccurred())
			Expect(analysis).To(HaveLen(1))
		})
	})

	PContext("Ethereum dark forest the horror story case", Ordered, func() {
		// swap price slippage
		attackTx := common.HexToHash(
			"0xcc7f752e990b32befa1e0c82b036b3753ec3d876b336007cd568983dca0af497",
		)
		victimTx := common.HexToHash(
			"0x93b1a493e9d871b9d3f553996653c4d2f50bf6a2c744ce31e8de89956472863e",
		)
		var attack *Attack
		var window *SearchWindow

		AfterAll(func() {
			if window != nil {
				window.Close()
			}
		})

		It("should be found by searcher", func() {
			window, attack = searchForAttack(attackTx, victimTx, nil)
			Expect(attack).NotTo(BeNil())
		})

		It("should be analyzed", func() {
			analysis, err := attack.Analyze(global.BlockchainReader(), window.TxHistorySession())
			Expect(err).NotTo(HaveOccurred())
			Expect(analysis).To(HaveLen(3))
		})
	})

	Context("Standard sandwich attack on UniswapV2 case", Ordered, func() {
		// swap price slippage
		attackTx := common.HexToHash(
			"0x38af50c691fe2c2441f4ddddb1aa3217581343d601165178807148d9266c0546",
		)
		victimTx := common.HexToHash(
			"0xcef9ef2886b6fb04c7185673123b2f2794a048e41e45018ec6026d335226769c",
		)
		profitTx := common.HexToHash(
			"0x65a1a18bb912aa8db5bfd460bd6a7ac91aa6d5dbb8ca161b293b19316485b219",
		)
		var attack *Attack
		var window *SearchWindow

		AfterAll(func() {
			if window != nil {
				window.Close()
			}
		})

		It("should be found by searcher", func() {
			window, attack = searchForAttack(attackTx, victimTx, &profitTx)
			Expect(attack).NotTo(BeNil())
			Expect(attack.Attacker.Hex()).To(Equal("0x000000000035B5e5ad9019092C665357240f594e"))
			Expect(attack.Victim.Hex()).To(Equal("0xF41d7209849AccACA059a93460F23fbb6A12EF6c"))
		})

		It("should be analyzed", func() {
			analysis, err := attack.Analyze(global.BlockchainReader(), window.TxHistorySession())
			Expect(err).NotTo(HaveOccurred())
			Expect(analysis).To(HaveLen(1))
		})
	})

	Context("out of gas case", Ordered, func() {
		// the victim tx runs out of gas due to attack transaction.
		attackTx := common.HexToHash(
			"0x344f1a6eb9ca880748928f6810779f9816db9369a9127bd01bad0ea05b9d9def",
		)
		victimTx := common.HexToHash(
			"0x2eb6a786109536f526990e3dba0033a2261675ea061c69ac2fd0bc39734d0ea0",
		)
		var attack *Attack
		var window *SearchWindow

		AfterAll(func() {
			if window != nil {
				window.Close()
			}
		})

		It("should be found by searcher", func() {
			window, attack = searchForAttack(attackTx, victimTx, nil)
			Expect(attack).NotTo(BeNil())
			Expect(attack.Attacker.Hex()).To(Equal("0x01C2e7b1de06DA53Bd0EC82fdB59E5767b8c6dA1"))
			Expect(attack.Victim.Hex()).To(Equal("0x603b93e94b997598d642b3FcF28fB4841c24634a"))
		})

		It("should be analyzed", func() {
			analysis, err := attack.Analyze(global.BlockchainReader(), window.TxHistorySession())
			Expect(err).NotTo(HaveOccurred())
			// analysis will not be available since this is out of gas and not necessarily a vulnerability.
			Expect(len(analysis)).To(BeNumerically(">", 0))
		})
	})

	Context("UniswapV2 out of gas case", Ordered, func() {
		// the victim tx runs out of gas due to attack transaction.
		attackTx := common.HexToHash(
			"0xf08753e2005d40cd6ac213f34bd3c2420d9aa105558301ae4973ccf05af03b97",
		)
		victimTx := common.HexToHash(
			"0x47d5bd93151d4317213923e2ba87d440c0db7fc4190360146cd7843a51195ccc",
		)
		var attack *Attack
		var window *SearchWindow

		AfterAll(func() {
			if window != nil {
				window.Close()
			}
		})

		It("should be found by searcher", func() {
			window, attack = searchForAttack(attackTx, victimTx, nil)
			Expect(attack).NotTo(BeNil())
			Expect(attack.Attacker.Hex()).To(Equal("0x433311a2F08cA7d7062fE7e3D703D8E8C865e9bC"))
			Expect(attack.Victim.Hex()).To(Equal("0x15c643dc1B4DcF2c5384be30Ce819d65D34aBF07"))
		})

		It("should be analyzed", func() {
			analysis, err := attack.Analyze(global.BlockchainReader(), window.TxHistorySession())
			Expect(err).NotTo(HaveOccurred())
			// analysis will not be available since this is out of gas and not necessarily a vulnerability.
			Expect(len(analysis)).To(BeNumerically(">", 0))
		})
	})

	Context("UniswapV2 swap another case", Ordered, func() {
		attackTx := common.HexToHash(
			"0x8ccad00d248873243168a848c6e64889cd4598c81ed5088d22275d0852752fb4",
		)
		victimTx := common.HexToHash(
			"0x1e5f025f6f5a00979339e430a038c8c65566f5d707377b9d53f026369041ec9d",
		)
		var attack *Attack
		var window *SearchWindow

		AfterAll(func() {
			if window != nil {
				window.Close()
			}
		})

		It("should be found by searcher", func() {
			window, attack = searchForAttack(attackTx, victimTx, nil)
			Expect(attack).NotTo(BeNil())
			Expect(attack.Attacker.Hex()).To(Equal("0xA428936Fa3cA794fa6a5dBfA80b4E3e16AB4E782"))
			Expect(attack.Victim.Hex()).To(Equal("0xC5026221a3546104Ba125Aeb7Ea5d95C1BfDf9b3"))
		})

		It("should be analyzed", func() {
			analysis, err := attack.Analyze(global.BlockchainReader(), window.TxHistorySession())
			Expect(err).NotTo(HaveOccurred())
			Expect(analysis).To(HaveLen(1))
		})
	})

	Context("Shared variable not def-clear case", Ordered, func() {
		// the shared variable written in attackTx is not directly used in the victimTx.
		// the victimTx first override the shared variable and then use it.
		// This is still an attack, but the shared variable should not be this one.
		attackTx := common.HexToHash(
			"0x94b5f3ff63dd727f6d273a0f363833268f545df62d6471f61cee5309da278c2d",
		)
		victimTx := common.HexToHash(
			"0x6c8ff8eaca26b4ff4f8125ba8f11a701037843894b3047954691f22fe06d2dc2",
		)
		var attack *Attack
		var window *SearchWindow

		AfterAll(func() {
			if window != nil {
				window.Close()
			}
		})

		It("should be found by searcher", func() {
			window, attack = searchForAttack(attackTx, victimTx, nil)
			Expect(attack).NotTo(BeNil())
			Expect(attack.Attacker.Hex()).To(Equal("0x9801F8Fc3C712E4a0541a73aAb75d688cD4e5b09"))
			Expect(attack.Victim.Hex()).To(Equal("0x5f8A1752b9aFd9Fc7f2D0e60DDca86f4cb32C450"))
		})

		It("should be analyzed", func() {
			analysis, err := attack.Analyze(global.BlockchainReader(), window.TxHistorySession())
			Expect(err).NotTo(HaveOccurred())
			Expect(len(analysis)).To(BeNumerically(">", 0))
		})
	})

	Context("UniswapV2Router swap case", Ordered, func() {
		// the unconditional jump destination of the consequence point (branches have different profits)
		// is pushed before the taint source. The unconditional jump should still be tainted.
		attackTx := common.HexToHash(
			"0x5e7da8bd2ba32b67f22aae34fb5c43398ebfa44749b1405c4573d3caecd9c440",
		)
		victimTx := common.HexToHash(
			"0x294808e179f88885a3ecc1d185d25490792f5fd105ab999540a25b3b965d6905",
		)
		var attack *Attack
		var window *SearchWindow

		AfterAll(func() {
			if window != nil {
				window.Close()
			}
		})

		It("should be found by searcher", func() {
			window, attack = searchForAttack(attackTx, victimTx, nil)
			Expect(attack).NotTo(BeNil())
			Expect(attack.Attacker).To(Equal(common.HexToAddress(
				"0x792De7a1f350D36C7886cAc24dEd219c498b5Bc9",
			)))
			Expect(attack.Victim).To(Equal(common.HexToAddress(
				"0x7aFE0ca3a151085ce84acF14B46bd4FF308f8Bc1",
			)))
		})

		It("should be analyzed", func() {
			analysis, err := attack.Analyze(global.BlockchainReader(), window.TxHistorySession())
			Expect(err).NotTo(HaveOccurred())
			Expect(analysis).To(HaveLen(1))
			for _, a := range analysis {
				Expect(a.InfluenceString(attack.VictimTxRecord.State.GetCodeHash)).NotTo(BeEmpty())
			}
		})
	})

	Context("UniswapV3Router swap case", Ordered, func() {
		// the unconditional jump destination of the consequence point (branches have different profits)
		// is pushed before the taint source. The unconditional jump should still be tainted.
		attackTx := common.HexToHash(
			"0x8bc9ab4aab6975ea276f1864492d2845a664ef34aa885378016dc63b88688d2e",
		)
		victimTx := common.HexToHash(
			"0x60edd8ebd25935de7ab61a6d33d9adc4108f82d22a116c4bc08ea58f7c38c5c0",
		)
		var attack *Attack
		var window *SearchWindow

		AfterAll(func() {
			if window != nil {
				window.Close()
			}
		})

		It("should be found by searcher", func() {
			window, attack = searchForAttack(attackTx, victimTx, nil)
			Expect(attack).NotTo(BeNil())
			Expect(attack.Attacker).To(Equal(common.HexToAddress(
				"0xc5a2f28bacf1425bfaea14834486f0e9d0832155",
			)))
			Expect(attack.Victim).To(Equal(common.HexToAddress(
				"0xa267f3a4a92531f47811e05e890e977a0fe375b4",
			)))
		})

		It("should be analyzed", func() {
			analysis, err := attack.Analyze(global.BlockchainReader(), window.TxHistorySession())
			Expect(err).NotTo(HaveOccurred())
			Expect(analysis).To(HaveLen(1))
			for _, a := range analysis {
				Expect(a.InfluenceString(attack.VictimTxRecord.State.GetCodeHash)).NotTo(BeEmpty())
			}
		})
	})

	Context("NFT token mint out of gas case", Ordered, func() {
		// the total gas cost depends on _tokenIdCounter, which is modified by attack tx.
		// as a result, the gas cost of victim tx increases and the it runs out of gas
		attackTx := common.HexToHash(
			"0x3c329b68bbe2118219f99469c1e98df709c5b232a046d218f2e45c1bad61f791",
		)
		victimTx := common.HexToHash(
			"0x9cdb5f5f85e743af3a5514960a9290ee44e5c59c9f57cfd5c270329653eddf38",
		)
		var attack *Attack
		var window *SearchWindow

		AfterAll(func() {
			if window != nil {
				window.Close()
			}
		})

		It("should be found by searcher", func() {
			window, attack = searchForAttack(attackTx, victimTx, nil)
			Expect(attack).NotTo(BeNil())
			Expect(attack.Attacker).To(Equal(common.HexToAddress(
				"0x01DBF05FB100Fd2f0d963EBA2cBEdF765c7B347e",
			)))
			Expect(attack.Victim).To(Equal(common.HexToAddress(
				"0x3Dd739c21678a3d14AB9aA48d4AA575c25472C09",
			)))
		})

		It("should be analyzed", func() {
			analysis, err := attack.Analyze(global.BlockchainReader(), window.TxHistorySession())
			Expect(err).NotTo(HaveOccurred())
			Expect(analysis).To(HaveLen(1))
		})
	})

	Context("UniswapV3 taint flow through ether transfer", Ordered, func() {
		// the taint flows to the value of ether transfer.
		// both sender and receiver's balance should be tainted.
		attackTx := common.HexToHash(
			"0xc5f6810db332446e147c143b54b5b7a62fe01df9c08d1524c395ac9524353406",
		)
		victimTx := common.HexToHash(
			"0xb7d3c8f8707adf00af83462559199f90321ebf11dd3153a2fb9a4f65c58caefd",
		)
		var attack *Attack
		var window *SearchWindow

		AfterAll(func() {
			if window != nil {
				window.Close()
			}
		})

		It("should be found by searcher", func() {
			window, attack = searchForAttack(attackTx, victimTx, nil)
			Expect(attack).NotTo(BeNil())
			Expect(attack.Attacker).To(Equal(common.HexToAddress(
				"0xE9bB3b9000c006cF5231908e8729d4c04D00D985",
			)))
			Expect(attack.Victim).To(Equal(common.HexToAddress(
				"0x4d31f6B8352F14326f82EE2bd146eaCA9c542F0f",
			)))
		})

		It("should be analyzed", func() {
			analysis, err := attack.Analyze(global.BlockchainReader(), window.TxHistorySession())
			Expect(err).NotTo(HaveOccurred())
			Expect(analysis).To(HaveLen(1))
		})

	})

	Context("UniswapV2 computation alteration non-last profit diff case", Ordered, func() {
		// the taint flows to the value of ether transfer.
		// both sender and receiver's balance should be tainted.
		attackTx := common.HexToHash(
			"0x821601dae232bfbefab673376a3a36e106c0f6be645f6964c7a3d953f091def8",
		)
		victimTx := common.HexToHash(
			"0xcb780b23493fd2e14860cc6e4069fb47236959ab2fdfde26e2f1106ee4d00177",
		)
		var attack *Attack
		var window *SearchWindow

		AfterAll(func() {
			if window != nil {
				window.Close()
			}
		})

		It("should be found by searcher", func() {
			window, attack = searchForAttack(attackTx, victimTx, nil)
			Expect(attack).NotTo(BeNil())
			Expect(attack.Attacker).To(Equal(common.HexToAddress(
				"0xa12171Fecd8e54EA1e4A3DF0f6a0ac14fB6fD987",
			)))
			Expect(attack.Victim).To(Equal(common.HexToAddress(
				"0xe3c2172865B311ee7f5536546F1CF40369F6790A",
			)))
		})

		It("should be analyzed", func() {
			analysis, err := attack.Analyze(global.BlockchainReader(), window.TxHistorySession())
			Expect(err).NotTo(HaveOccurred())
			Expect(analysis).To(HaveLen(1))
		})

	})

	Context(
		"UniswapV3 cross-contract taint propagation via contract storage case",
		Ordered,
		func() {
			// the taint flows to the value of ether transfer.
			// both sender and receiver's balance should be tainted.
			attackTx := common.HexToHash(
				"0x0b7d78b0a0aabc7d7a7c8a4d8835038cdfd0f5a3f62842d70f7b49f4d009fc05",
			)
			victimTx := common.HexToHash(
				"0xfbde75cd437228833692780e80860289e235b92fb5d851f072bf8a30563252e3",
			)
			var attack *Attack
			var window *SearchWindow

			AfterAll(func() {
				if window != nil {
					window.Close()
				}
			})

			It("should be found by searcher", func() {
				window, attack = searchForAttack(attackTx, victimTx, nil)
				Expect(attack).NotTo(BeNil())
				Expect(attack.Attacker).To(Equal(common.HexToAddress(
					"0x5f62593C70069AbB35dFe2B63db969e8906609d6",
				)))
				Expect(attack.Victim).To(Equal(common.HexToAddress(
					"0x99790C5608b082E7400131E21126C749E4B8C1Ab",
				)))
			})

			It("should be analyzed", func() {
				analysis, err := attack.Analyze(
					global.BlockchainReader(),
					window.TxHistorySession(),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(analysis).To(HaveLen(1))
			})

		},
	)

	Context("ConstructAttack", func() {
		DescribeTable(
			"should work on corner cases",
			func(attackTx, victimTx string, profitTx *string) {
				atx := common.HexToHash(attackTx)
				vxt := common.HexToHash(victimTx)
				var ptx *common.Hash
				if profitTx != nil {
					ptx = lo.ToPtr(common.HexToHash(*profitTx))
				}
				attack, err := ConstructAttack(
					global.Ctx(),
					global.BlockchainReader(),
					atx,
					vxt,
					ptx,
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(attack).NotTo(BeNil())
				Expect(len(attack.Analysis)).To(BeNumerically(">", 0))
			},
			Entry(
				"a0",
				"0x0e0a199f1ff2ccde60a2dbf74f2a4c6bec14421f9a81a7339c95bfa19a36b873",
				"0xe53a03f6127c0fa38355c149a99f5f6727706e151fa7fcdb9ee862c0b8f00f52",
				nil,
			),
			Entry(
				"a1",
				"0xdc30b95bedea8e5aa0d3c11e556d1774e36cdb3f78efa889dee73a919da1554e",
				"0x11001a5f09efb7b149c552a39f1ca7b3e55efcf5ef1e9324860cbd9da25fafe1",
				nil,
			),
			Entry(
				"a2",
				"0x44b26747a6d76a8596e5fd0611f5349e26918edb37d338fca73b112d2bf08cd6",
				"0xfa7c21332252a28d0ea2fe35481bfd62e8a318e778f89e8e85672f25d4cbfbb3",
				lo.ToPtr("0xb453c47c514b761c4f5256028133e1150c587cd682c8720282521a6f372ff800"),
			),
			Entry(
				"a3",
				"0x98615ee56b9a448a475c93d95088131c7686719215c5a73dc5f1e965be72ce73",
				"0x0aaead0ec601e35eee0592f8fc33eb6ab4ac1c826f1bb6c02e731b1c5fcb3c54",
				nil,
			),
			Entry(
				"a4",
				"0x102b122fccfc0cc7c5f7f7e22530a4937409eec65a3a49281a52c6ed6251e183",
				"0xa39a3cbdc8fb1c8857b3792af6b4a8704868e6934852ad9d57dc27fb188f8959",
				nil,
			),
			Entry(
				"a5",
				"0xa3373bc11c3f4540fde744dc0f0782c55944cc8edaaf96863e9e69510f6f745f",
				"0xbc1e34de754d488683ca67f5ef4535f1301fcea1b8f568ced93dab9a2f4a82fe",
				nil,
			),
			Entry(
				"a6",
				"0x02c5674e6cb3f1446e908e204cdc4e18cf338c9bde3e8e335cd7fdea445f1b5f",
				"0x3f11f2994c584960fe50bbe6600164abf1df51497cce3dfe6e048eb65a19e95d",
				nil,
			),
			Entry(
				"a7",
				"0xbb5d980ef2fe480677ab11c385bbda54f103a57e88e2cf2b5aa15554266a6a03",
				"0x23711918259881755329687b8aea6c52a6957efa11de43b579a207166c0d1b28",
				lo.ToPtr("0x97bc72b4d61da6a45e1cf6bd773682dc1e53e5b37393451fb19e60fc7504b9de"),
			),
			Entry(
				"a8",
				"0xa3373bc11c3f4540fde744dc0f0782c55944cc8edaaf96863e9e69510f6f745f",
				"0xbc1e34de754d488683ca67f5ef4535f1301fcea1b8f568ced93dab9a2f4a82fe",
				nil,
			),
		)
	})
})
