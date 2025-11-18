<!--
Sync Impact Report:
- Version change: 1.1.0 → 1.2.0 (added MCP Resources, Configuration Excellence, Docker-First Deployment principles; enhanced performance targets and security requirements)
- Modified principles: Enhanced V. Performance & Caching Excellence with specific metrics; added VII. MCP Resources Implementation, VIII. Configuration Excellence, IX. Docker-First Deployment
- Added sections: Three new principles with comprehensive rationales, enhanced Security Requirements and Development Workflow
- Removed sections: None
- Templates requiring updates: ✅ All template references validated
- Follow-up TODOs: None - all PRD-informed enhancements complete
-->

# OneNote MCP Server Constitution

## Core Principles

### I. Security-First Architecture
Security is non-negotiable. All authentication must use OAuth 2.0 PKCE flow. Authorization system MUST prevent AI agents from accessing unauthorized notebooks. NO client secrets stored. Token refresh MUST be automatic and transparent. Input validation MUST prevent injection attacks and illegal OneNote characters.

*Rationale: As an integration with Microsoft Graph API handling sensitive user data, security failures could compromise entire OneNote accounts. The OAuth 2.0 PKCE flow eliminates client secret storage risks, while notebook-scoped authorization provides AI agent safety guardrails.*

### II. Multi-Protocol MCP Implementation
MUST support both stdio and HTTP modes for MCP protocol communication. HTTP mode MUST include Server-Sent Events (SSE) for progress notifications. Protocol switching MUST be seamless without code duplication. All MCP tools and resources MUST work identically across both transport modes.

*Rationale: MCP clients have varying integration needs - CLI tools prefer stdio while web applications require HTTP. Supporting both protocols maximizes compatibility while SSE streaming enables real-time progress feedback for long-running operations.*

### III. Modular Domain Architecture
Code MUST be organized by domain boundaries: auth/, notebooks/, pages/, sections/, graph/. Each domain MUST be independently testable with clear interfaces. HTTP client composition MUST be shared across domains. NO circular dependencies between domain modules.

*Rationale: Domain separation enables focused testing, easier maintenance, and clear responsibility boundaries. Shared HTTP client composition prevents code duplication while maintaining domain isolation.*

### IV. Test-Driven Development (NON-NEGOTIABLE)
Every feature MUST have comprehensive unit tests with mocks for Graph API calls. Test coverage MUST exceed 80% for all business logic. Integration tests MUST validate authorization enforcement. Mock-based testing prevents external API dependencies in CI/CD.

*Rationale: Microsoft Graph API integration complexity requires rigorous testing to prevent data corruption. Mock-based testing ensures reliable CI/CD while comprehensive coverage catches authorization bypass vulnerabilities.*

### V. Performance & Caching Excellence
Multi-layer caching MUST be implemented: page metadata (5min), search results, notebook lookups. Cache invalidation MUST be automatic on content changes. Progress notifications MUST distinguish cached vs API operations. Thread-safe operations MUST support concurrent access. Response times MUST be under 2 seconds for cached operations and under 10 seconds for fresh API calls. Memory usage MUST remain under 100MB baseline footprint.

*Rationale: OneNote operations can be slow due to API latency. Intelligent caching dramatically improves user experience while progress notifications provide transparency about operation status. Specific performance targets ensure reliable user experience.*

### VI. Graceful Error Handling
ALL external API failures MUST be handled gracefully with meaningful user messages. Network timeouts MUST trigger automatic retry with exponential backoff. Authentication failures MUST automatically attempt token refresh before surfacing errors. Data corruption protection MUST validate API responses before processing.

*Rationale: Microsoft Graph API can experience intermittent failures and OneNote API has known corruption issues. Graceful error handling with automatic recovery prevents user frustration and data loss scenarios.*

### VII. MCP Resources Implementation
MUST provide URI-based resource access following hierarchical patterns: `onenote://notebooks/`, `onenote://sections/`, `onenote://pages/`. Resources MUST support dynamic content generation from live OneNote data. Resource URIs MUST follow RESTful conventions with proper encoding. All resources MUST respect authorization boundaries and selected notebook context.

*Rationale: MCP Resources enable direct content access and data discovery without tool calls. Hierarchical URI patterns provide intuitive navigation while respecting security boundaries established by the authorization system.*

### VIII. Configuration Excellence
MUST support multi-source configuration: environment variables, JSON files, command-line flags. Toolset management MUST allow selective enabling/disabling of feature groups. Configuration precedence MUST follow: CLI args > environment variables > config files > defaults. Tool descriptions MUST be customizable via configuration overrides.

*Rationale: Flexible configuration enables deployment versatility across different environments and use cases. Toolset management allows fine-grained control over server capabilities for security and performance optimization.*

### IX. Docker-First Deployment
Docker deployment MUST be the primary supported method. Multi-stage builds MUST optimize image size and security with non-root execution. Local development MUST maintain complete parity with Docker deployment. Environment variable configuration MUST work identically across both deployment modes.

*Rationale: Docker-first approach ensures consistent deployment across environments while simplifying distribution and scaling. Development parity prevents deployment surprises and reduces debugging complexity.*

## Security Requirements

All features MUST enforce defensive security principles:
- Authorization system prevents cross-notebook access through selected notebook scoping (read/write/none permissions only)
- AI agent operations MUST be restricted to explicitly authorized notebooks only
- Input validation blocks OneNote illegal characters: `?*\\/:<>|&#''%%~` and sanitizes HTML content
- OAuth tokens stored securely with automatic refresh before expiration
- NO credentials logged or exposed in error messages
- Bearer token authentication available for HTTP mode with OAuth callback bypass
- Rate limiting MUST be implemented for Graph API calls to prevent abuse
- API permissions MUST be limited to minimal required scope (Notes.ReadWrite only)

## Development Workflow

Development MUST follow this workflow:
- Feature specifications created before implementation (`/specs/` directory structure)
- Constitution compliance checked during planning phase
- Test-driven development with failing tests before implementation
- Authorization enforcement validated through dedicated test scripts (`scripts/test-security-model.sh`)
- All security-sensitive operations MUST have corresponding authorization tests
- Integration tests MUST validate against Microsoft Graph API sandbox environment
- MCP protocol compliance MUST be tested for all tools and resources
- Docker deployment scenarios MUST be tested for parity with local development
- Performance regression testing MUST validate response time requirements
- Docker and local development parity maintained through docker-compose

## Governance

This constitution supersedes all other development practices. Amendments require:
1. Documentation of change rationale and backward compatibility impact
2. Version increment following semantic versioning (MAJOR for breaking changes)
3. Update of dependent templates and documentation
4. Validation that all existing tests still pass

All code reviews MUST verify constitutional compliance. Architectural complexity MUST be justified against simplicity principles. Use `CLAUDE.md` for runtime development guidance and architectural decisions.

**Version**: 1.2.0 | **Ratified**: 2025-01-27 | **Last Amended**: 2025-01-27