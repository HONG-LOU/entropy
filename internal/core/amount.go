package core

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	UnitsPerENT        uint64 = 100_000_000
	MaxSupply          uint64 = 2_000_000 * UnitsPerENT
	TargetBlockSeconds        = 10
	EmissionYears             = 10
	EmissionBlocks     uint64 = EmissionYears * 365 * 24 * 60 * 60 / TargetBlockSeconds
	BaseSubsidy        uint64 = MaxSupply / EmissionBlocks
	BonusSubsidyBlocks uint64 = MaxSupply % EmissionBlocks
)

func Subsidy(height uint64) uint64 {
	if height == 0 || height > EmissionBlocks {
		return 0
	}
	reward := BaseSubsidy
	if height <= BonusSubsidyBlocks {
		reward++
	}
	return reward
}

func MintedThrough(height uint64) uint64 {
	if height > EmissionBlocks {
		height = EmissionBlocks
	}
	bonusBlocks := height
	if bonusBlocks > BonusSubsidyBlocks {
		bonusBlocks = BonusSubsidyBlocks
	}
	return height*BaseSubsidy + bonusBlocks
}

func ParseAmount(value string) (uint64, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "-") || strings.HasPrefix(value, "+") {
		return 0, fmt.Errorf("invalid amount %q", value)
	}

	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, fmt.Errorf("invalid amount %q", value)
	}
	whole, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil || whole > ^uint64(0)/UnitsPerENT {
		return 0, fmt.Errorf("invalid amount %q", value)
	}

	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
		if fraction == "" || len(fraction) > 8 {
			return 0, fmt.Errorf("amount supports at most 8 decimal places")
		}
	}
	for len(fraction) < 8 {
		fraction += "0"
	}
	frac := uint64(0)
	if fraction != "" {
		frac, err = strconv.ParseUint(fraction, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid amount %q", value)
		}
	}
	units := whole*UnitsPerENT + frac
	if units < whole*UnitsPerENT {
		return 0, fmt.Errorf("amount overflows")
	}
	return units, nil
}

func FormatAmount(units uint64) string {
	whole := units / UnitsPerENT
	fraction := units % UnitsPerENT
	if fraction == 0 {
		return strconv.FormatUint(whole, 10)
	}
	return fmt.Sprintf("%d.%08d", whole, fraction)
}

func safeAdd(a, b uint64) (uint64, error) {
	if ^uint64(0)-a < b {
		return 0, fmt.Errorf("amount overflows")
	}
	return a + b, nil
}
