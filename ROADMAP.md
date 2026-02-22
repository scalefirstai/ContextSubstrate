# ContextSubstrate Roadmap

This document outlines the development roadmap for **ContextSubstrate** (`ctx`), an execution substrate for AI agents that makes their work reproducible, debuggable, and contestable. The project uses content-addressed context packs (SHA-256 hashes), replay, drift detection, provenance verification, and forking to bring transparency and auditability to agent-driven workflows.

Items are organized into milestone phases. Checked items have shipped; unchecked items represent planned work. Timelines are approximate and may shift based on community feedback and priorities.

---

## v0.1.0 -- Foundation (Shipped)

- [x] Content-addressed blob store (SHA-256)
- [x] Context pack creation, inspection, and listing
- [x] Structured diff / decision drift detection
- [x] Step-by-step replay with fidelity reporting
- [x] Artifact provenance verification
- [x] Pack forking for mutable drafts
- [x] Shell completion (bash, zsh, fish, powershell)
- [x] Git commit indexing and context graphs
- [x] File-level delta computation
- [x] Token-optimized context pack generation
- [x] Token savings metrics and benchmarking
- [x] Cross-platform releases (Linux, macOS, Windows; amd64, arm64)

---

## v0.2.0 -- Security and Discovery

Focus: strengthen the trust model and make packs easier to find and query.

- [ ] Pack signing and attestation -- cryptographic signatures for tamper-evident packs
- [ ] Pack search and querying -- full-text and metadata search across local pack stores
- [ ] MCP server for agent discoverability -- expose context packs through the Model Context Protocol so agents can locate and consume them automatically

---

## v0.3.0 -- CI/CD and Developer Tooling

Focus: meet developers where they already work.

- [ ] GitHub Actions integration -- generate and publish context packs as part of CI pipelines
- [ ] VS Code extension -- browse, inspect, and diff context packs directly in the editor
- [ ] JetBrains extension -- equivalent pack browsing and inspection for IntelliJ-based IDEs

---

## v0.4.0 -- Agent Framework Integrations

Focus: first-class support for the most widely adopted agent orchestration libraries.

- [ ] LangChain integration -- callbacks and middleware for automatic context pack capture
- [ ] CrewAI integration -- task-level pack generation for multi-agent crews
- [ ] AutoGen integration -- conversation-turn capture and replay support

---

## v0.5.0 -- Collaboration and Remote Storage

Focus: enable teams to share, store, and govern context packs at scale.

- [ ] Remote pack storage (S3, GCS) -- push and pull packs to cloud object stores
- [ ] Team collaboration with shared pack registries -- access control, namespaces, and shared discovery across organizations
- [ ] Web UI for browsing context packs -- a standalone web interface for exploring pack contents, diffs, and provenance chains

---

## Future Considerations

The following ideas are under evaluation and may be incorporated into a future milestone:

- Fine-grained RBAC for pack registries
- OpenTelemetry trace correlation
- Plugin system for custom pack transformers
- Streaming replay for long-running agent sessions
- Pack garbage collection and retention policies

---

## Contributing

We welcome contributions of all kinds -- code, documentation, bug reports, and feature requests. Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on how to get started.

Have an idea or want to discuss a roadmap item? Open a GitHub Issue and let us know. Community input directly shapes the priorities above.
