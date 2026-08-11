package reportworkflow

import (
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
)

// Family는 Plasma가 제품 코드로 고정해 둔 report graph family 이름이다.
type Family string

const (
	FamilyOneTake               Family = "one_take"
	FamilyPlanned               Family = "planned"
	FamilyLongFormSerial        Family = "long_form_serial"
	FamilyLongFormSectionFanout Family = "long_form_section_fanout"
)

// SelectFamily는 pending payload의 mode/strategy에서 고정 graph family를 고른다.
func SelectFamily(mode string, strategy string) (Family, error) {
	normalizedMode, err := reportexecution.NormalizeMode(mode)
	if err != nil {
		return "", err
	}
	switch normalizedMode {
	case reportexecution.ModeOneTake:
		return FamilyOneTake, nil
	case reportexecution.ModePlanned:
		return FamilyPlanned, nil
	case reportexecution.ModeLongForm:
		if strings.TrimSpace(strategy) == "section_fanout" {
			return FamilyLongFormSectionFanout, nil
		}
		return FamilyLongFormSerial, nil
	default:
		return "", fmt.Errorf("%w: unsupported report workflow mode", producterror.ErrInvalidInput)
	}
}
