package bls

type Pairs = []Pair

type Pair struct {
	g1 PointG1
	g2 PointG2
}

type BLSPairingEngine struct {
	G1    *G1
	G2    *G2
	fp12  *fp12
	fp2   *fp2
	t2    [10]*fe2
	t12   [9]fe12
	pairs Pairs
}

func NewBLSPairingEngine() *BLSPairingEngine {
	fp2 := newFp2()
	fp6 := newFp6(fp2)
	fp12 := newFp12(fp6)
	g1 := NewG1()
	g2 := NewG2(fp2)
	t2 := [10]*fe2{}
	for i := 0; i < 10; i++ {
		t2[i] = &fe2{}
	}
	t12 := [9]fe12{}
	return &BLSPairingEngine{
		fp2:  fp2,
		fp12: fp12,
		t2:   t2,
		t12:  t12,
		G1:   g1,
		G2:   g2,
	}
}

// Adaptation of Formula 3 in https://eprint.iacr.org/2010/526.pdf
func (e *BLSPairingEngine) doublingStep(coeff *[3]fe2, r *PointG2) {
	fp2 := e.fp2
	t := e.t2
	fp2.mul(t[0], &r[0], &r[1])
	fp2.mulByFq(t[0], t[0], twoInv)
	fp2.square(t[1], &r[1])
	fp2.square(t[2], &r[2])
	fp2.double(t[7], t[2])
	fp2.add(t[7], t[7], t[2])
	fp2.mulByB(t[3], t[7])
	fp2.double(t[4], t[3])
	fp2.add(t[4], t[4], t[3])
	fp2.add(t[5], t[1], t[4])
	fp2.mulByFq(t[5], t[5], twoInv)
	fp2.add(t[6], &r[1], &r[2])
	fp2.square(t[6], t[6])
	fp2.add(t[7], t[2], t[1])
	fp2.sub(t[6], t[6], t[7])
	fp2.sub(&coeff[0], t[3], t[1])
	fp2.square(t[7], &r[0])
	fp2.sub(t[4], t[1], t[4])
	fp2.mul(&r[0], t[4], t[0])
	fp2.square(t[2], t[3])
	fp2.double(t[3], t[2])
	fp2.add(t[3], t[3], t[2])
	fp2.square(t[5], t[5])
	fp2.sub(&r[1], t[5], t[3])
	fp2.mul(&r[2], t[1], t[6])
	fp2.double(t[0], t[7])
	fp2.add(&coeff[1], t[0], t[7])
	fp2.neg(&coeff[2], t[6])
}

// Algorithm 12 in https://eprint.iacr.org/2010/526.pdf
func (e *BLSPairingEngine) additionStep(coeff *[3]fe2, r, q *PointG2) {
	fp2 := e.fp2
	t := e.t2
	fp2.mul(t[0], &q[1], &r[2])
	fp2.neg(t[0], t[0])
	fp2.add(t[0], t[0], &r[1])
	fp2.mul(t[1], &q[0], &r[2])
	fp2.neg(t[1], t[1])
	fp2.add(t[1], t[1], &r[0])
	fp2.square(t[2], t[0])
	fp2.square(t[3], t[1])
	fp2.mul(t[4], t[1], t[3])
	fp2.mul(t[2], &r[2], t[2])
	fp2.mul(t[3], &r[0], t[3])
	fp2.double(t[5], t[3])
	fp2.sub(t[5], t[4], t[5])
	fp2.add(t[5], t[5], t[2])
	fp2.mul(&r[0], t[1], t[5])
	fp2.sub(t[2], t[3], t[5])
	fp2.mul(t[2], t[2], t[0])
	fp2.mul(t[3], &r[1], t[4])
	fp2.sub(&r[1], t[2], t[3])
	fp2.mul(&r[2], &r[2], t[4])
	fp2.mul(t[2], t[1], &q[1])
	fp2.mul(t[3], t[0], &q[0])
	fp2.sub(&coeff[0], t[3], t[2])
	fp2.neg(&coeff[1], t[0])
	fp2.copy(&coeff[2], t[1])
}

// Algorithm 5 in  https://eprint.iacr.org/2019/077.pdf
func (e *BLSPairingEngine) preCompute(ellCoeffs *[70][3]fe2, twistPoint *PointG2) {
	if e.G2.IsZero(twistPoint) {
		return
	}
	r := &PointG2{}
	e.G2.Copy(r, twistPoint)
	j := 0
	for i := int(x.BitLen() - 2); i >= 0; i-- {
		e.doublingStep(&ellCoeffs[j], r)
		if x.Bit(i) != 0 {
			j++
			ellCoeffs[j] = fe6{}
			e.additionStep(&ellCoeffs[j], r, twistPoint)
		}
		j++
	}
}

func (e *BLSPairingEngine) millerLoop(f *fe12, pairs Pairs) {
	for i := 0; i <= len(pairs)-1; i++ {
		e.G1.Affine(&pairs[i].g1)
		e.G2.Affine(&pairs[i].g2)
	}
	ellCoeffs := make([][70][3]fe2, len(pairs))
	for i := 0; i < len(pairs); i++ {
		e.preCompute(&ellCoeffs[i], &pairs[i].g2)
	}
	fp12 := e.fp12
	fp2 := e.fp2
	t := e.t2
	fp12.copy(f, fp12.one())
	for i := 0; i <= len(pairs)-1; i++ {
		fp2.mulByFq(t[0], &ellCoeffs[i][0][2], &pairs[i].g1[1])
		fp2.mulByFq(t[1], &ellCoeffs[i][0][1], &pairs[i].g1[0])
		fp12.mulBy014Assign(f, &ellCoeffs[i][0][0], t[1], t[0])
	}
	for i := 0; i <= len(pairs)-1; i++ {
		fp2.mulByFq(t[0], &ellCoeffs[i][1][2], &pairs[i].g1[1])
		fp2.mulByFq(t[1], &ellCoeffs[i][1][1], &pairs[i].g1[0])
		fp12.mulBy014Assign(f, &ellCoeffs[i][1][0], t[1], t[0])
	}
	j := 2
	for i := int(x.BitLen() - 3); i >= 0; i-- {
		fp12.square(f, f)
		for i := 0; i <= len(pairs)-1; i++ {
			fp2.mulByFq(t[0], &ellCoeffs[i][j][2], &pairs[i].g1[1])
			fp2.mulByFq(t[1], &ellCoeffs[i][j][1], &pairs[i].g1[0])
			fp12.mulBy014Assign(f, &ellCoeffs[i][j][0], t[1], t[0])
		}
		if x.Bit(i) != 0 {
			j++
			for i := 0; i <= len(pairs)-1; i++ {
				fp2.mulByFq(t[0], &ellCoeffs[i][j][2], &pairs[i].g1[1])
				fp2.mulByFq(t[1], &ellCoeffs[i][j][1], &pairs[i].g1[0])
				fp12.mulBy014Assign(f, &ellCoeffs[i][j][0], t[1], t[0])
			}
		}
		j++
	}
	fp12.conjugate(f, f)
}

func (e *BLSPairingEngine) exp(c, a *fe12) {
	fp12 := e.fp12
	fp12.cyclotomicExp(c, a, x)
	fp12.conjugate(c, c)
}

func (e *BLSPairingEngine) finalExp(f *fe12) {
	fp12 := e.fp12
	t := e.t12
	// easy part
	fp12.frobeniusMap(&t[0], f, 6)
	fp12.inverse(&t[1], f)
	fp12.mul(&t[2], &t[0], &t[1])
	fp12.copy(&t[1], &t[2])
	fp12.frobeniusMapAssign(&t[2], 2)
	fp12.mulAssign(&t[2], &t[1])
	fp12.cyclotomicSquare(&t[1], &t[2])
	fp12.conjugate(&t[1], &t[1])
	// hard but tricky part
	e.exp(&t[3], &t[2])
	fp12.cyclotomicSquare(&t[4], &t[3])
	fp12.mul(&t[5], &t[1], &t[3])
	e.exp(&t[1], &t[5])
	e.exp(&t[0], &t[1])
	e.exp(&t[6], &t[0])
	fp12.mulAssign(&t[6], &t[4])
	e.exp(&t[4], &t[6])
	fp12.conjugate(&t[5], &t[5])
	fp12.mulAssign(&t[4], &t[5])
	fp12.mulAssign(&t[4], &t[2])
	fp12.conjugate(&t[5], &t[2])
	fp12.mulAssign(&t[1], &t[2])
	fp12.frobeniusMapAssign(&t[1], 3)
	fp12.mulAssign(&t[6], &t[5])
	fp12.frobeniusMapAssign(&t[6], 1)
	fp12.mulAssign(&t[3], &t[0])
	fp12.frobeniusMapAssign(&t[3], 2)
	fp12.mulAssign(&t[3], &t[1])
	fp12.mulAssign(&t[3], &t[6])
	fp12.mul(f, &t[3], &t[4])
}

func (e *BLSPairingEngine) pair(pairs Pairs) *fe12 {
	f := e.fp12.one()
	if len(pairs) == 0 {
		return f
	}
	var newPairs Pairs
	for i := 0; i < len(pairs); i++ {
		if !e.G1.IsZero(&pairs[i].g1) && !e.G2.IsZero(&pairs[i].g2) {
			newPairs = append(newPairs, pairs[i])
		}
	}
	if len(newPairs) == 0 {
		return f
	}
	e.millerLoop(f, newPairs)
	e.finalExp(f)
	return f
}

func (e *BLSPairingEngine) Check(pairs Pairs) bool {
	return e.fp12.equal(e.fp12.one(), e.pair(pairs))
}
