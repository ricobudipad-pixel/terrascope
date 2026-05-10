# 🔭 TerraScope

[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go 1.22+](https://img.shields.io/badge/go-1.22+-00ADD8.svg)](https://golang.org)
[![MiMo v2.5](https://img.shields.io/badge/MiMo-v2.5-orange.svg)](https://mimo.xiaomi.com)

> **Infrastructure config drift detector with AI-powered security analysis — built on Xiaomi MiMo v2.5.**

Upload Terraform, Kubernetes, or Docker Compose configurations. TerraScope detects drift from baselines, identifies security misconfigurations, flags cost optimization opportunities, and provides actionable remediation — all powered by MiMo's reasoning engine.

**[Live Demo](https://terrascope-demo.trycloudflare.com)** · **[Report Bug](../../issues)** · **[Request Feature](../../issues)**

---

## 📸 Screenshots

| Dashboard | Scan Results | Baseline Management |
|-----------|-------------|---------------------|
| ![Dashboard](docs/screenshots/dashboard.png) | ![Scan](docs/screenshots/scan.png) | ![Baselines](docs/screenshots/baselines.png) |

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        TerraScope                                │
│                  (Go net/http + HTML Templates)                   │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐       │
│  │   Upload     │    │   Parsers    │    │   MiMo API   │       │
│  │   Config     │───►│  HCL / YAML  │───►│   Analysis   │       │
│  └──────────────┘    └──────┬───────┘    └──────┬───────┘       │
│                              │                    │               │
│                              ▼                    ▼               │
│                     ┌──────────────┐    ┌──────────────┐        │
│                     │  Diff Engine │    │  Security &  │        │
│                     │  (Baseline   │    │  Cost Rules  │        │
│                     │  Comparison) │    │  (AI + Heur) │        │
│                     └──────┬───────┘    └──────┬───────┘        │
│                            │                    │                 │
│                            └────────┬───────────┘                │
│                                     ▼                            │
│                            ┌──────────────┐                     │
│                            │   SQLite     │                     │
│                            │  (History)   │                     │
│                            └──────────────┘                     │
└─────────────────────────────────────────────────────────────────┘
```

### Analysis Pipeline

1. **Parse** — Detect format (Terraform HCL, Kubernetes YAML) and extract resources
2. **Baseline Compare** — Diff against known-good configs (missing, modified, added)
3. **AI Analysis** — MiMo scans for security misconfigs, cost issues, compliance gaps
4. **Score** — Assign severity (Critical/High/Medium/Low) and generate remediation

---

## ✨ Features

- 🏗️ **Multi-Format Parsing** — Terraform HCL, Kubernetes YAML, Docker Compose
- 🔍 **Baseline Comparison** — Define known-good configs, detect drift automatically
- 🛡️ **AI Security Analysis** — MiMo-powered detection of OWASP infra vulnerabilities
- 💰 **Cost Optimization** — Flag over-provisioned instances, unused resources
- 📊 **Severity Scoring** — Critical / High / Medium / Low with remediation steps
- 🔗 **REST API** — CI/CD integration for automated infra audits
- 📈 **Scan History** — Track drift trends over time
- 🐳 **Docker Ready** — Single binary, minimal container

---

## 🔥 Token Economics

| Scan Type | Config Size | Tokens/Scan | Scans/Day | Daily Tokens |
|-----------|-------------|-------------|-----------|--------------|
| Simple (10 resources) | ~500 LOC | 3K–5K | 50–100 | ~400K |
| Medium (50 resources) | ~2K LOC | 8K–15K | 20–40 | ~500K |
| Complex (200+ resources) | ~10K LOC | 20K–40K | 5–10 | ~300K |
| Enterprise (daily audit) | varies | 15K–30K | 100+ | ~3M |
| CI/CD per PR (diff only) | ~200 LOC diff | 2K–5K | 200+ | ~800K |

---

## 🚀 Quick Start

### Prerequisites

- Go 1.22+
- MiMo API key ([get one here](https://mimo.xiaomi.com))

### Build & Run

```bash
git clone https://github.com/ricobudipad-pixel/terrascope.git
cd terrascope
cp .env.example .env
# Edit .env — set MIMO_API_KEY

go build -o terrascope ./cmd/terrascope/
./terrascope
```

Open [http://localhost:8080](http://localhost:8080)

### Docker

```bash
docker build -t terrascope .
docker run -p 8080:8080 -e MIMO_API_KEY=your-key terrascope
```

### Run Tests

```bash
go test ./...
```

---

## 📖 API Documentation

### Create Scan

```bash
curl -X POST http://localhost:8080/api/scan \
  -H "Content-Type: application/json" \
  -d '{
    "name": "production-check",
    "config_type": "terraform",
    "content": "resource \"aws_instance\" \"web\" { ... }"
  }'
```

### List Scans

```bash
curl http://localhost:8080/api/scans
```

### Create Baseline

```bash
curl -X POST http://localhost:8080/api/baselines/create \
  -H "Content-Type: application/json" \
  -d '{"name": "prod-baseline", "config_type": "terraform", "content": "..."}'
```

---

## ⚙️ Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `MIMO_API_KEY` | MiMo API key | — (optional, enables AI analysis) |
| `MIMO_BASE_URL` | MiMo API endpoint | `https://api.mimo.xiaomi.com/v1` |
| `MIMO_MODEL` | Model identifier | `MiMo-v2.5` |
| `PORT` | Server port | `8080` |
| `DATABASE_URL` | SQLite database path | `terrascope.db` |

---

## 🤝 Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## 📄 License

MIT — see [LICENSE](LICENSE).
