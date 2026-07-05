UPDATE policies SET v0 = "*" WHERE v1 == "/" AND v0 == "role::User";
DELETE FROM policies WHERE v1 == "/" AND v0 != "*";