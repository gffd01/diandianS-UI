#!/usr/bin/env python3
"""从公开热门域名列表生成 REALITY 候选库。

这里的结果只是候选。S-UI 面板仍会从用户自己的服务器重新测试三次，
不会直接相信 GitHub Actions 所在网络测得的结果。
"""

from __future__ import annotations

import argparse
import concurrent.futures
import csv
import datetime as dt
import io
import ipaddress
import json
import random
import socket
import ssl
import urllib.request
import zipfile
from pathlib import Path


DEFAULT_SOURCE = "https://tranco-list.eu/top-1m-incl-subdomains.csv.zip"
MAX_DOWNLOAD_BYTES = 40 * 1024 * 1024
SOURCE_LIMIT = 10_000
SAMPLE_SIZE = 1_200
OUTPUT_LIMIT = 200
MINIMUM_OUTPUT = 50

SEED_DOMAINS = [
    "www.apple.com",
    "addons.mozilla.org",
    "www.amazon.com",
    "aws.amazon.com",
    "www.cloudflare.com",
    "www.microsoft.com",
    "visualstudio.microsoft.com",
    "store-images.s-microsoft.com",
    "www.samsung.com",
    "www.nvidia.com",
    "www.amd.com",
    "www.intel.com",
    "www.sony.com",
    "dl.google.com",
    "r.bing.com",
    "th.bing.com",
    "cdn.userway.org",
    "tag.demandbase.com",
    "munchkin.marketo.net",
    "consent.trustarc.com",
    "rum.hlx.page",
]


def normalize_domain(raw: str) -> str | None:
    domain = raw.strip().lower().rstrip(".")
    if not 4 <= len(domain) <= 253 or "." not in domain:
        return None
    if any(char in domain for char in "_/:* "):
        return None
    labels = domain.split(".")
    for label in labels:
        if not label or len(label) > 63 or label.startswith("-") or label.endswith("-"):
            return None
        if not all(char.isascii() and (char.isalnum() or char == "-") for char in label):
            return None
    return domain


def download_candidates(source_url: str) -> list[str]:
    request = urllib.request.Request(
        source_url,
        headers={"User-Agent": "diandianS-UI-reality-feed/1"},
    )
    with urllib.request.urlopen(request, timeout=30) as response:
        payload = response.read(MAX_DOWNLOAD_BYTES + 1)
    if len(payload) > MAX_DOWNLOAD_BYTES:
        raise RuntimeError("public source is larger than the configured limit")

    with zipfile.ZipFile(io.BytesIO(payload)) as archive:
        csv_names = [name for name in archive.namelist() if name.lower().endswith(".csv")]
        if len(csv_names) != 1:
            raise RuntimeError("public source must contain exactly one CSV file")
        with archive.open(csv_names[0]) as raw_file:
            text_file = io.TextIOWrapper(raw_file, encoding="utf-8", newline="")
            reader = csv.reader(text_file)
            candidates: list[str] = []
            for row in reader:
                if len(row) < 2:
                    continue
                domain = normalize_domain(row[1])
                if domain:
                    candidates.append(domain)
                if len(candidates) >= SOURCE_LIMIT:
                    break
    return candidates


def public_addresses(domain: str) -> list[str]:
    addresses: list[str] = []
    for family, _, _, _, sockaddr in socket.getaddrinfo(domain, 443, type=socket.SOCK_STREAM):
        if family not in (socket.AF_INET, socket.AF_INET6):
            continue
        address = ipaddress.ip_address(sockaddr[0])
        if address.is_global:
            addresses.append(str(address))
    return list(dict.fromkeys(addresses))


def probe(domain: str) -> str | None:
    try:
        addresses = public_addresses(domain)
        if not addresses:
            return None
        context = ssl.create_default_context()
        context.minimum_version = ssl.TLSVersion.TLSv1_3
        context.maximum_version = ssl.TLSVersion.TLSv1_3
        context.set_alpn_protocols(["h2"])
        try:
            context.set_ecdh_curve("X25519")
        except (AttributeError, ValueError):
            return None

        for address in addresses[:2]:
            try:
                with socket.create_connection((address, 443), timeout=2.5) as tcp_sock:
                    with context.wrap_socket(tcp_sock, server_hostname=domain) as tls_sock:
                        if tls_sock.version() == "TLSv1.3" and tls_sock.selected_alpn_protocol() == "h2":
                            return domain
            except (OSError, ssl.SSLError):
                continue
    except (OSError, UnicodeError, ValueError):
        return None
    return None


def build_feed(source_url: str) -> dict[str, object]:
    public_domains = download_candidates(source_url)
    daily_seed = int(dt.datetime.now(dt.timezone.utc).strftime("%Y%m%d"))
    randomizer = random.Random(daily_seed)
    sample = randomizer.sample(public_domains, min(SAMPLE_SIZE, len(public_domains)))
    candidates = list(dict.fromkeys(SEED_DOMAINS + sample))

    passed: list[str] = []
    with concurrent.futures.ThreadPoolExecutor(max_workers=32) as executor:
        for result in executor.map(probe, candidates):
            if result:
                passed.append(result)

    domains = sorted(set(passed))[:OUTPUT_LIMIT]
    if len(domains) < MINIMUM_OUTPUT:
        raise RuntimeError(
            f"only {len(domains)} candidates passed; keeping the previous public feed"
        )
    return {
        "version": 1,
        "generated_at": dt.datetime.now(dt.timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z"),
        "source": "Tranco daily list with subdomains plus maintained seed domains",
        "source_url": source_url,
        "criteria": ["public IP", "valid certificate", "TLS 1.3", "HTTP/2", "X25519"],
        "domains": domains,
    }


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--source-url", default=DEFAULT_SOURCE)
    parser.add_argument("--output", required=True)
    args = parser.parse_args()

    feed = build_feed(args.source_url)
    output = Path(args.output)
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps(feed, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"published {len(feed['domains'])} REALITY candidates")


if __name__ == "__main__":
    main()
