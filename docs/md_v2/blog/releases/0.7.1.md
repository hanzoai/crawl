# 🛠️ Crawl v0.7.1: Minor Cleanup Update

*July 17, 2025 • 2 min read*

---

A small maintenance release that removes unused code and improves documentation.

## 🎯 What's Changed

- **Removed unused StealthConfig** from `crawl/browser_manager.py`
- **Updated documentation** with better examples and parameter explanations
- **Fixed virtual scroll configuration** examples in docs

## 🧹 Code Cleanup

Removed unused `StealthConfig` import and configuration that wasn't being used anywhere in the codebase. The project uses its own custom stealth implementation through JavaScript injection instead.

```python
# Removed unused code:
from playwright_stealth import StealthConfig
stealth_config = StealthConfig(...)  # This was never used
```

## 📖 Documentation Updates

- Fixed adaptive crawling parameter examples
- Updated session management documentation
- Corrected virtual scroll configuration examples

## 🚀 Installation

```bash
pip install crawl==0.7.1
```

No breaking changes - upgrade directly from v0.7.0.

---

Questions? Issues? 
- GitHub: [github.com/hanzoai/crawl](https://github.com/hanzoai/crawl)
- Discord: [discord.gg/crawl](https://discord.gg/jP8KfhDhyN)