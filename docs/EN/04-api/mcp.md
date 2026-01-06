# MCP Protocol

> Interoperability with the Model Context Protocol

## Role
- Acts as an MCP client to reuse community tools and expose gm-agent capabilities.

## Client Behavior
- Discover tools from an MCP server and map them into the tool registry.
- Apply circuit breaker when MCP host is unavailable or slow.
- Propagate auth headers/tokens according to server requirements.

## Tool Calls
- Translate MCP tool signatures into gm-agent tool schemas.
- Capture tool results as events/artifacts for replay and auditing.
