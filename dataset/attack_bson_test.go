package dataset_test

import (
	"fmt"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/Troublor/erebus-redgiant/analysis/summary"
	. "github.com/Troublor/erebus-redgiant/dataset"
	"github.com/Troublor/erebus-redgiant/global"
)

var _ = Describe("AttackBson", Ordered, func() {
	attackTx := common.HexToHash("0xd402ef75435ddef1fb443b6fe7025f29aa767e51a27f074c1e8f43ef16a8ad1b")
	victimTx := common.HexToHash("0x141f6f6d7809509288b778d2a9bfd6d8a03a7b69164452b8720d9b25b1b9e3c3")
	var attack *Attack
	var attackRaw *Attack
	var err error

	attackExtJson := `{"_id":{"$oid":"6257b2e246342d8c36e64602"},"hash":"0x7cc46766099d1ea6d6b5b543ad42861239c2e8b7a4e85a4210dc5d88d5f6f3a8","block":10046428,"attacker":"0x40DDE6092a77eC2d00eB4fa14f0c5d92d835d673","victim":"0xdd5a1C148Ca114af2F4EBC639ce21fEd4730a608","attackTx":"0xd402ef75435ddef1fb443b6fe7025f29aa767e51a27f074c1e8f43ef16a8ad1b","victimTx":"0x141f6f6d7809509288b778d2a9bfd6d8a03a7b69164452b8720d9b25b1b9e3c3","attackerProfits":{"attack":[{"account":"0x40DDE6092a77eC2d00eB4fa14f0c5d92d835d673","amount":"3199260000000000","type":"ETHER"}],"attackFree":[]},"victimProfits":{"attack":[],"attackFree":[{"account":"0xdd5a1C148Ca114af2F4EBC639ce21fEd4730a608","amount":"3199260000000000","type":"ETHER"}]},"outOfGas":false,"analysis":[{"$oid":"6257b2e246342d8c36e64601"}]}`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  //nolint:lll
	attackBson := hexutil.MustDecode(`0xed020000075f6964006257b2aaf9ec825e7d1754a1026861736800430000003078376363343637363630393964316561366436623562353433616434323836313233396332653862376134653835613432313064633564383864356636663361380012626c6f636b00dc4b9900000000000261747461636b6572002b000000307834304444453630393261373765433264303065423466613134663063356439326438333564363733000276696374696d002b000000307864643561314331343843613131346166324634454243363339636532316645643437333061363038000261747461636b54780043000000307864343032656637353433356464656631666234343362366665373032356632396161373637653531613237663037346331653866343365663136613861643162000276696374696d54780043000000307831343166366636643738303935303932383862373738643261396266643664386130336137623639313634343532623837323064396232356231623965336333000361747461636b657250726f6669747300900000000461747461636b00720000000330006a000000026163636f756e74002b0000003078343044444536303932613737654332643030654234666131346630633564393264383335643637330002616d6f756e74001100000033313939323630303030303030303030000274797065000600000045544845520000000461747461636b46726565000500000000000376696374696d50726f6669747300900000000461747461636b0005000000000461747461636b4672656500720000000330006a000000026163636f756e74002b0000003078646435613143313438436131313461663246344542433633396365323166456434373330613630380002616d6f756e740011000000333139393236303030303030303030300002747970650006000000455448455200000000086f75744f66476173000004616e616c7973697300140000000730006257b2aaf9ec825e7d1754a00000`) //nolint:lll

	BeforeAll(func() {
		attackRaw, err = ConstructAttack(
			global.Ctx(), global.BlockchainReader(),
			attackTx, victimTx, nil,
		)
		if err != nil {
			Fail(fmt.Sprintf("Failed to construct attack: %v", err.Error()))
		}
	})

	BeforeEach(func() {
		attack = &Attack{
			Attacker:            attackRaw.Attacker,
			Victim:              attackRaw.Victim,
			AttackTxRecord:      attackRaw.AttackTxRecord,
			VictimTxRecord:      attackRaw.VictimTxRecord,
			ProfitTxRecord:      attackRaw.ProfitTxRecord,
			AttackTxAsIfSummary: attackRaw.AttackTxAsIfSummary,
			VictimTxAsIfSummary: attackRaw.VictimTxAsIfSummary,
			ProfitTxAsIfSummary: attackRaw.ProfitTxAsIfSummary,
			// OutOfGas:            attackRaw.OutOfGas,
			Analysis: attackRaw.Analysis,
		}
	})

	It("should serialize attack to BSON", func() {
		data, err := bson.Marshal(attack.AsAttackBSON())
		Expect(err).NotTo(HaveOccurred())
		Expect(data).NotTo(BeEmpty())
	})

	It("should deserialize bson attack", func() {
		var a AttackBSON
		registry := bson.NewRegistryBuilder().
			RegisterTypeDecoder(
				reflect.TypeOf((*summary.Profit)(nil)).Elem(),
				&summary.ProfitBsonDecoder{},
			).
			RegisterTypeDecoder(
				reflect.TypeOf((*summary.StateVariable)(nil)).Elem(),
				&summary.StateVariableBsonDecoder{},
			).
			Build()
		err := bson.UnmarshalWithRegistry(registry, attackBson, &a)
		Expect(err).NotTo(HaveOccurred())
		attack, err := a.AsAttack(global.Ctx(), global.BlockchainReader())
		Expect(err).NotTo(HaveOccurred())
		Expect(attack.Attacker).To(Equal(attackRaw.Attacker))
		Expect(attack.Victim).To(Equal(attackRaw.Victim))
	})

	It("should serialize attack to extended JSON", func() {
		data, err := bson.MarshalExtJSON(attack.AsAttackBSON(), false, false)
		Expect(err).NotTo(HaveOccurred())
		Expect(data).NotTo(BeEmpty())
	})

	It("should deserialize extended JSON attack", func() {
		var a AttackBSON
		registry := bson.NewRegistryBuilder().
			RegisterTypeDecoder(
				reflect.TypeOf((*summary.Profit)(nil)).Elem(),
				&summary.ProfitBsonDecoder{},
			).
			RegisterTypeDecoder(
				reflect.TypeOf((*summary.StateVariable)(nil)).Elem(),
				&summary.StateVariableBsonDecoder{},
			).
			Build()
		err := bson.UnmarshalExtJSONWithRegistry(registry, []byte(attackExtJson), false, &a)
		Expect(err).NotTo(HaveOccurred())
		attack, err := a.AsAttack(global.Ctx(), global.BlockchainReader())
		Expect(err).NotTo(HaveOccurred())
		Expect(attack.Attacker).To(Equal(attackRaw.Attacker))
		Expect(attack.Victim).To(Equal(attackRaw.Victim))
	})

})
