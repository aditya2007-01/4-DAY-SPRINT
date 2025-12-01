# BHIV Blockchain Debug & Block Inspector Tool

A production-ready Go-based CLI tool for inspecting blockchain blocks, verifying chain integrity, comparing nodes, debugging P2P sync failures, and performing comprehensive error analysis with classification.

## 🎯 Project Overview

This tool provides enterprise-grade blockchain inspection and debugging capabilities for blockchain node operators and developers. It enables quick diagnostics, integrity verification, performance analysis, multi-node comparison, and detailed error scanning with automated reporting.

**Built as part of a 7-day learning and development task focused on creating production-usable blockchain debugging utilities.**

---

## ✨ Features

### Core Capabilities
- **📦 Block Viewing**: Inspect individual blocks with detailed metadata
- **📊 Chain Statistics**: Comprehensive blockchain metrics and analytics
- **✅ Complete Chain Verification**: Validate block hashes, prevHash linkage, timestamps, and height
- **🔄 Node Comparison**: Compare two blockchain databases to detect divergence and sync issues
- **🔍 Error Scanner**: Advanced error detection with detailed classification
- **💾 Data Loading**: Load sample blockchain data for testing and development
- **📋 JSON Output Mode**: Machine-readable output for automation and CI/CD

### Advanced Error Detection
- **Corrupted JSON**: Detects invalid or malformed block data
- **Bad Hash**: Identifies hash mismatches and data tampering
- **Timestamp Anomalies**: Flags future timestamps, past timestamps, and non-increasing sequences
- **Duplicate Hashes**: Detects blocks with identical hashes
- **Empty Blocks**: Identifies blocks with no transaction data
- **Missing Blocks**: Finds gaps in the blockchain
- **Chain Linkage**: Validates prevHash connections
- **Out of Order**: Detects blocks in wrong sequence

---

### Installation

