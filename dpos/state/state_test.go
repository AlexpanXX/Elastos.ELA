package state

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"

	"github.com/stretchr/testify/assert"
)

// mockBlock creates a block instance by the given height and transactions.
func mockBlock(height uint32, txs ...*types.Transaction) *types.Block {
	return &types.Block{
		Header: types.Header{
			Height: height,
		},
		Transactions: txs,
	}
}

// mockRegisterProducerTx creates a register producer transaction with the given
// ProducerInfo.
func mockRegisterProducerTx(info *payload.ProducerInfo) *types.Transaction {
	return &types.Transaction{
		TxType:  types.RegisterProducer,
		Payload: info,
	}
}

// mockUpdateProducerTx creates a update producer transaction with the given
// ProducerInfo.
func mockUpdateProducerTx(info *payload.ProducerInfo) *types.Transaction {
	return &types.Transaction{
		TxType:  types.UpdateProducer,
		Payload: info,
	}
}

// mockCancelProducerTx creates a cancel producer transaction by the producer
// public key.
func mockCancelProducerTx(publicKey []byte) *types.Transaction {
	return &types.Transaction{
		TxType: types.CancelProducer,
		Payload: &payload.ProcessProducer{
			OwnerPublicKey: publicKey,
		},
	}
}

func mockActivateProducerTx(publicKey []byte) *types.Transaction {
	return &types.Transaction{
		TxType: types.ActivateProducer,
		Payload: &payload.ProcessProducer{
			OwnerPublicKey: publicKey,
		},
	}
}

// mockVoteTx creates a vote transaction with the producers public keys.
func mockVoteTx(publicKeys [][]byte) *types.Transaction {
	output := &types.Output{
		Value: 100,
		Type:  types.OTVote,
		Payload: &outputpayload.VoteOutput{
			Version: 0,
			Contents: []outputpayload.VoteContent{
				{outputpayload.Delegate, publicKeys},
			},
		},
	}

	return &types.Transaction{
		Version: types.TxVersion09,
		TxType:  types.TransferAsset,
		Outputs: []*types.Output{output},
	}
}

// mockVoteTx creates a cancel vote transaction with the previous vote
// transaction.
func mockCancelVoteTx(tx *types.Transaction) *types.Transaction {
	inputs := make([]*types.Input, len(tx.Outputs))
	for i := range tx.Outputs {
		inputs[i] = &types.Input{
			Previous: *types.NewOutPoint(tx.Hash(), uint16(i)),
		}
	}

	return &types.Transaction{
		Version: types.TxVersion09,
		TxType:  types.TransferAsset,
		Inputs:  inputs,
	}
}

// mockIllegalBlockTx creates a illegal block transaction with the producer
// public key.
func mockIllegalBlockTx(publicKey []byte) *types.Transaction {
	return &types.Transaction{
		TxType: types.IllegalBlockEvidence,
		Payload: &payload.DPOSIllegalBlocks{
			Evidence: payload.BlockEvidence{
				Signers: [][]byte{publicKey},
			},
			CompareEvidence: payload.BlockEvidence{
				Signers: [][]byte{publicKey},
			},
		},
	}
}

func TestState_ProcessTransaction(t *testing.T) {
	state := NewState(&config.DefaultParams, nil)
	// Create 10 producers info.
	producers := make([]*payload.ProducerInfo, 10)
	for i, p := range producers {
		p = &payload.ProducerInfo{
			OwnerPublicKey: make([]byte, 33),
			NodePublicKey:  make([]byte, 33),
		}
		for j := range p.OwnerPublicKey {
			p.OwnerPublicKey[j] = byte(i)
		}
		rand.Read(p.NodePublicKey)
		p.NickName = fmt.Sprintf("Producer-%d", i+1)
		producers[i] = p
	}

	// Register each producer on one height.
	for i, p := range producers {
		tx := mockRegisterProducerTx(p)
		state.ProcessBlock(mockBlock(uint32(i+1), tx), nil)
	}

	// At this point, we have 5 pending, 5 active and 10 in total producers.
	if !assert.Equal(t, 5, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 5, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 10, len(state.GetProducers())) {
		t.FailNow()
	}

	// Test update producer.
	producers[0].NickName = "Updated"
	nodePublicKey := make([]byte, 33)
	rand.Read(nodePublicKey)
	producers[0].NodePublicKey = nodePublicKey
	tx := mockUpdateProducerTx(producers[0])
	state.ProcessBlock(mockBlock(11, tx), nil)
	p := state.getProducer(producers[0].NodePublicKey)
	if !assert.NotNil(t, p) {
		t.FailNow()
	}
	if !assert.Equal(t, "Updated", p.info.NickName) {
		t.FailNow()
	}

	// Test cancel producer.
	tx = mockCancelProducerTx(producers[0].OwnerPublicKey)
	state.ProcessBlock(mockBlock(12, tx), nil)
	// at this point, we have 1 canceled, 3 pending, 6 active and 9 in total producers.
	if !assert.Equal(t, 1, len(state.GetCanceledProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 3, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 6, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 9, len(state.GetProducers())) {
		t.FailNow()
	}

	// Test vote producer.
	publicKeys := make([][]byte, 5)
	for i, p := range producers[1:6] {
		publicKeys[i] = p.OwnerPublicKey
	}
	tx = mockVoteTx(publicKeys)
	state.ProcessBlock(mockBlock(13, tx), nil)

	for _, pk := range publicKeys {
		p := state.getProducer(pk)
		if !assert.Equal(t, common.Fixed64(100), p.votes) {
			t.FailNow()
		}
	}

	// Test illegal producer.
	tx = mockIllegalBlockTx(producers[1].OwnerPublicKey)
	state.ProcessBlock(mockBlock(14, tx), nil)
	// at this point, we have 1 canceled, 1 pending, 7 active, 1 illegal and 8 in total producers.
	if !assert.Equal(t, 1, len(state.GetCanceledProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 1, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 7, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 1, len(state.GetIllegalProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 8, len(state.GetProducers())) {
		t.FailNow()
	}
}

func TestState_ProcessBlock(t *testing.T) {
	state := NewState(&config.DefaultParams, nil)

	// Create 100 producers info.
	producers := make([]*payload.ProducerInfo, 100)
	for i, p := range producers {
		p = &payload.ProducerInfo{
			OwnerPublicKey: make([]byte, 33),
			NodePublicKey:  make([]byte, 33),
		}
		for j := range p.OwnerPublicKey {
			p.OwnerPublicKey[j] = byte(i)
		}
		rand.Read(p.NodePublicKey)
		p.NickName = fmt.Sprintf("Producer-%d", i+1)
		producers[i] = p
	}

	// Register 10 producers on one height.
	for i := 0; i < 10; i++ {
		txs := make([]*types.Transaction, 10)
		for i, p := range producers[i*10 : (i+1)*10] {
			txs[i] = mockRegisterProducerTx(p)
		}
		state.ProcessBlock(mockBlock(uint32(i+1), txs...), nil)
	}
	// at this point, we have 50 pending, 50 active and 100 in total producers.
	if !assert.Equal(t, 50, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 50, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 100, len(state.GetProducers())) {
		t.FailNow()
	}

	// Update 10 producers.
	txs := make([]*types.Transaction, 10)
	for i := range txs {
		producers[i].NickName = fmt.Sprintf("Updated-%d", i)
		txs[i] = mockUpdateProducerTx(producers[i])
	}
	state.ProcessBlock(mockBlock(11, txs...), nil)
	for i := range txs {
		p := state.getProducer(producers[i].NodePublicKey)
		if !assert.Equal(t, fmt.Sprintf("Updated-%d", i), p.info.NickName) {
			t.FailNow()
		}
	}

	// Cancel 10 producers.
	txs = make([]*types.Transaction, 10)
	for i := range txs {
		txs[i] = mockCancelProducerTx(producers[i].OwnerPublicKey)
	}
	state.ProcessBlock(mockBlock(12, txs...), nil)
	// at this point, we have 10 canceled, 30 pending, 60 active and 90 in total producers.
	if !assert.Equal(t, 10, len(state.GetCanceledProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 30, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 60, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 90, len(state.GetProducers())) {
		t.FailNow()
	}

	// Vote 10 producers for 10 times.
	publicKeys := make([][]byte, 10)
	for i, p := range producers[10:20] {
		publicKeys[i] = p.OwnerPublicKey
	}
	txs = make([]*types.Transaction, 10)
	for i := range txs {
		txs[i] = mockVoteTx(publicKeys)
	}
	state.ProcessBlock(mockBlock(13, txs...), nil)
	for _, pk := range publicKeys {
		p := state.getProducer(pk)
		if !assert.Equal(t, common.Fixed64(1000), p.votes) {
			t.FailNow()
		}
	}

	// Illegal 10 producers.
	txs = make([]*types.Transaction, 10)
	for i := range txs {
		txs[i] = mockIllegalBlockTx(producers[10+i].OwnerPublicKey)
	}
	state.ProcessBlock(mockBlock(14, txs...), nil)
	// at this point, we have 10 canceled, 10 pending, 70 active, 10 illegal and 80 in total producers.
	if !assert.Equal(t, 10, len(state.GetCanceledProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 10, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 70, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 10, len(state.GetIllegalProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 80, len(state.GetProducers())) {
		t.FailNow()
	}

	// Mixed transactions 1 register, 2 cancel, 3 updates, 4 votes, 5 illegals.
	txs = make([]*types.Transaction, 15)
	info := &payload.ProducerInfo{
		OwnerPublicKey: make([]byte, 33),
		NodePublicKey:  make([]byte, 33),
	}
	for i := range info.OwnerPublicKey {
		info.OwnerPublicKey[i] = byte(101)
	}
	rand.Read(info.NodePublicKey)
	info.NickName = "Producer-101"
	txs[0] = mockRegisterProducerTx(info)

	for i := 0; i < 2; i++ {
		txs[1+i] = mockCancelProducerTx(producers[20+i].OwnerPublicKey)
	}

	for i := 0; i < 3; i++ {
		txs[3+i] = mockUpdateProducerTx(producers[30+i])
	}

	publicKeys = make([][]byte, 4)
	for i, p := range producers[40:44] {
		publicKeys[i] = p.OwnerPublicKey
	}
	for i := 0; i < 4; i++ {
		txs[6+i] = mockVoteTx(publicKeys)
	}

	for i := 0; i < 5; i++ {
		txs[10+i] = mockIllegalBlockTx(producers[50+i].OwnerPublicKey)
	}
	state.ProcessBlock(mockBlock(15, txs...), nil)
	// at this point, we have 12 canceled, 1 pending, 73 active, 15 illegal and 74 in total producers.
	// 10+2
	if !assert.Equal(t, 12, len(state.GetCanceledProducers())) {
		t.FailNow()
	}
	// 20-10+1
	if !assert.Equal(t, 1, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	// 60+10-2-5
	if !assert.Equal(t, 73, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	// 10+5
	if !assert.Equal(t, 15, len(state.GetIllegalProducers())) {
		t.FailNow()
	}
	// 101-12-15
	if !assert.Equal(t, 74, len(state.GetProducers())) {
		t.FailNow()
	}
	for _, pk := range publicKeys {
		p := state.getProducer(pk)
		if !assert.Equal(t, common.Fixed64(400), p.votes) {
			t.FailNow()
		}
	}
}

func TestState_ProcessIllegalBlockEvidence(t *testing.T) {
	state := NewState(&config.DefaultParams, nil)

	// Create 10 producers info.
	producers := make([]*payload.ProducerInfo, 10)
	for i, p := range producers {
		p = &payload.ProducerInfo{
			OwnerPublicKey: make([]byte, 33),
			NodePublicKey:  make([]byte, 33),
		}
		for j := range p.OwnerPublicKey {
			p.OwnerPublicKey[j] = byte(i)
		}
		rand.Read(p.NodePublicKey)
		p.NickName = fmt.Sprintf("Producer-%d", i+1)
		producers[i] = p
	}

	// Register each producer on one height.
	for i, p := range producers {
		tx := mockRegisterProducerTx(p)
		state.ProcessBlock(mockBlock(uint32(i+1), tx), nil)
	}
	// At this point, we have 5 pending, 5 active and 10 in total producers.

	// Make producer 0 illegal.
	tx := mockIllegalBlockTx(producers[0].OwnerPublicKey)
	state.ProcessSpecialTxPayload(tx.Payload)
	// At this point, we have 5 pending, 4 active 1 illegal and 9 in total producers.
	if !assert.Equal(t, 5, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 4, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 1, len(state.GetIllegalProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 9, len(state.GetProducers())) {
		t.FailNow()
	}

	// Process next height, state will rollback illegal producer.
	state.ProcessBlock(mockBlock(11), nil)
	// At this point, we have 4 pending, 6 active and 10 in total producers.
	if !assert.Equal(t, 4, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 6, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 10, len(state.GetProducers())) {
		t.FailNow()
	}
}

func TestState_Rollback(t *testing.T) {
	state := NewState(&config.DefaultParams, nil)

	// Create 10 producers info.
	producers := make([]*payload.ProducerInfo, 10)
	for i, p := range producers {
		p = &payload.ProducerInfo{
			OwnerPublicKey: make([]byte, 33),
			NodePublicKey:  make([]byte, 33),
		}
		for j := range p.OwnerPublicKey {
			p.OwnerPublicKey[j] = byte(i)
		}
		rand.Read(p.NodePublicKey)
		p.NickName = fmt.Sprintf("Producer-%d", i+1)
		producers[i] = p
	}

	// Register each producer on one height.
	for i, p := range producers {
		tx := mockRegisterProducerTx(p)
		state.ProcessBlock(mockBlock(uint32(i+1), tx), nil)
	}
	// At this point, we have 5 pending, 5 active and 10 in total producers.
	if !assert.Equal(t, 5, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 5, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 10, len(state.GetProducers())) {
		t.FailNow()
	}

	err := state.RollbackTo(9)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	// At this point, we have 5 pending, 4 active and 9 in total producers.
	if !assert.Equal(t, 5, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 4, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 9, len(state.GetProducers())) {
		t.FailNow()
	}
}

func TestState_GetHistory(t *testing.T) {
	state := NewState(&config.DefaultParams, nil)

	// Create 10 producers info.
	producers := make([]*payload.ProducerInfo, 10)
	for i, p := range producers {
		p = &payload.ProducerInfo{
			OwnerPublicKey: make([]byte, 33),
			NodePublicKey:  make([]byte, 33),
		}
		for j := range p.OwnerPublicKey {
			p.OwnerPublicKey[j] = byte(i)
		}
		rand.Read(p.NodePublicKey)
		p.NickName = fmt.Sprintf("Producer-%d", i+1)
		producers[i] = p
	}

	// Register each producer on one height.
	for i, p := range producers {
		tx := mockRegisterProducerTx(p)
		state.ProcessBlock(mockBlock(uint32(i+1), tx), nil)
	}
	// At this point, we have 5 pending, 5 active and 10 in total producers.

	// Test update producer.
	producers[0].NickName = "Updated"
	nodePublicKey := make([]byte, 33)
	rand.Read(nodePublicKey)
	producers[0].NodePublicKey = nodePublicKey
	tx := mockUpdateProducerTx(producers[0])
	state.ProcessBlock(mockBlock(11, tx), nil)
	p := state.getProducer(producers[0].NodePublicKey)
	if !assert.NotNil(t, p) {
		t.FailNow()
	}
	if !assert.Equal(t, "Updated", p.info.NickName) {
		t.FailNow()
	}

	// Test cancel producer.
	tx = mockCancelProducerTx(producers[0].OwnerPublicKey)
	state.ProcessBlock(mockBlock(12, tx), nil)
	// At this point, we have 1 canceled, 3 pending, 6 active and 9 in total producers.

	// Test vote producer.
	publicKeys := make([][]byte, 5)
	for i, p := range producers[1:6] {
		publicKeys[i] = p.OwnerPublicKey
	}
	tx = mockVoteTx(publicKeys)
	state.ProcessBlock(mockBlock(13, tx), nil)
	for _, pk := range publicKeys {
		p := state.getProducer(pk)
		if !assert.Equal(t, common.Fixed64(100), p.votes) {
			t.FailNow()
		}
	}

	// Test illegal producer.
	tx = mockIllegalBlockTx(producers[1].OwnerPublicKey)
	state.ProcessBlock(mockBlock(14, tx), nil)
	// At this point, we have 1 canceled, 1 pending, 7 active, 1 illegal and 8 in total producers.

	_, err := state.GetHistory(0)
	limitHeight := state.history.height - uint32(len(state.history.changes))
	if !assert.EqualError(t, err, fmt.Sprintf("seek to %d overflow"+
		" history capacity, at most seek to %d", 0, limitHeight)) {
		t.FailNow()
	}

	s, err := state.GetHistory(10)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	// At this point, we have 5 pending and 5 in total producers.
	if !assert.Equal(t, 5, len(s.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 5, len(s.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 10, len(s.GetProducers())) {
		t.FailNow()
	}

	s, err = state.GetHistory(14)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	// At this point, we have 1 canceled, 1 pending, 7 active, 1 illegal and 8 in total producers.
	if !assert.Equal(t, 1, len(state.GetCanceledProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 1, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 7, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 1, len(state.GetIllegalProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 8, len(state.GetProducers())) {
		t.FailNow()
	}

	s, err = state.GetHistory(12)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	// At this point, we have 1 canceled, 3 pending, 6 active and 9 in total producers.
	if !assert.Equal(t, 1, len(s.GetCanceledProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 3, len(s.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 6, len(s.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 9, len(s.GetProducers())) {
		t.FailNow()
	}

	// Process a new height see if state go to best height.
	state.ProcessBlock(mockBlock(15, tx), nil)
	// At this point, we have 1 canceled, 0 pending, 8 active, 1 illegal and 8 in total producers.
	if !assert.Equal(t, 1, len(state.GetCanceledProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 0, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 8, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 1, len(state.GetIllegalProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 8, len(state.GetProducers())) {
		t.FailNow()
	}

}

func TestState_NicknameExists(t *testing.T) {
	state := NewState(&config.DefaultParams, nil)

	// Create 10 producers info.
	producers := make([]*payload.ProducerInfo, 10)
	for i, p := range producers {
		p = &payload.ProducerInfo{
			OwnerPublicKey: make([]byte, 33),
			NodePublicKey:  make([]byte, 33),
		}
		for j := range p.OwnerPublicKey {
			p.OwnerPublicKey[j] = byte(i)
		}
		rand.Read(p.NodePublicKey)
		p.NickName = fmt.Sprintf("Producer-%d", i+1)
		producers[i] = p
	}

	// Register each producer on one height.
	for i, p := range producers {
		tx := mockRegisterProducerTx(p)
		state.ProcessBlock(mockBlock(uint32(i+1), tx), nil)
	}

	for i := range producers {
		if !assert.Equal(t, true, state.NicknameExists(
			fmt.Sprintf("Producer-%d", i+1))) {
			t.FailNow()
		}
	}

	// Change producer-1 nickname to Updated.
	producers[0].NickName = "Updated"
	nodePublicKey := make([]byte, 33)
	rand.Read(nodePublicKey)
	producers[0].NodePublicKey = nodePublicKey
	tx := mockUpdateProducerTx(producers[0])
	state.ProcessBlock(mockBlock(11, tx), nil)
	p := state.getProducer(producers[0].NodePublicKey)
	if !assert.NotNil(t, p) {
		t.FailNow()
	}
	if !assert.Equal(t, "Updated", p.info.NickName) {
		t.FailNow()
	}

	if !assert.Equal(t, false, state.NicknameExists("Producer-1")) {
		t.FailNow()
	}

	// Cancel producer-2, see if nickname change to unused.
	tx = mockCancelProducerTx(producers[1].OwnerPublicKey)
	state.ProcessBlock(mockBlock(12, tx), nil)
	if !assert.Equal(t, false, state.NicknameExists("Producer-2")) {
		t.FailNow()
	}

	// Make producer-3 illegal, see if nickname change to unused.
	tx = mockIllegalBlockTx(producers[2].OwnerPublicKey)
	state.ProcessSpecialTxPayload(tx.Payload)
	if !assert.Equal(t, false, state.NicknameExists("Producer-3")) {
		t.FailNow()
	}
}

func TestState_ProducerExists(t *testing.T) {
	state := NewState(&config.DefaultParams, nil)

	// Create 10 producers info.
	producers := make([]*payload.ProducerInfo, 10)
	for i, p := range producers {
		p = &payload.ProducerInfo{
			OwnerPublicKey: make([]byte, 33),
			NodePublicKey:  make([]byte, 33),
		}
		for j := range p.OwnerPublicKey {
			p.OwnerPublicKey[j] = byte(i)
		}
		rand.Read(p.NodePublicKey)
		p.NickName = fmt.Sprintf("Producer-%d", i+1)
		producers[i] = p
	}

	// Register each producer on one height.
	for i, p := range producers {
		tx := mockRegisterProducerTx(p)
		state.ProcessBlock(mockBlock(uint32(i+1), tx), nil)
	}

	for _, p := range producers {
		if !assert.Equal(t, true, state.ProducerExists(p.NodePublicKey)) {
			t.FailNow()
		}
		if !assert.Equal(t, true, state.ProducerExists(p.OwnerPublicKey)) {
			t.FailNow()
		}
	}

	// Change producer node public key
	oldPublicKey := producers[0].NodePublicKey
	producers[0].NodePublicKey = make([]byte, 33)
	rand.Read(producers[0].NodePublicKey)
	tx := mockUpdateProducerTx(producers[0])
	state.ProcessBlock(mockBlock(11, tx), nil)
	if !assert.Equal(t, true, state.ProducerExists(producers[0].NodePublicKey)) {
		t.FailNow()
	}
	if !assert.Equal(t, false, state.ProducerExists(oldPublicKey)) {
		t.FailNow()
	}

	// Canceled producer also existed.
	tx = mockCancelProducerTx(producers[0].OwnerPublicKey)
	state.ProcessBlock(mockBlock(12, tx), nil)
	if !assert.Equal(t, true, state.ProducerExists(producers[0].OwnerPublicKey)) {
		t.FailNow()
	}
}

func TestState_IsDPOSTransaction(t *testing.T) {
	state := NewState(&config.DefaultParams, nil)

	producer := &payload.ProducerInfo{
		OwnerPublicKey: make([]byte, 33),
		NodePublicKey:  make([]byte, 33),
		NickName:       "Producer",
	}
	rand.Read(producer.OwnerPublicKey)
	rand.Read(producer.NodePublicKey)

	tx := mockRegisterProducerTx(producer)
	if !assert.Equal(t, true, state.IsDPOSTransaction(tx)) {
		t.FailNow()
	}
	state.ProcessBlock(mockBlock(1, tx), nil)
	for i := uint32(2); i < 10; i++ {
		state.ProcessBlock(mockBlock(i), nil)
	}

	tx = mockUpdateProducerTx(producer)
	if !assert.Equal(t, true, state.IsDPOSTransaction(tx)) {
		t.FailNow()
	}

	tx = mockCancelProducerTx(producer.OwnerPublicKey)
	if !assert.Equal(t, true, state.IsDPOSTransaction(tx)) {
		t.FailNow()
	}

	tx = mockVoteTx([][]byte{producer.OwnerPublicKey})
	if !assert.Equal(t, true, state.IsDPOSTransaction(tx)) {
		t.FailNow()
	}
	state.ProcessBlock(mockBlock(10, tx), nil)
	p := state.getProducer(producer.NodePublicKey)
	if !assert.Equal(t, common.Fixed64(100), p.votes) {
		t.FailNow()
	}

	tx = mockCancelVoteTx(tx)
	if !assert.Equal(t, true, state.IsDPOSTransaction(tx)) {
		t.FailNow()
	}
	state.ProcessBlock(mockBlock(11, tx), nil)
	p = state.getProducer(producer.OwnerPublicKey)
	if !assert.Equal(t, common.Fixed64(0), p.votes) {
		t.FailNow()
	}

	tx = mockIllegalBlockTx(producer.OwnerPublicKey)
	if !assert.Equal(t, true, state.IsDPOSTransaction(tx)) {
		t.FailNow()
	}
}

func TestState_InactiveProducer_Normal(t *testing.T) {
	arbitrators := &ArbitratorsMock{}
	state := NewState(&config.DefaultParams, arbitrators.GetArbitrators)

	// Create 10 producers info.
	producers := make([]*payload.ProducerInfo, 10)
	for i, p := range producers {
		p = &payload.ProducerInfo{
			OwnerPublicKey: make([]byte, 33),
			NodePublicKey:  make([]byte, 33),
		}
		for j := range p.OwnerPublicKey {
			p.OwnerPublicKey[j] = byte(i)
		}
		rand.Read(p.NodePublicKey)
		p.NickName = fmt.Sprintf("Producer-%d", i+1)
		producers[i] = p
	}

	// Register each producer on one height.
	for i, p := range producers {
		tx := mockRegisterProducerTx(p)
		state.ProcessBlock(mockBlock(uint32(i+1), tx), nil)
	}

	// At this point, we have 5 pending, 5 active and 10 in total producers.
	if !assert.Equal(t, 5, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 5, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 10, len(state.GetProducers())) {
		t.FailNow()
	}

	// arbitrators should set inactive after continuous three blocks
	arbitrators.CurrentArbitrators = [][]byte{
		producers[0].NodePublicKey,
		producers[1].NodePublicKey,
		producers[2].NodePublicKey,
		producers[3].NodePublicKey,
		producers[4].NodePublicKey,
	}

	currentHeight := 11
	config.DefaultParams.PublicDPOSHeight = 11
	config.DefaultParams.MaxInactiveRounds = 10

	// simulate producers[0] do not sign for continuous 11 blocks
	for round := 0; round < 3; round++ {
		for arIndex := 1; arIndex <= 4; arIndex++ {
			state.ProcessBlock(mockBlock(uint32(currentHeight)),
				&payload.Confirm{
					Proposal: payload.DPOSProposal{
						Sponsor: producers[arIndex].NodePublicKey,
					},
					Votes: []payload.DPOSProposalVote{
						{
							Signer: producers[arIndex].NodePublicKey,
						},
					},
				})
			currentHeight++
		}
	}

	// only producer[0] will be inactive
	if !assert.Equal(t, 1, len(state.GetInactiveProducers())) ||
		!assert.True(t, state.isInactiveProducer(producers[0].NodePublicKey)) {
		t.FailNow()
	}

	// check penalty
	inactiveProducer := state.GetProducer(producers[0].NodePublicKey)
	if !assert.Equal(t, inactiveProducer.Penalty(),
		state.chainParams.InactivePenalty) {
		t.FailNow()
	}
}

func TestState_InactiveProducer_FailNoContinuous(t *testing.T) {
	arbitrators := &ArbitratorsMock{}
	state := NewState(&config.DefaultParams, arbitrators.GetArbitrators)

	// Create 10 producers info.
	producers := make([]*payload.ProducerInfo, 10)
	for i, p := range producers {
		p = &payload.ProducerInfo{
			OwnerPublicKey: make([]byte, 33),
			NodePublicKey:  make([]byte, 33),
		}
		for j := range p.OwnerPublicKey {
			p.OwnerPublicKey[j] = byte(i)
		}
		rand.Read(p.NodePublicKey)
		p.NickName = fmt.Sprintf("Producer-%d", i+1)
		producers[i] = p
	}

	// Register each producer on one height.
	for i, p := range producers {
		tx := mockRegisterProducerTx(p)
		state.ProcessBlock(mockBlock(uint32(i+1), tx), nil)
	}

	// At this point, we have 5 pending, 5 active and 10 in total producers.
	if !assert.Equal(t, 5, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 5, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 10, len(state.GetProducers())) {
		t.FailNow()
	}

	// arbitrators should set inactive after continuous three blocks
	arbitrators.CurrentArbitrators = [][]byte{
		producers[0].NodePublicKey,
		producers[1].NodePublicKey,
		producers[2].NodePublicKey,
		producers[3].NodePublicKey,
		producers[4].NodePublicKey,
	}

	currentHeight := 11
	config.DefaultParams.PublicDPOSHeight = 11
	config.DefaultParams.MaxInactiveRounds = 10

	// simulate producers[0] do not sign for over 10 blocks,
	// but is not continuous
	for round := 0; round < 4; round++ {
		for arIndex := 1; arIndex <= 4; arIndex++ {

			if round == 2 && arIndex == 4 {
				state.ProcessBlock(mockBlock(uint32(currentHeight)),
					&payload.Confirm{
						Proposal: payload.DPOSProposal{
							Sponsor: producers[0].NodePublicKey,
						},
						Votes: []payload.DPOSProposalVote{
							{
								Signer: producers[0].NodePublicKey,
							},
						},
					})
			} else {
				state.ProcessBlock(mockBlock(uint32(currentHeight)),
					&payload.Confirm{
						Proposal: payload.DPOSProposal{
							Sponsor: producers[arIndex].NodePublicKey,
						},
						Votes: []payload.DPOSProposalVote{
							{
								Signer: producers[arIndex].NodePublicKey,
							},
						},
					})
			}
			currentHeight++
		}
	}

	// only producer[0] will be inactive
	if !assert.Equal(t, 0, len(state.GetInactiveProducers())) {
		t.FailNow()
	}
}

func TestState_InactiveProducer_RecoverFromInactiveState(t *testing.T) {
	arbitrators := &ArbitratorsMock{}
	state := NewState(&config.DefaultParams, arbitrators.GetArbitrators)

	// Create 10 producers info.
	producers := make([]*payload.ProducerInfo, 10)
	for i, p := range producers {
		p = &payload.ProducerInfo{
			OwnerPublicKey: make([]byte, 33),
			NodePublicKey:  make([]byte, 33),
		}
		for j := range p.OwnerPublicKey {
			p.OwnerPublicKey[j] = byte(i)
		}
		rand.Read(p.NodePublicKey)
		p.NickName = fmt.Sprintf("Producer-%d", i+1)
		producers[i] = p
	}

	// Register each producer on one height.
	for i, p := range producers {
		tx := mockRegisterProducerTx(p)
		state.ProcessBlock(mockBlock(uint32(i+1), tx), nil)
	}

	// At this point, we have 5 pending, 5 active and 10 in total producers.
	if !assert.Equal(t, 5, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 5, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 10, len(state.GetProducers())) {
		t.FailNow()
	}

	// arbitrators should set inactive after continuous three blocks
	arbitrators.CurrentArbitrators = [][]byte{
		producers[0].NodePublicKey,
		producers[1].NodePublicKey,
		producers[2].NodePublicKey,
		producers[3].NodePublicKey,
		producers[4].NodePublicKey,
	}

	currentHeight := 11
	config.DefaultParams.PublicDPOSHeight = 11
	config.DefaultParams.MaxInactiveRounds = 10

	// simulate producers[0] do not sign for continuous 11 blocks
	for round := 0; round < 3; round++ {
		for arIndex := 1; arIndex <= 4; arIndex++ {
			state.ProcessBlock(mockBlock(uint32(currentHeight)),
				&payload.Confirm{
					Proposal: payload.DPOSProposal{
						Sponsor: producers[arIndex].NodePublicKey,
					},
					Votes: []payload.DPOSProposalVote{
						{
							Signer: producers[arIndex].NodePublicKey,
						},
					},
				})
			currentHeight++
		}
	}

	// only producer[0] will be inactive
	if !assert.Equal(t, 1, len(state.GetInactiveProducers())) ||
		!assert.True(t, state.isInactiveProducer(producers[0].NodePublicKey)) {
		t.FailNow()
	}

	// check penalty
	inactiveProducer := state.GetProducer(producers[0].NodePublicKey)
	if !assert.Equal(t, inactiveProducer.Penalty(),
		state.chainParams.InactivePenalty) {
		t.FailNow()
	}

	// request for activating
	state.ProcessBlock(mockBlock(uint32(currentHeight),
		mockActivateProducerTx(producers[0].OwnerPublicKey)), nil)
	currentHeight++

	// producer[0] will not be active util 6 blocks later
	for i := 0; i < 4; i++ {
		state.ProcessBlock(mockBlock(uint32(currentHeight)), nil)
		if !assert.Equal(t, 1, len(state.GetInactiveProducers())) {
			t.FailNow()
		}
		currentHeight++
	}
	state.ProcessBlock(mockBlock(uint32(currentHeight)), nil)
	if !assert.Equal(t, 0, len(state.GetInactiveProducers())) {
		t.FailNow()
	}
}
