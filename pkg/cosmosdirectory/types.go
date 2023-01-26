package cosmosdirectory

type CosmosDirectory struct {
	Repository Repository `json:"repository"`
	Chains     []Chain    `json:"chains"`
}

type Repository struct {
	Url       string  `json:"url"`
	Branch    string  `json:"branch"`
	Commit    string  `json:"commit"`
	Timestamp float64 `json:"timestamp"`
}

type Chain struct {
	Decimals      float64     `json:"decimals"`
	BestApis      BestApis    `json:"best_apis"`
	Versions      Versions    `json:"versions"`
	Prices        Prices      `json:"prices"`
	Name          string      `json:"name"`
	ChainName     string      `json:"chain_name"`
	Denom         string      `json:"denom"`
	Bech32_prefix string      `json:"bech32_prefix"`
	Image         string      `json:"image"`
	Height        float64     `json:"height"`
	ProxyStatus   ProxyStatus `json:"proxy_status"`
	Assets        []Assets    `json:"assets"`
	Network_type  string      `json:"network_type"`
	Pretty_name   string      `json:"pretty_name"`
	ChainID       string      `json:"chain_id"`
	Coingecko_id  string      `json:"coingecko_id"`
	Explorers     []Explorers `json:"explorers"`
	Display       string      `json:"display"`
	Params        Params      `json:"params"`
	Path          string      `json:"path"`
	Status        string      `json:"status"`
	Symbol        string      `json:"symbol"`
}

func (chain Chain) GetPriceUSD() float64 {
	prices, exist := chain.Prices.Coingecko[chain.Display].(map[string]interface{})
	if !exist {
		return float64(0)
	}
	price, exist := prices["usd"].(float64)
	if !exist {
		return float64(0)
	}
	return price
}

type CoingeckoPrices map[string]interface{} // map[string]interface{}

type Prices struct {
	// Coingecko map[string]CoingeckoPrices `json:"coingecko"`
	Coingecko map[string]interface{} `json:"coingecko"`
	// Coingecko Coingecko `json:"coingecko"`
}

type Rpc struct {
	Address  string `json:"address"`
	Provider string `json:"provider"`
}

type Versions struct {
	ApplicationVersion string `json:"application_version"`
	CosmosSDKVersion   string `json:"cosmos_sdk_version"`
	TendermintVersion  string `json:"tendermint_version"`
}

type Denom_units struct {
	Denom    string  `json:"denom"`
	Exponent float64 `json:"exponent"`
}

type Staking struct {
	MaxValidators     float64 `json:"max_validators"`
	MaxEntries        float64 `json:"max_entries"`
	HistoricalEntries float64 `json:"historical_entries"`
	Bond_denom        string  `json:"bond_denom"`
	UnbondingTime     string  `json:"unbonding_time"`
}

type Distribution struct {
	CommunityTax          string `json:"community_tax"`
	Base_proposer_reward  string `json:"base_proposer_reward"`
	Bonus_proposer_reward string `json:"bonus_proposer_reward"`
	Withdraw_addr_enabled bool   `json:"withdraw_addr_enabled"`
}

type BestApis struct {
	Rest []Rest `json:"rest"`
	Rpc  []Rpc  `json:"rpc"`
}

type Explorers struct {
	Tx_page string `json:"tx_page"`
	Kind    string `json:"kind"`
	Url     string `json:"url"`
}

type Coingecko struct {
	Usd float64 `json:"usd"`
}

type Acre struct {
	Usd float64 `json:"usd"`
}

type ProxyStatus struct {
	Rest bool `json:"rest"`
	Rpc  bool `json:"rpc"`
}

type Assets struct {
	Denom        string        `json:"denom"`
	Coingecko_id string        `json:"coingecko_id"`
	Base         Base          `json:"base"`
	Denom_units  []Denom_units `json:"denom_units"`
	Image        string        `json:"image"`
	Symbol       string        `json:"symbol"`
	Description  string        `json:"description"`
	Decimals     float64       `json:"decimals"`
	Display      Display       `json:"display"`
	Logo_URIs    Logo_URIs     `json:"logo_URIs"`
	Prices       Prices        `json:"prices"`
	Name         string        `json:"name"`
}

type Display struct {
	Denom    string  `json:"denom"`
	Exponent float64 `json:"exponent"`
}

type Params struct {
	Authz                  bool         `json:"authz"`
	Actual_blocks_per_year float64      `json:"actual_blocks_per_year"`
	UnbondingTime          float64      `json:"unbonding_time"`
	Bonded_tokens          string       `json:"bonded_tokens"`
	Total_supply           string       `json:"total_supply"`
	Actual_block_time      float64      `json:"actual_block_time"`
	Current_block_height   string       `json:"current_block_height"`
	MaxValidators          float64      `json:"max_validators"`
	Staking                Staking      `json:"staking"`
	Slashing               Slashing     `json:"slashing"`
	Bonded_ratio           float64      `json:"bonded_ratio"`
	CommunityTax           float64      `json:"community_tax"`
	Distribution           Distribution `json:"distribution"`
}

type Slashing struct {
	Downtime_jail_duration     string `json:"downtime_jail_duration"`
	Slash_fraction_double_sign string `json:"slash_fraction_double_sign"`
	Slash_fraction_downtime    string `json:"slash_fraction_downtime"`
	Signed_blocks_window       string `json:"signed_blocks_window"`
	Min_signed_per_window      string `json:"min_signed_per_window"`
}

type Base struct {
	Denom    string  `json:"denom"`
	Exponent float64 `json:"exponent"`
}

type Logo_URIs struct {
	Png string `json:"png"`
	Svg string `json:"svg"`
}

type Rest struct {
	Provider string `json:"provider"`
	Address  string `json:"address"`
}
