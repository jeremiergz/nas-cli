package str

import (
	"github.com/samber/lo"

	"github.com/jeremiergz/nas-cli/internal/util"
)

type Padder struct {
	maxFilenameLength int
}

func NewPadder(filenames []string) *Padder {
	maxFilenameLength := len(lo.MaxBy(filenames, func(a, b string) bool {
		a, _ = util.RemoveDiacritics(a)
		b, _ = util.RemoveDiacritics(b)
		return len(a) > len(b)
	}))

	return &Padder{maxFilenameLength: maxFilenameLength}
}

func (p *Padder) PaddingLength(filename string, margin int) int {
	videoFilenameWithoutDiacritics, _ := util.RemoveDiacritics(filename)
	return p.maxFilenameLength - len(videoFilenameWithoutDiacritics) + margin
}
