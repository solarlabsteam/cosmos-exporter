# cosmos-exporter

![Latest release](https://img.shields.io/github/v/release/solarlabsteam/cosmos-exporter)
[![Actions Status](https://github.com/solarlabsteam/cosmos-exporter/workflows/test/badge.svg)](https://github.com/solarlabsteam/cosmos-exporter/actions)

cosmos-exporter is a Prometheus scraper that fetches the data from a full node of a Cosmos-based blockchain via gRPC.

## What can I use it for?

You can run a full node, run cosmos-exporter on the same host, set up Prometheus to scrape the data from it (see below for instructions), then set up Grafana to visualize the data coming from the exporter and probably add some alerting. Here are some examples of Grafana dashboards we created for ourselves:

![Validator dashboard](https://raw.githubusercontent.com/solarlabsteam/cosmos-exporter/master/images/dashboard_validator.png)
![Validators dashboard](https://raw.githubusercontent.com/solarlabsteam/cosmos-exporter/master/images/dashboard_validators.png)
![Wallet dashboard](https://raw.githubusercontent.com/solarlabsteam/cosmos-exporter/master/images/dashboard_wallet.png)

## How can I set it up?

First of all, you need to download the latest release from [the releases page](https://github.com/solarlabsteam/cosmos-exporter/releases/). After that, you should unzip it and you are ready to go:

```sh
wget <the link from the releases page>
tar xvfz cosmos-exporter-*.*-amd64.tar.gz
cd cosmos-exporter-*.*-amd64.tar.gz
./cosmos-exporter
```

That's not really interesting, what you probably want to do is to have it running in the background. For that, first of all, we have to copy the file to the system apps folder:

```sh
sudo cp ./cosmos-exporter /usr/bin
```

Then we need to create a systemd service for our app:

```sh
sudo nano /etc/systemd/system/cosmos-exporter.service
```

You can use this template (change the user to whatever user you want this to be executed from. It's advised to create a separate user for that instead of running it from root):

```
[Unit]
Description=Cosmos Exporter
After=network-online.target

[Service]
User=<username>
TimeoutStartSec=0
CPUWeight=95
IOWeight=95
ExecStart=cosmos-exporter
Restart=always
RestartSec=2
LimitNOFILE=800000
KillSignal=SIGTERM

[Install]
WantedBy=multi-user.target
```

Then we'll add this service to the autostart and run it:

```sh
sudo systemctl enable cosmos-exporter
sudo systemctl start cosmos-exporter
sudo systemctl status cosmos-exporter # validate it's running
```

If you need to, you can also see the logs of the process:

```sh
sudo journalctl -u cosmos-exporter -f
```

## How can I scrape data from it?

Here's the example of the Prometheus config you can use for scraping data:

```yaml
scrape-configs:
  # specific validator(s)
  - job_name:       'validator'
    scrape_interval: 15s
    metrics_path: /metrics/validator
    static_configs:
      - targets:
        - <list of validators you want to monitor>
    relabel_configs:
      - source_labels: [__address__]
        target_label: __param_address
      - source_labels: [__param_address]
        target_label: instance
      - target_label: __address__
        replacement: <node hostname or IP>:9300
  # specific wallet(s)
  - job_name:       'wallet'
    scrape_interval: 15s
    metrics_path: /metrics/wallet
    static_configs:
      - targets:
        - <list of wallets>
    relabel_configs:
      - source_labels: [__address__]
        target_label: __param_address
      - source_labels: [__param_address]
        target_label: instance
      - target_label: __address__
        replacement: <node hostname or IP>:9300

  # all validators
  - job_name:       'validators'
    scrape_interval: 15s
    metrics_path: /metrics/validators
    static_configs:
      - targets:
        replacement: <node hostname or IP>:9300
```

Then restart Prometheus and you're good to go!

## How does it work?

It queries the full node via gRPC and returns it in the format Prometheus can consume.

## How can I configure it?

You can pass the artuments to the executable file to configure it. Here is the parameters list:

- `--bech-prefix` - the prefix for addresses. Defaults to `persistence`
- `--denom` - the currency, for example, `uatom` for Cosmos. Defaults to `uxprt`
- `--listen-address` - the address with port the node would listen to. For example, you can use it to redefine port or to make the exporter accessible from the outside by listening on `127.0.0.1`. Defaults to `:9300` (so it's accessible from the outside on port 9300)
- `--node` - the gRPC node URL. Defaults to `localhost:9090`

## Which networks this is guaranteed to work?

In theory, it should work on a Cosmos-based blockchains that expose a gRPC endpoint (for example, Sentinel hub v0.5.0 doesn't expose it, so it won't work with it). In practice, this definitely works with the following blockchains:

- Persistence

## How can I contribute?

Bug reports and feature requests are always welcome! If you want to contribute, feel free to open issues or PRs.
