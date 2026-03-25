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
		a = util.RemoveDiacritics(a)
		b = util.RemoveDiacritics(b)
		return len(a) > len(b)
	}))

	return &Padder{maxFilenameLength: maxFilenameLength}
}

func (p *Padder) PaddingLength(filename string, margin int) int {
	videoFilenameWithoutDiacritics := util.RemoveDiacritics(filename)
	return p.maxFilenameLength - len(videoFilenameWithoutDiacritics) + margin
}
