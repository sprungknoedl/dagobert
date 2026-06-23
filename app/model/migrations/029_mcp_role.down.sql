DELETE FROM policies WHERE ptype = 'p' AND v0 = 'role::MCP' AND v1 = '/mcp' AND v2 = '*';
DELETE FROM policies WHERE ptype = 'g' AND v0 = '<mcp>' AND v1 = 'role::MCP';
DELETE FROM users WHERE id = '<mcp>';
