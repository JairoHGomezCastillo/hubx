# Quickstart: HU-05 Verify the integrity of a shipment's chain

1. `make up`, then sign in at http://localhost:3000 as `operator@hubx.co` (demo password in memory.md).
2. Record In flight and At destination airport on a shipment (HU-03 view), then open "Verify integrity", type the
   code and press "Verify integrity": the result says the chain is intact and 2 updates were checked.
3. Tamper test (local only, never on a shared environment):

   ```sql
   -- make psql
   BEGIN;
   SET LOCAL session_replication_role = replica;
   UPDATE updates SET recorded_at = recorded_at + interval '1 second'
    WHERE sequence = 2 AND shipment_id = (SELECT id FROM shipments WHERE code = '<code>');
   COMMIT;
   ```

   Verify again: the chain is not intact and the first alteration is in update 2.
4. API: `curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/shipments/<code>/integrity`
   returns the body in `contracts/hu05-verify-integrity.md`.
5. Full proof: `make test-integration` (HU-05 feature, API scenarios and `@e2e`).
