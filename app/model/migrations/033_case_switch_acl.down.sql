DELETE FROM policies WHERE ptype = 'p' AND v1 = '/cases/switch' AND v2 = 'GET' AND v0 IN ('role::User', 'role::Read-Only');
