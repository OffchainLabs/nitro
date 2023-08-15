package blobs

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/params"
)

/*
// Compressed BLS12-381 G1 element
type KZGCommitment [48]byte

func (p *KZGCommitment) Deserialize(dr *codec.DecodingReader) error {
	if p == nil {
		return errors.New("nil pubkey")
	}
	_, err := dr.Read(p[:])
	return err
}

func (p *KZGCommitment) Serialize(w *codec.EncodingWriter) error {
	return w.Write(p[:])
}

func (KZGCommitment) ByteLength() uint64 {
	return 48
}

func (KZGCommitment) FixedLength() uint64 {
	return 48
}

func (p KZGCommitment) HashTreeRoot(hFn tree.HashFn) tree.Root {
	var a, b tree.Root
	copy(a[:], p[0:32])
	copy(b[:], p[32:48])
	return hFn(a, b)
}

func (p KZGCommitment) MarshalText() ([]byte, error) {
	return []byte("0x" + hex.EncodeToString(p[:])), nil
}

func (p KZGCommitment) String() string {
	return "0x" + hex.EncodeToString(p[:])
}

func (p *KZGCommitment) UnmarshalText(text []byte) error {
	return hexutil.UnmarshalFixedText("KZGCommitment", text, p[:])
}

func (p *KZGCommitment) Point() (*bls.G1Point, error) {
	return bls.FromCompressedG1(p[:])
}

func (kzg KZGCommitment) ComputeVersionedHash() common.Hash {
	h := crypto.Keccak256Hash(kzg[:])
	h[0] = params.BlobCommitmentVersionKZG
	return h
}

// Compressed BLS12-381 G1 element
type KZGProof [48]byte

func (p *KZGProof) Deserialize(dr *codec.DecodingReader) error {
	if p == nil {
		return errors.New("nil pubkey")
	}
	_, err := dr.Read(p[:])
	return err
}

func (p *KZGProof) Serialize(w *codec.EncodingWriter) error {
	return w.Write(p[:])
}

func (KZGProof) ByteLength() uint64 {
	return 48
}

func (KZGProof) FixedLength() uint64 {
	return 48
}

func (p KZGProof) HashTreeRoot(hFn tree.HashFn) tree.Root {
	var a, b tree.Root
	copy(a[:], p[0:32])
	copy(b[:], p[32:48])
	return hFn(a, b)
}

func (p KZGProof) MarshalText() ([]byte, error) {
	return []byte("0x" + hex.EncodeToString(p[:])), nil
}

func (p KZGProof) String() string {
	return "0x" + hex.EncodeToString(p[:])
}

func (p *KZGProof) UnmarshalText(text []byte) error {
	return hexutil.UnmarshalFixedText("KZGProof", text, p[:])
}

func (p *KZGProof) Point() (*bls.G1Point, error) {
	return bls.FromCompressedG1(p[:])
}

type BLSFieldElement [32]byte

func (p BLSFieldElement) MarshalText() ([]byte, error) {
	return []byte("0x" + hex.EncodeToString(p[:])), nil
}

func (p BLSFieldElement) String() string {
	return "0x" + hex.EncodeToString(p[:])
}

func (p *BLSFieldElement) UnmarshalText(text []byte) error {
	return hexutil.UnmarshalFixedText("BLSFieldElement", text, p[:])
}

// Blob data
type Blob [params.FieldElementsPerBlob]BLSFieldElement

func (blob *Blob) Deserialize(dr *codec.DecodingReader) error {
	if blob == nil {
		return errors.New("cannot decode ssz into nil Blob")
	}
	for i := uint64(0); i < params.FieldElementsPerBlob; i++ {
		// TODO: do we want to check if each field element is within range?
		if _, err := dr.Read(blob[i][:]); err != nil {
			return err
		}
	}
	return nil
}

func (blob *Blob) Serialize(w *codec.EncodingWriter) error {
	for i := range blob {
		if err := w.Write(blob[i][:]); err != nil {
			return err
		}
	}
	return nil
}

func (blob *Blob) ByteLength() (out uint64) {
	return params.FieldElementsPerBlob * 32
}

func (blob *Blob) FixedLength() uint64 {
	return params.FieldElementsPerBlob * 32
}

func (blob *Blob) HashTreeRoot(hFn tree.HashFn) tree.Root {
	return hFn.ComplexVectorHTR(func(i uint64) tree.HTR {
		return (*tree.Root)(&blob[i])
	}, params.FieldElementsPerBlob)
}

func (blob *Blob) MarshalText() ([]byte, error) {
	out := make([]byte, 2+params.FieldElementsPerBlob*32*2)
	copy(out[:2], "0x")
	j := 2
	for _, elem := range blob {
		hex.Encode(out[j:j+64], elem[:])
		j += 64
	}
	return out, nil
}

func (blob *Blob) String() string {
	v, err := blob.MarshalText()
	if err != nil {
		return "<invalid-blob>"
	}
	return string(v)
}

func (blob *Blob) UnmarshalText(text []byte) error {
	if blob == nil {
		return errors.New("cannot decode text into nil Blob")
	}
	l := 2 + params.FieldElementsPerBlob*32*2
	if len(text) != l {
		return fmt.Errorf("expected %d characters but got %d", l, len(text))
	}
	if !(text[0] == '0' && text[1] == 'x') {
		return fmt.Errorf("expected '0x' prefix in Blob string")
	}
	j := 0
	for i := 2; i < l; i += 64 {
		if _, err := hex.Decode(blob[j][:], text[i:i+64]); err != nil {
			return fmt.Errorf("blob item %d is not formatted correctly: %w", j, err)
		}
		j += 1
	}
	return nil
}

// Parse blob into Fr elements array
func (blob *Blob) Parse() (out []bls.Fr, err error) {
	out = make([]bls.Fr, params.FieldElementsPerBlob)
	for i, chunk := range blob {
		ok := bls.FrFrom32(&out[i], chunk)
		if !ok {
			return nil, errors.New("internal error commitments")
		}
	}
	return out, nil
}

type BlobKzgs []KZGCommitment

// Extract the crypto material underlying these commitments
func (li BlobKzgs) Parse() ([]*bls.G1Point, error) {
	out := make([]*bls.G1Point, len(li))
	for i, c := range li {
		p, err := c.Point()
		if err != nil {
			return nil, fmt.Errorf("failed to parse commitment %d: %w", i, err)
		}
		out[i] = p
	}
	return out, nil
}

func (li *BlobKzgs) Deserialize(dr *codec.DecodingReader) error {
	return dr.List(func() codec.Deserializable {
		i := len(*li)
		*li = append(*li, KZGCommitment{})
		return &(*li)[i]
	}, 48, params.MaxBlobsPerBlock)
}

func (li BlobKzgs) Serialize(w *codec.EncodingWriter) error {
	return w.List(func(i uint64) codec.Serializable {
		return &li[i]
	}, 48, uint64(len(li)))
}

func (li BlobKzgs) ByteLength() uint64 {
	return uint64(len(li)) * 48
}

func (li *BlobKzgs) FixedLength() uint64 {
	return 0
}

func (li BlobKzgs) HashTreeRoot(hFn tree.HashFn) tree.Root {
	return hFn.ComplexListHTR(func(i uint64) tree.HTR {
		return &li[i]
	}, uint64(len(li)), params.MaxBlobsPerBlock)
}

type Blobs []Blob

func (a *Blobs) Deserialize(dr *codec.DecodingReader) error {
	return dr.List(func() codec.Deserializable {
		i := len(*a)
		*a = append(*a, Blob{})
		return &(*a)[i]
	}, params.FieldElementsPerBlob*32, params.FieldElementsPerBlob)
}

func (a Blobs) Serialize(w *codec.EncodingWriter) error {
	return w.List(func(i uint64) codec.Serializable {
		return &a[i]
	}, params.FieldElementsPerBlob*32, uint64(len(a)))
}

func (a Blobs) ByteLength() (out uint64) {
	return uint64(len(a)) * params.FieldElementsPerBlob * 32
}

func (a *Blobs) FixedLength() uint64 {
	return 0 // it's a list, no fixed length
}

func (li Blobs) HashTreeRoot(hFn tree.HashFn) tree.Root {
	length := uint64(len(li))
	return hFn.ComplexListHTR(func(i uint64) tree.HTR {
		if i < length {
			return &li[i]
		}
		return nil
	}, length, params.MaxBlobsPerBlock)
}

type BlobsAndCommitments struct {
	Blobs    Blobs
	BlobKzgs BlobKzgs
}

func (b *BlobsAndCommitments) HashTreeRoot(hFn tree.HashFn) tree.Root {
	return hFn.HashTreeRoot(&b.Blobs, &b.BlobKzgs)
}

func (b *BlobsAndCommitments) Serialize(w *codec.EncodingWriter) error {
	return w.Container(&b.Blobs, &b.BlobKzgs)
}

func (b *BlobsAndCommitments) ByteLength() uint64 {
	return codec.ContainerLength(&b.Blobs, &b.BlobKzgs)
}

func (b *BlobsAndCommitments) FixedLength() uint64 {
	return 0
}

type PolynomialAndCommitment struct {
	b Blob
	c KZGCommitment
}

func (p *PolynomialAndCommitment) HashTreeRoot(hFn tree.HashFn) tree.Root {
	return hFn.HashTreeRoot(&p.b, &p.c)
}

func (p *PolynomialAndCommitment) Serialize(w *codec.EncodingWriter) error {
	return w.Container(&p.b, &p.c)
}

func (p *PolynomialAndCommitment) ByteLength() uint64 {
	return codec.ContainerLength(&p.b, &p.c)
}

func (p *PolynomialAndCommitment) FixedLength() uint64 {
	return 0
}

type BlobTxWrapper struct {
	Tx                 types.SignedBlobTx
	BlobKzgs           BlobKzgs
	Blobs              Blobs
	KzgAggregatedProof KZGProof
}

func (txw *BlobTxWrapper) Deserialize(dr *codec.DecodingReader) error {
	return dr.Container(&txw.Tx, &txw.BlobKzgs, &txw.Blobs, &txw.KzgAggregatedProof)
}

func (txw *BlobTxWrapper) Serialize(w *codec.EncodingWriter) error {
	return w.Container(&txw.Tx, &txw.BlobKzgs, &txw.Blobs, &txw.KzgAggregatedProof)
}

func (txw *BlobTxWrapper) ByteLength() uint64 {
	return codec.ContainerLength(&txw.Tx, &txw.BlobKzgs, &txw.Blobs, &txw.KzgAggregatedProof)
}

func (txw *BlobTxWrapper) FixedLength() uint64 {
	return 0
}

func (txw *BlobTxWrapper) HashTreeRoot(hFn tree.HashFn) tree.Root {
	return hFn.HashTreeRoot(&txw.Tx, &txw.BlobKzgs, &txw.Blobs, &txw.KzgAggregatedProof)
}

type BlobTxWrapData struct {
	BlobKzgs           BlobKzgs
	Blobs              Blobs
	KzgAggregatedProof KZGProof
}

func ToGethWrapData(b *BlobTxWrapData) types.TxWrapData {
	proof := [48]byte{}
	copy(proof[:], b.KzgAggregatedProof[:])
	blobs := make(types.Blobs, len(b.Blobs))
	for i, blob := range b.Blobs {
		rawBlob := types.Blob{}
		for j, val := range blob {
			rawBlob[j] = types.BLSFieldElement(val)
		}
		blobs[i] = rawBlob
	}
	commits := make([]types.KZGCommitment, len(b.BlobKzgs))
	for i, commit := range b.BlobKzgs {
		commits[i] = types.KZGCommitment(commit)
	}
	return &types.BlobTxWrapData{
		BlobKzgs:           commits,
		Blobs:              blobs,
		KzgAggregatedProof: proof,
	}
}
*/

// EncodeBlobs takes in raw bytes data to convert into blobs used for KZG commitment EIP-4844
// transactions on Ethereum.
func EncodeBlobs(data []byte) []kzg4844.Blob {
	blobs := []kzg4844.Blob{{}}
	blobIndex := 0
	fieldIndex := -1
	for i := 0; i < len(data); i += 31 {
		fieldIndex++
		if fieldIndex == params.BlobTxFieldElementsPerBlob {
			blobs = append(blobs, kzg4844.Blob{})
			blobIndex++
			fieldIndex = 0
		}
		max := i + 31
		if max > len(data) {
			max = len(data)
		}
		copy(blobs[blobIndex][fieldIndex*32+1:], data[i:max])
	}
	return blobs
}

/*
func (blob *Blob) ComputeCommitment() (commitment KZGCommitment, ok bool) {
	frs := make([]bls.Fr, len(blob))
	for i, elem := range blob {
		if !bls.FrFrom32(&frs[i], elem) {
			return KZGCommitment{}, false
		}
	}
	// data is presented in eval form
	commitmentG1 := kzg.BlobToKzg(frs)
	var out KZGCommitment
	copy(out[:], bls.ToCompressedG1(commitmentG1))
	return out, true
}

// Extract the crypto material underlying these blobs
func (blobs Blobs) Parse() ([][]bls.Fr, error) {
	out := make([][]bls.Fr, len(blobs))
	for i, b := range blobs {
		blob, err := b.Parse()
		if err != nil {
			return nil, fmt.Errorf("failed to parse blob %d: %w", i, err)
		}
		out[i] = blob
	}
	return out, nil
}

// Return KZG commitments and versioned hashes that correspond to these blobs
func (blobs Blobs) ComputeCommitments() (commitments []KZGCommitment, versionedHashes []common.Hash, ok bool) {
	commitments = make([]KZGCommitment, len(blobs))
	versionedHashes = make([]common.Hash, len(blobs))
	for i, blob := range blobs {
		commitments[i], ok = blob.ComputeCommitment()
		if !ok {
			return nil, nil, false
		}
		versionedHashes[i] = commitments[i].ComputeVersionedHash()
	}
	return commitments, versionedHashes, true
}
*/

// Return KZG commitments, proofs, and versioned hashes that corresponds to these blobs
func ComputeCommitmentsProofsAndHashes(blobs []kzg4844.Blob) ([]kzg4844.Commitment, []kzg4844.Proof, []common.Hash, error) {
	commitments := make([]kzg4844.Commitment, len(blobs))
	proofs := make([]kzg4844.Proof, len(blobs))
	versionedHashes := make([]common.Hash, len(blobs))

	for i := range blobs {
		var err error
		commitments[i], err = kzg4844.BlobToCommitment(blobs[i])
		if err != nil {
			return nil, nil, nil, err
		}
		proofs[i], err = kzg4844.ComputeBlobProof(blobs[i], commitments[i])
		if err != nil {
			return nil, nil, nil, err
		}
		versionedHashes[i] = vm.KZGToVersionedHash(commitments[i])
	}

	return commitments, proofs, versionedHashes, nil
}

/*

func computePowers(r *bls.Fr, n int) []bls.Fr {
	var currentPower bls.Fr
	bls.AsFr(&currentPower, 1)
	powers := make([]bls.Fr, n)
	for i := range powers {
		powers[i] = currentPower
		bls.MulModFr(&currentPower, &currentPower, r)
	}
	return powers
}

func computeAggregateKzgCommitment(blobs Blobs, commitments []KZGCommitment) ([]bls.Fr, *bls.G1Point, error) {
	// create challenges
	sum, err := sszHash(&BlobsAndCommitments{blobs, commitments})
	if err != nil {
		return nil, nil, err
	}
	var r bls.Fr
	hashToFr(&r, sum)

	powers := computePowers(&r, len(blobs))

	commitmentsG1 := make([]bls.G1Point, len(commitments))
	for i := 0; i < len(commitmentsG1); i++ {
		p, _ := commitments[i].Point()
		bls.CopyG1(&commitmentsG1[i], p)
	}
	aggregateCommitmentG1 := bls.LinCombG1(commitmentsG1, powers)
	var aggregateCommitment KZGCommitment
	copy(aggregateCommitment[:], bls.ToCompressedG1(aggregateCommitmentG1))

	polys, err := blobs.Parse()
	if err != nil {
		return nil, nil, err
	}
	aggregatePoly := kzg.MatrixLinComb(polys, powers)
	return aggregatePoly, aggregateCommitmentG1, nil
}

func hashToFr(out *bls.Fr, h [32]byte) {
	// re-interpret as little-endian
	var b [32]byte = h
	for i := 0; i < 16; i++ {
		b[31-i], b[i] = b[i], b[31-i]
	}
	zB := new(big.Int).Mod(new(big.Int).SetBytes(b[:]), kzg.BLSModulus)
	_ = kzg.BigToFr(out, zB)
}

// sszHash returns the hash ot the raw serialized ssz-container (i.e. without merkelization)
func sszHash(c codec.Serializable) ([32]byte, error) {
	sha := sha256.New()
	if err := c.Serialize(codec.NewEncodingWriter(sha)); err != nil {
		return [32]byte{}, err
	}
	var sum [32]byte
	copy(sum[:], sha.Sum(nil))
	return sum, nil
}
*/
