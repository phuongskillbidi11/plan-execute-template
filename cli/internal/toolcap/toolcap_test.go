package toolcap

import "testing"

func TestRiskRankOrdering(t *testing.T) {
	if !(RiskRank(RiskRead) < RiskRank(RiskWrite)) {
		t.Fatal("expected READ < WRITE")
	}
	if !(RiskRank(RiskWrite) < RiskRank(RiskDestructive)) {
		t.Fatal("expected WRITE < DESTRUCTIVE")
	}
	if !(RiskRank(RiskDestructive) < RiskRank(RiskHighRisk)) {
		t.Fatal("expected DESTRUCTIVE < HIGH_RISK")
	}
}

func TestRiskRankUnknownRanksAboveHighRisk(t *testing.T) {
	if RiskRank(Risk("bogus")) <= RiskRank(RiskHighRisk) {
		t.Fatal("expected an unknown risk to rank above HIGH_RISK (fail restrictive)")
	}
}
