package cosmosdirectory_test

import (
	"fmt"
	"main/pkg/cosmosdirectory"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetChain(t *testing.T) {
	tests := []struct {
		Chain string
	}{
		{Chain: "juno"},
		{Chain: "stargaze"},
		{Chain: "cheqd"},
	}

	for _, test := range tests {
		t.Run(test.Chain, func(t *testing.T) {
			chain, err := cosmosdirectory.GetChain(test.Chain)
			require.NoError(t, err)
			require.NotNil(t, chain)
			require.NotZero(t, chain.GetPriceUSD())
		})
	}
}

func TestGetChainByChainID(t *testing.T) {
	tests := []struct {
		ChainID string
	}{
		{ChainID: "juno-1"},
		{ChainID: "stargaze-1"},
		{ChainID: "cheqd-mainnet-1"},
		{ChainID: "kichain-2"},
		{ChainID: "Oraichain"},
		{ChainID: "chihuahua-1"},
	}

	for _, test := range tests {
		t.Run(test.ChainID, func(t *testing.T) {
			chain, err := cosmosdirectory.GetChainByChainID(test.ChainID)
			require.NoError(t, err)
			require.NotNil(t, chain)
			require.NotZero(t, chain.GetPriceUSD())

			fmt.Printf("price: %v\n", chain.GetPriceUSD())
		})
	}
}
