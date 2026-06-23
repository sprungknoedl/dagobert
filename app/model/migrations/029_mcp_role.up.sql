-- Bind the 'MCP' api key type to a least-privilege, read-only role. MCP keys
-- authenticate as the shared '<mcp>' principal, which may reach the stateless
-- Streamable-HTTP MCP endpoint (POST /mcp) and nothing else. The endpoint only
-- exposes read tools, so this principal can read all cases but never write.
INSERT INTO users (id, upn, name, email, role) VALUES ('<mcp>', '<mcp>', 'MCP', 'mcp@dagobert', 'MCP');
INSERT INTO policies (ptype, v0, v1) VALUES ('g', '<mcp>', 'role::MCP');
INSERT INTO policies (ptype, v0, v1, v2) VALUES ('p', 'role::MCP', '/mcp', '*');
