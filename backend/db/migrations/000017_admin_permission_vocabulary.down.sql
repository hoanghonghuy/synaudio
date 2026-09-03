-- Roll back only authorization rows that migration 000017 actually created.
-- Permission codes and ADMIN bindings may have existed before the migration;
-- ownership metadata recorded by the up migration prevents destructive rollback.
DELETE FROM role_permissions rp
USING migration_000017_admin_permission_ownership o
WHERE rp.permission_id = o.permission_id
  AND o.admin_binding_created = TRUE;

DELETE FROM permissions p
USING migration_000017_admin_permission_ownership o
WHERE p.id = o.permission_id
  AND o.permission_created = TRUE;

DROP TABLE migration_000017_admin_permission_ownership;
