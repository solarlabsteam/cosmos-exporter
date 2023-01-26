package cosmosdirectory

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

const CosmosDirectoryURL = "https://chains.cosmos.directory"

func query() (*CosmosDirectory, error) {
	httpClient := &http.Client{Timeout: 2 * time.Second}
	r, err := httpClient.Get(CosmosDirectoryURL)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	resp := &CosmosDirectory{}

	err = json.NewDecoder(r.Body).Decode(resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func GetChain(name string) (*Chain, error) {
	resp, err := query()
	if err != nil {
		return nil, err
	}

	for _, chain := range resp.Chains {
		if chain.Name == name || chain.ChainName == name {
			return &chain, nil
		}
	}
	return nil, errors.New("chain not found")
}

func GetChainByChainID(chainID string) (*Chain, error) {
	resp, err := query()
	if err != nil {
		return nil, err
	}

	for _, chain := range resp.Chains {
		if chain.ChainID == chainID {
			return &chain, nil
		}
	}
	return nil, errors.New("chain not found")
}
