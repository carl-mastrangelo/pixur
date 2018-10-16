package schema

import (
	"fmt"
	"math"
	"path/filepath"
	"time"
)

func (p *Pic) SetCreatedTime(now time.Time) {
	p.CreatedTs = ToTspb(now)
}

func (p *Pic) SetModifiedTime(now time.Time) {
	p.ModifiedTs = ToTspb(now)
}

func (p *Pic) GetCreatedTime() time.Time {
	return ToTime(p.CreatedTs)
}

func (p *Pic) GetModifiedTime() time.Time {
	return ToTime(p.ModifiedTs)
}

func (p *Pic) Version() int64 {
	return ToTime(p.ModifiedTs).UnixNano()
}

func (p *Pic) NonHiddenIndexOrder() int64 {
	return p.GetCreatedTime().UnixNano()
}

func (p *Pic) IdCol() int64 {
	return p.PicId
}

func (p *Pic) IndexOrderCol() int64 {
	return p.IndexOrder()
}

func (p *Pic) IndexOrder() int64 {
	if p.isHidden() {
		return -1
	}
	return p.NonHiddenIndexOrder()
}

func (p *Pic) GetVarPicID() string {
	return Varint(p.PicId).Encode()
}

func (p *Pic) RelativeURL() string {
	return fmt.Sprintf("pix/%s.%s", p.GetVarPicID(), p.Mime.Ext())
}

func PicBaseDir(pixPath string, id int64) string {
	vid := Varint(id).Encode()
	path := []string{pixPath}

	for i := 0; i < len(vid)-1; i++ {
		path = append(path, string(vid[i:i+1]))
	}

	return filepath.Join(path...)
}

func (p *Pic) isHidden() bool {
	return p.HardDeleted()
}

func (p *Pic) SoftDeleted() bool {
	return p.GetDeletionStatus().GetMarkedDeletedTs() != nil && !p.HardDeleted()
}

func (p *Pic) HardDeleted() bool {
	return p.GetDeletionStatus().GetActualDeletedTs() != nil
}

const (
	Z_99        = 1.95996398612
	PicScoreMin = 0
	PicScoreMax = 1<<31 - 1
)

// TODO: test
func (p *Pic) WilsonScoreInterval(z float64) (lo float64, hi float64) {
	if p.HardDeleted() {
		return 0, 0
	}
	n := float64(p.VoteDown + p.VoteUp)
	if n == 0 {
		return 0.025, 0.975 // Just return something
	}

	phat := float64(p.VoteUp) / n
	scoremid := phat + z*z/(2*n)
	stddev := z * math.Sqrt(phat*(1-phat)/n+z*z/(4*n*n))
	norm := 1 + z*z/n
	return (scoremid - stddev) / norm, (scoremid + stddev) / norm
}

// Just use 30 bits, incase of accidental float overflow
// In a perfect world, wilson assures that the value is 0<=x<=1
func (p *Pic) LowerScoreBound() int32 {
	lo, _ := p.WilsonScoreInterval(Z_99)
	return int32(lo * (1 << 30))
}

func (p *Pic) UpperScoreBound() int32 {
	_, hi := p.WilsonScoreInterval(Z_99)
	return int32(hi * (1 << 30))
}
